package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type commandUsageError struct {
	cause error
	usage string
}

func (e commandUsageError) Error() string {
	return fmt.Sprintf("%s\n\n%s", e.cause, e.usage)
}

func (e commandUsageError) Unwrap() error {
	return e.cause
}

func configureCommandErrorHandling(root *cobra.Command) {
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return newCommandUsageError(cmd, err)
	})
	configureCommandErrors(root)
}

func configureCommandErrors(command *cobra.Command) {
	if command.Args != nil {
		validateArgs := command.Args
		command.Args = func(cmd *cobra.Command, args []string) error {
			if err := validateArgs(cmd, args); err != nil {
				return newCommandUsageError(cmd, err)
			}
			if len(cmd.ValidArgs) > 0 {
				if err := cobra.OnlyValidArgs(cmd, args); err != nil {
					return newCommandUsageError(cmd, err)
				}
			}
			return nil
		}
	}

	wrapFlagValidation(command)

	if !command.Runnable() && command.HasSubCommands() {
		command.RunE = func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		}
	}

	for _, child := range command.Commands() {
		configureCommandErrors(child)
	}
}

func wrapFlagValidation(command *cobra.Command) {
	preRunE := command.PreRunE
	preRun := command.PreRun
	command.PreRun = nil
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		if err := cmd.ValidateRequiredFlags(); err != nil {
			return newCommandUsageError(cmd, err)
		}
		if err := cmd.ValidateFlagGroups(); err != nil {
			return newCommandUsageError(cmd, err)
		}
		if preRunE != nil {
			return preRunE(cmd, args)
		}
		if preRun != nil {
			preRun(cmd, args)
		}
		return nil
	}
}

func newCommandUsageError(command *cobra.Command, cause error) error {
	return commandUsageError{
		cause: cause,
		usage: strings.TrimSpace(command.UsageString()),
	}
}
