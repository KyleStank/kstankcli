package cli

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/KyleStank/kstankcli/internal/process"
)

type unavailableIPsecRunner struct {
	commands []process.Command
}

func (r *unavailableIPsecRunner) Run(_ context.Context, command process.Command) error {
	r.commands = append(r.commands, command)
	if reflect.DeepEqual(command.Args, []string{"ipsec", "listplugins"}) {
		return errors.New("IPsec is not ready")
	}
	return nil
}

func TestSonoraVPNCommands(t *testing.T) {
	tests := []struct {
		name   string
		action string
		want   []process.Command
	}{
		{
			name:   "start",
			action: "start",
			want: []process.Command{
				{Name: "sudo", Args: []string{"ipsec", "start"}},
				{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
				{Name: "sudo", Args: []string{"ipsec", "up", "sonoraquestv10"}},
			},
		},
		{
			name:   "stop",
			action: "stop",
			want: []process.Command{
				{Name: "sudo", Args: []string{"ipsec", "stop"}},
			},
		},
		{
			name:   "restart",
			action: "restart",
			want: []process.Command{
				{Name: "sudo", Args: []string{"ipsec", "stop"}},
				{Name: "sudo", Args: []string{"ipsec", "start"}},
				{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
				{Name: "sudo", Args: []string{"ipsec", "up", "sonoraquestv10"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options, runner, _ := testOptions(t)
			if err := execute(t, options, "sonora", "vpn", test.action); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(runner.commands, test.want) {
				t.Fatalf("commands = %#v, want %#v", runner.commands, test.want)
			}
		})
	}
}

func TestSonoraVPNRejectsWindows(t *testing.T) {
	options, runner, _ := testOptions(t)
	options.GOOS = "windows"

	err := execute(t, options, "sonora", "vpn", "start")
	if err == nil || !strings.Contains(err.Error(), "supported: linux") {
		t.Fatalf("expected Linux-only error, got %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("executed commands on Windows: %#v", runner.commands)
	}
}

func TestSonoraVPNStopsAfterStartFailure(t *testing.T) {
	options, runner, _ := testOptions(t)
	runner.failAt = 1
	runner.failure = errors.New("sudo failed")

	err := execute(t, options, "sonora", "vpn", "start")
	if err == nil || !strings.Contains(err.Error(), "start IPsec service") {
		t.Fatalf("expected start error, got %v", err)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("executed %d commands, want 1", len(runner.commands))
	}
}

func TestSonoraVPNRetriesReadinessBeforeConnecting(t *testing.T) {
	options, runner, _ := testOptions(t)
	runner.failAt = 2
	runner.failure = errors.New("IPsec is not ready")
	waits := 0
	options.Wait = func(_ context.Context, duration time.Duration) error {
		if duration != ipsecReadinessInterval {
			t.Fatalf("wait duration = %s, want %s", duration, ipsecReadinessInterval)
		}
		waits++
		return nil
	}

	if err := execute(t, options, "sonora", "vpn", "start"); err != nil {
		t.Fatal(err)
	}
	want := []process.Command{
		{Name: "sudo", Args: []string{"ipsec", "start"}},
		{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
		{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
		{Name: "sudo", Args: []string{"ipsec", "up", "sonoraquestv10"}},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}
	if waits != 1 {
		t.Fatalf("wait count = %d, want 1", waits)
	}
}

func TestSonoraVPNReadinessTimesOutWithoutConnecting(t *testing.T) {
	options, _, _ := testOptions(t)
	runner := &unavailableIPsecRunner{}
	options.Runner = runner
	waits := 0
	options.Wait = func(context.Context, time.Duration) error {
		waits++
		return nil
	}

	err := execute(t, options, "sonora", "vpn", "start")
	if err == nil || !strings.Contains(err.Error(), "did not become ready within 5s") {
		t.Fatalf("expected readiness timeout, got %v", err)
	}
	if got, want := len(runner.commands), ipsecReadinessAttempts+1; got != want {
		t.Fatalf("command count = %d, want %d", got, want)
	}
	if waits != ipsecReadinessAttempts-1 {
		t.Fatalf("wait count = %d, want %d", waits, ipsecReadinessAttempts-1)
	}
	for _, command := range runner.commands[1:] {
		if !reflect.DeepEqual(command.Args, []string{"ipsec", "listplugins"}) {
			t.Fatalf("unexpected command before readiness: %#v", command)
		}
	}
}
