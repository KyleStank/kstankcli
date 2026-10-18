package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	Name  string
	Args  []string
	Dir   string
	Env   []string
	Quiet bool
}

type Runner interface {
	Run(context.Context, Command) error
}

type OSRunner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func NewOSRunner() *OSRunner {
	return &OSRunner{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
}

func (r *OSRunner) Run(ctx context.Context, command Command) error {
	if command.Name == "" {
		return fmt.Errorf("executable must not be empty")
	}
	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Dir = command.Dir
	cmd.Stdin = r.Stdin
	if command.Quiet {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		cmd.Stdout = r.Stdout
		cmd.Stderr = r.Stderr
	}
	cmd.Env = mergeEnvironment(os.Environ(), command.Env)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", command.Name, err)
	}
	return nil
}

func mergeEnvironment(base, overrides []string) []string {
	if len(overrides) == 0 {
		return base
	}
	keys := make(map[string]struct{}, len(overrides))
	for _, item := range overrides {
		key, _, _ := strings.Cut(item, "=")
		keys[strings.ToUpper(key)] = struct{}{}
	}
	merged := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, _ := strings.Cut(item, "=")
		if _, overridden := keys[strings.ToUpper(key)]; !overridden {
			merged = append(merged, item)
		}
	}
	return append(merged, overrides...)
}
