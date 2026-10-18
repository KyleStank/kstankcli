package cli

import (
	"fmt"

	"github.com/KyleStank/kstankcli/internal/process"
	"github.com/spf13/cobra"
)

type gitProfile struct {
	Name  string
	Email string
}

var gitProfiles = map[string]gitProfile{
	"personal": {Name: "Kyle Stankovich", Email: "stankawake@gmail.com"},
	"iology":   {Name: "Kyle Stankovich", Email: "kstankovich@i-ology.com"},
}

func newGitCommand(options Options) *cobra.Command {
	command := &cobra.Command{Use: "git", Short: "Run common Git workflows", Args: cobra.NoArgs}
	command.AddCommand(&cobra.Command{
		Use: "pull", Short: "Prune remote references and fast-forward the current branch", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := options.Runner.Run(cmd.Context(), process.Command{Name: "git", Args: []string{"fetch", "--prune", "--prune-tags"}}); err != nil {
				return fmt.Errorf("fetch repository: %w", err)
			}
			if err := options.Runner.Run(cmd.Context(), process.Command{Name: "git", Args: []string{"pull", "--ff-only"}}); err != nil {
				return fmt.Errorf("pull repository: %w", err)
			}
			return nil
		},
	})
	command.AddCommand(newGitConfigCommand(options))
	return command
}

func newGitConfigCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use:       "config <personal|iology>",
		Short:     "Set the repository-local Git identity",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"personal", "iology"},
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := args[0]
			profile, ok := gitProfiles[profileName]
			if !ok {
				return fmt.Errorf("unknown Git configuration %q (expected personal or iology)", profileName)
			}

			if err := options.Runner.Run(cmd.Context(), process.Command{
				Name: "git",
				Args: []string{"config", "--local", "user.name", profile.Name},
			}); err != nil {
				return fmt.Errorf("set repository-local Git user.name for %s: %w", profileName, err)
			}
			if err := options.Runner.Run(cmd.Context(), process.Command{
				Name: "git",
				Args: []string{"config", "--local", "user.email", profile.Email},
			}); err != nil {
				return fmt.Errorf("set repository-local Git user.email for %s: %w", profileName, err)
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Set repository-local Git configuration to %s\n", profileName)
			return err
		},
	}
}
