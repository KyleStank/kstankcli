package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/KyleStank/kstankcli/internal/config"
	"github.com/KyleStank/kstankcli/internal/process"
)

type recordingRunner struct {
	commands []process.Command
	failAt   int
	failure  error
}

func (r *recordingRunner) Run(_ context.Context, command process.Command) error {
	r.commands = append(r.commands, command)
	if r.failAt > 0 && len(r.commands) == r.failAt {
		return r.failure
	}
	return nil
}

func testOptions(t *testing.T) (Options, *recordingRunner, *bytes.Buffer) {
	t.Helper()
	root := t.TempDir()
	runner := &recordingRunner{}
	output := &bytes.Buffer{}
	return Options{
		RootResolver: config.RootResolver{
			LookupEnv:   func(string) (string, bool) { return root, true },
			UserHomeDir: func() (string, error) { return "", nil },
		},
		Runner: runner,
		Wait: func(context.Context, time.Duration) error {
			return nil
		},
		GOOS: "linux", Stdout: output, Stderr: output,
	}, runner, output
}

func execute(t *testing.T, options Options, args ...string) error {
	t.Helper()
	command := NewRootCommand(options)
	command.SetArgs(args)
	return command.Execute()
}

func TestConfigAndPZRun(t *testing.T) {
	options, runner, _ := testOptions(t)
	install := t.TempDir()
	executable := filepath.Join(install, "projectzomboid.sh")
	if err := os.WriteFile(executable, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := execute(t, options, "config", "set", "projectZomboid.installDirectory", install); err != nil {
		t.Fatal(err)
	}
	if err := execute(t, options, "config", "set", "projectZomboid.executable", executable); err != nil {
		t.Fatal(err)
	}
	if err := execute(t, options, "pz", "run", "--debug", "--no-steam"); err != nil {
		t.Fatal(err)
	}
	want := process.Command{Name: executable, Args: []string{"-debug", "-nosteam"}, Dir: install}
	if len(runner.commands) != 1 || !reflect.DeepEqual(runner.commands[0], want) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}
}

func TestConfigValidateChecksSchemaPathFields(t *testing.T) {
	options, _, output := testOptions(t)
	missing := filepath.Join(t.TempDir(), "missing")
	root, err := options.RootResolver.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.ProjectZomboid.HomeDirectory = missing
	if err := config.Write(config.Path(root), cfg); err != nil {
		t.Fatal(err)
	}

	output.Reset()
	err = execute(t, options, "config", "validate")
	if err == nil {
		t.Fatal("expected validation error")
	}
	got := err.Error()
	if !bytes.Contains([]byte(got), []byte(`projectZomboid.homeDirectory (exists): "`+missing+`" does not exist`)) {
		t.Fatalf("error %q does not contain path validation failure", got)
	}
	for _, key := range []string{"projectZomboid.installDirectory", "projectZomboid.executable"} {
		if !bytes.Contains([]byte(got), []byte(key+" (required)")) {
			t.Errorf("error %q does not contain required failure for %s", got, key)
		}
	}
}

func TestConfigEditRejectsUnsupportedEditorWithoutBlockingLoad(t *testing.T) {
	options, runner, _ := testOptions(t)
	root, err := options.RootResolver.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Editor = "vim"
	if err := config.Write(config.Path(root), cfg); err != nil {
		t.Fatal(err)
	}

	err = execute(t, options, "config", "edit")
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("editor (oneOf)")) {
		t.Fatalf("expected editor validation failure, got %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("invalid editor executed commands: %#v", runner.commands)
	}
}

func TestGitPullAndConfigInvocations(t *testing.T) {
	options, runner, _ := testOptions(t)
	if err := execute(t, options, "git", "pull"); err != nil {
		t.Fatal(err)
	}
	if err := execute(t, options, "git", "config", "personal"); err != nil {
		t.Fatal(err)
	}
	want := []process.Command{
		{Name: "git", Args: []string{"fetch", "--prune", "--prune-tags"}},
		{Name: "git", Args: []string{"pull", "--ff-only"}},
		{Name: "git", Args: []string{"config", "--local", "user.name", "Kyle Stankovich"}},
		{Name: "git", Args: []string{"config", "--local", "user.email", "stankawake@gmail.com"}},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}
}

func TestGitConfigIologyAndUnknownProfile(t *testing.T) {
	options, runner, _ := testOptions(t)
	if err := execute(t, options, "git", "config", "iology"); err != nil {
		t.Fatal(err)
	}
	want := []process.Command{
		{Name: "git", Args: []string{"config", "--local", "user.name", "Kyle Stankovich"}},
		{Name: "git", Args: []string{"config", "--local", "user.email", "kstankovich@i-ology.com"}},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}

	if err := execute(t, options, "git", "config", "wgu"); err == nil {
		t.Fatal("expected unknown-profile error")
	}
	if len(runner.commands) != 2 {
		t.Fatalf("unknown profile executed commands: %#v", runner.commands[2:])
	}
}

func TestGitConfigStopsWhenSettingNameFails(t *testing.T) {
	options, runner, _ := testOptions(t)
	runner.failAt = 1
	runner.failure = errors.New("git failed")

	err := execute(t, options, "git", "config", "personal")
	if err == nil {
		t.Fatal("expected Git configuration error")
	}
	if len(runner.commands) != 1 {
		t.Fatalf("executed %d commands, want 1", len(runner.commands))
	}
}

func TestRootRejectsArguments(t *testing.T) {
	options, _, _ := testOptions(t)
	if err := execute(t, options, "unexpected"); err == nil {
		t.Fatal("expected argument error")
	}
}

func TestVersionIncludesBuildMode(t *testing.T) {
	options, _, output := testOptions(t)
	oldVersion, oldMode := Version, BuildMode
	Version, BuildMode = "1.2.3", "prod"
	t.Cleanup(func() { Version, BuildMode = oldVersion, oldMode })

	if err := execute(t, options, "--version"); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "kstankcli version 1.2.3 (prod)\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPZRequiresConfiguredPath(t *testing.T) {
	options, _, _ := testOptions(t)
	if err := execute(t, options, "config", "init"); err != nil {
		t.Fatal(err)
	}
	err := execute(t, options, "pz", "path", "home")
	if err == nil {
		t.Fatal("expected missing-setting error")
	}
	want := "kstankcli config set projectZomboid.homeDirectory <absolute-path>"
	if !bytes.Contains([]byte(err.Error()), []byte(want)) {
		t.Fatalf("error %q does not contain %q", err, want)
	}
}

func TestWindowsPZRunRequiresExecutable(t *testing.T) {
	options, runner, _ := testOptions(t)
	options.GOOS = "windows"
	install := t.TempDir()
	if err := execute(t, options, "config", "set", "projectZomboid.installDirectory", install); err != nil {
		t.Fatal(err)
	}
	err := execute(t, options, "pz", "run")
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("projectZomboid.executable")) {
		t.Fatalf("expected actionable executable error, got %v", err)
	}
	executable := filepath.Join(t.TempDir(), "ProjectZomboid64.exe")
	if err := os.WriteFile(executable, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := execute(t, options, "config", "set", "projectZomboid.executable", executable); err != nil {
		t.Fatal(err)
	}
	if err := execute(t, options, "pz", "run"); err != nil {
		t.Fatal(err)
	}
	want := process.Command{Name: executable, Dir: install}
	if len(runner.commands) != 1 || !reflect.DeepEqual(runner.commands[0], want) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}
}

func TestUnsupportedOperatingSystem(t *testing.T) {
	options, _, _ := testOptions(t)
	options.GOOS = "darwin"
	err := execute(t, options, "config", "path")
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("unsupported")) {
		t.Fatalf("expected unsupported-platform error, got %v", err)
	}
}
