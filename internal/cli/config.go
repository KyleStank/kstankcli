package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/KyleStank/kstankcli/internal/config"
	"github.com/KyleStank/kstankcli/internal/process"
	"github.com/spf13/cobra"
)

func newConfigCommand(options Options) *cobra.Command {
	command := &cobra.Command{Use: "config", Short: "Manage local kstankcli configuration", Args: cobra.NoArgs}
	command.AddCommand(newConfigPathCommand(options))
	command.AddCommand(newConfigInitCommand(options))
	command.AddCommand(newConfigShowCommand(options))
	command.AddCommand(newConfigEditCommand(options))
	command.AddCommand(newConfigValidateCommand(options))
	command.AddCommand(newConfigSetCommand(options))
	return command
}

func newConfigPathCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use: "path", Short: "Print the resolved configuration file path", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := configPath(options)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), path)
			return err
		},
	}
}

func newConfigInitCommand(options Options) *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use: "init", Short: "Create a default configuration file", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := configPath(options)
			if err != nil {
				return err
			}
			if err := config.Init(path, force); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Created", path)
			return err
		},
	}
	command.Flags().BoolVar(&force, "force", false, "replace an existing configuration file")
	return command
}

func newConfigShowCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use: "show", Short: "Print the effective configuration", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, cfg, err := loadConfig(options)
			if err != nil {
				return err
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(cfg)
		},
	}
}

func newConfigEditCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use: "edit", Short: "Open the configuration file in its configured editor", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := configPath(options)
			if err != nil {
				return err
			}
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				if err := config.Init(path, false); err != nil {
					return err
				}
			} else if err != nil {
				return fmt.Errorf("inspect config %q: %w", path, err)
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			if err := config.ValidateFields(cfg, "editor"); err != nil {
				return fmt.Errorf("validate editor configuration: %w", err)
			}
			return options.Runner.Run(cmd.Context(), process.Command{Name: cfg.Editor, Args: []string{path}})
		},
	}
}

func newConfigValidateCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use: "validate", Short: "Validate configuration schema and field values", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, cfg, err := loadConfig(options)
			if err != nil {
				return err
			}
			if err := config.ValidateAllFields(cfg); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Valid:", path)
			return err
		},
	}
}

func newConfigSetCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use: "set <key> <value>", Short: "Set a typed configuration value", Args: cobra.ExactArgs(2),
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return config.SettableKeys(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := configPath(options)
			if err != nil {
				return err
			}
			cfg := config.Default()
			if _, err := os.Stat(path); err == nil {
				cfg, err = config.Load(path)
				if err != nil {
					return err
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("inspect config %q: %w", path, err)
			}
			if err := config.SetValue(&cfg, args[0], args[1]); err != nil {
				return err
			}
			if err := config.Write(path, cfg); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated %s in %s\n", args[0], path)
			return err
		},
	}
	return command
}

func loadConfig(options Options) (string, config.Config, error) {
	path, err := configPath(options)
	if err != nil {
		return "", config.Config{}, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return path, config.Config{}, err
	}
	return path, cfg, nil
}
