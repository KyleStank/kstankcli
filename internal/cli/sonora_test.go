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

type vpnLifecycleRunner struct {
	commands               []process.Command
	running                bool
	disconnectFailure      error
	stopFailure            error
	stopLeavesRunning      bool
	terminateFailure       error
	terminateLeavesRunning bool
}

func (r *vpnLifecycleRunner) Run(_ context.Context, command process.Command) error {
	r.commands = append(r.commands, command)
	if reflect.DeepEqual(command.Args, []string{"ipsec", "down", "sonoraquestv10"}) && r.disconnectFailure != nil {
		return r.disconnectFailure
	}
	if reflect.DeepEqual(command.Args, []string{"systemctl", "start", strongSwanService}) {
		r.running = true
	}
	if reflect.DeepEqual(command.Args, []string{"systemctl", "stop", strongSwanService}) {
		if r.stopFailure != nil {
			return r.stopFailure
		}
		if !r.stopLeavesRunning {
			r.running = false
		}
	}
	if reflect.DeepEqual(command.Args, terminateCharonArgs()) {
		if r.terminateFailure != nil {
			return r.terminateFailure
		}
		if !r.terminateLeavesRunning {
			r.running = false
		}
	}
	if reflect.DeepEqual(command.Args, []string{"ipsec", "listplugins"}) && !r.running {
		return errors.New("IPsec is not running")
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
				{Name: "sudo", Args: []string{"systemctl", "start", strongSwanService}},
				{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
				{Name: "sudo", Args: []string{"ipsec", "up", "sonoraquestv10"}},
			},
		},
		{
			name:   "stop",
			action: "stop",
			want: []process.Command{
				{Name: "sudo", Args: []string{"ipsec", "down", "sonoraquestv10"}},
				{Name: "sudo", Args: []string{"systemctl", "stop", strongSwanService}},
				{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
			},
		},
		{
			name:   "restart",
			action: "restart",
			want: []process.Command{
				{Name: "sudo", Args: []string{"ipsec", "down", "sonoraquestv10"}},
				{Name: "sudo", Args: []string{"systemctl", "stop", strongSwanService}},
				{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
				{Name: "sudo", Args: []string{"systemctl", "start", strongSwanService}},
				{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
				{Name: "sudo", Args: []string{"ipsec", "up", "sonoraquestv10"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options, _, _ := testOptions(t)
			runner := &vpnLifecycleRunner{running: true}
			options.Runner = runner
			if err := execute(t, options, "sonora", "vpn", test.action); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(runner.commands, test.want) {
				t.Fatalf("commands = %#v, want %#v", runner.commands, test.want)
			}
		})
	}
}

func TestSonoraVPNStopStillStopsServiceWhenDisconnectFails(t *testing.T) {
	options, _, _ := testOptions(t)
	runner := &vpnLifecycleRunner{running: true, disconnectFailure: errors.New("connection is not active")}
	options.Runner = runner

	if err := execute(t, options, "sonora", "vpn", "stop"); err != nil {
		t.Fatalf("stop returned an error after confirming service shutdown: %v", err)
	}
	want := []process.Command{
		{Name: "sudo", Args: []string{"ipsec", "down", "sonoraquestv10"}},
		{Name: "sudo", Args: []string{"systemctl", "stop", strongSwanService}},
		{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}
}

func TestSonoraVPNStopReportsDisconnectAndServiceFailures(t *testing.T) {
	options, _, _ := testOptions(t)
	runner := &vpnLifecycleRunner{
		running:           true,
		disconnectFailure: errors.New("connection failure"),
		stopFailure:       errors.New("service failure"),
	}
	options.Runner = runner

	err := execute(t, options, "sonora", "vpn", "stop")
	if err == nil {
		t.Fatal("expected stop error")
	}
	for _, message := range []string{"disconnect Sonora VPN", "stop strongSwan service"} {
		if !strings.Contains(err.Error(), message) {
			t.Fatalf("error %q does not contain %q", err, message)
		}
	}
	if got, want := len(runner.commands), 2; got != want {
		t.Fatalf("command count = %d, want %d", got, want)
	}
}

func TestSonoraVPNStopTerminatesLingeringCharon(t *testing.T) {
	options, _, _ := testOptions(t)
	runner := &vpnLifecycleRunner{running: true, stopLeavesRunning: true}
	options.Runner = runner

	if err := execute(t, options, "sonora", "vpn", "stop"); err != nil {
		t.Fatal(err)
	}
	want := []process.Command{
		{Name: "sudo", Args: []string{"ipsec", "down", "sonoraquestv10"}},
		{Name: "sudo", Args: []string{"systemctl", "stop", strongSwanService}},
		{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
		{Name: "sudo", Args: terminateCharonArgs()},
		{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}
}

func terminateCharonArgs() []string {
	return []string{
		"systemd-run",
		"--quiet",
		"--wait",
		"--collect",
		"--service-type=exec",
		"/usr/bin/pkill",
		"--signal",
		"TERM",
		"--exact",
		"charon",
	}
}

func TestSonoraVPNStopReportsLingeringCharonTerminationFailure(t *testing.T) {
	options, _, _ := testOptions(t)
	runner := &vpnLifecycleRunner{
		running:           true,
		stopLeavesRunning: true,
		terminateFailure:  errors.New("permission denied"),
	}
	options.Runner = runner

	err := execute(t, options, "sonora", "vpn", "stop")
	if err == nil || !strings.Contains(err.Error(), "terminate lingering IPsec daemon") {
		t.Fatalf("expected daemon termination error, got %v", err)
	}
	wantLast := process.Command{Name: "sudo", Args: []string{"ipsec", "listplugins"}, Quiet: true}
	if got := runner.commands[len(runner.commands)-1]; !reflect.DeepEqual(got, wantLast) {
		t.Fatalf("last command = %#v, want %#v", got, wantLast)
	}
}

func TestSonoraVPNRestartDoesNotStartWhileIPsecIsStillRunning(t *testing.T) {
	options, _, _ := testOptions(t)
	runner := &vpnLifecycleRunner{running: true, stopLeavesRunning: true, terminateLeavesRunning: true}
	options.Runner = runner
	waits := 0
	options.Wait = func(context.Context, time.Duration) error {
		waits++
		return nil
	}

	err := execute(t, options, "sonora", "vpn", "restart")
	if err == nil || !strings.Contains(err.Error(), "still running 30s after SIGTERM") {
		t.Fatalf("expected shutdown timeout, got %v", err)
	}
	if waits != ipsecShutdownAttempts-1 {
		t.Fatalf("wait count = %d, want %d", waits, ipsecShutdownAttempts-1)
	}
	if got := runner.commands[len(runner.commands)-1].Args; !reflect.DeepEqual(got, []string{"ipsec", "listplugins"}) {
		t.Fatalf("last command = %#v, want shutdown readiness check", got)
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
	if err == nil || !strings.Contains(err.Error(), "start strongSwan service") {
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
		if duration != ipsecPollingInterval {
			t.Fatalf("wait duration = %s, want %s", duration, ipsecPollingInterval)
		}
		waits++
		return nil
	}

	if err := execute(t, options, "sonora", "vpn", "start"); err != nil {
		t.Fatal(err)
	}
	want := []process.Command{
		{Name: "sudo", Args: []string{"systemctl", "start", strongSwanService}},
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
