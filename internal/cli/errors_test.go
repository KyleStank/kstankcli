package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSyntaxErrorsIncludeRelevantUsage(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantUsage string
		wantText  string
	}{
		{
			name:      "missing positional argument",
			args:      []string{"pz", "open"},
			wantUsage: "Usage:\n  kstankcli pz open <home|install> [flags]",
			wantText:  "accepts 1 arg(s), received 0",
		},
		{
			name:      "missing one of two positional arguments",
			args:      []string{"config", "set", "editor"},
			wantUsage: "Usage:\n  kstankcli config set <key> <value> [flags]",
			wantText:  "accepts 2 arg(s), received 1",
		},
		{
			name:      "invalid constrained argument",
			args:      []string{"git", "config", "unknown"},
			wantUsage: "Usage:\n  kstankcli git config <personal|iology> [flags]",
			wantText:  `invalid argument "unknown"`,
		},
		{
			name:      "unknown flag",
			args:      []string{"pz", "run", "--unknown"},
			wantUsage: "Usage:\n  kstankcli pz run [flags]",
			wantText:  "unknown flag: --unknown",
		},
		{
			name:      "unknown subcommand",
			args:      []string{"pz", "unknown"},
			wantUsage: "Usage:\n  kstankcli pz [flags]\n  kstankcli pz [command]",
			wantText:  `unknown command "unknown"`,
		},
		{
			name:      "generated completion command",
			args:      []string{"completion", "bash", "unexpected"},
			wantUsage: "Usage:\n  kstankcli completion bash",
			wantText:  `unknown command "unexpected"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options, runner, _ := testOptions(t)
			err := execute(t, options, test.args...)
			if err == nil {
				t.Fatal("expected syntax error")
			}
			if !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("error %q does not contain %q", err, test.wantText)
			}
			if !strings.Contains(err.Error(), test.wantUsage) {
				t.Fatalf("error %q does not contain %q", err, test.wantUsage)
			}
			if len(runner.commands) != 0 {
				t.Fatalf("syntax error executed commands: %#v", runner.commands)
			}
		})
	}
}

func TestRequiredFlagErrorsReceiveUsageAutomatically(t *testing.T) {
	root := &cobra.Command{Use: "example", SilenceErrors: true, SilenceUsage: true}
	child := &cobra.Command{Use: "run", RunE: func(*cobra.Command, []string) error { return nil }}
	child.Flags().String("required", "", "required test value")
	if err := child.MarkFlagRequired("required"); err != nil {
		t.Fatal(err)
	}
	root.AddCommand(child)
	configureCommandErrorHandling(root)
	root.SetArgs([]string{"run"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected required-flag error")
	}
	if !strings.Contains(err.Error(), `required flag(s) "required" not set`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "Usage:\n  example run [flags]") {
		t.Fatalf("required-flag error lacks usage: %v", err)
	}
}

func TestRuntimeErrorsDoNotIncludeUsage(t *testing.T) {
	options, runner, _ := testOptions(t)
	runner.failAt = 1
	runner.failure = errors.New("git failed")

	err := execute(t, options, "git", "pull")
	if err == nil {
		t.Fatal("expected runtime error")
	}
	if strings.Contains(err.Error(), "Usage:") {
		t.Fatalf("runtime error unexpectedly contains usage: %v", err)
	}
}
