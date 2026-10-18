package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/KyleStank/kstankcli/internal/config"
	"github.com/KyleStank/kstankcli/internal/process"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildMode = "dev"
)

type Options struct {
	RootResolver config.RootResolver
	Runner       process.Runner
	Wait         func(context.Context, time.Duration) error
	GOOS         string
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
}

func DefaultOptions() Options {
	return Options{
		RootResolver: config.DefaultRootResolver(),
		Runner:       process.NewOSRunner(),
		Wait:         waitWithContext,
		GOOS:         runtime.GOOS,
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	}
}

func NewRootCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:     "kstankcli",
		Short:   "Cross-platform commands for my development workflows.",
		Long:    "kstankcli combines development workflows across multiple operating systems behind one command line interface.",
		Version: Version,
		Args:    cobra.NoArgs,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			// TODO: Add Mac eventually
			if options.GOOS != "linux" && options.GOOS != "windows" {
				return fmt.Errorf("operating system %q is unsupported (supported: linux, windows)", options.GOOS)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	command.SetVersionTemplate("kstankcli version {{.Version}} (" + BuildMode + ")\n")

	if options.Stdin != nil {
		command.SetIn(options.Stdin)
	}
	if options.Stdout != nil {
		command.SetOut(options.Stdout)
	}
	if options.Stderr != nil {
		command.SetErr(options.Stderr)
	}

	command.AddCommand(newConfigCommand(options))
	command.AddCommand(newGitCommand(options))
	command.AddCommand(newPZCommand(options))
	command.AddCommand(newSonoraCommand(options))

	command.InitDefaultHelpCmd()
	command.InitDefaultCompletionCmd()
	configureCommandErrorHandling(command)

	return command
}

func waitWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func configPath(options Options) (string, error) {
	root, err := options.RootResolver.Resolve()
	if err != nil {
		return "", err
	}
	return config.Path(root), nil
}
