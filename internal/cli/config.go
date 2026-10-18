package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
			return options.Runner.Run(cmd.Context(), process.Command{Name: cfg.Editor, Args: []string{path}})
		},
	}
}

func newConfigValidateCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use: "validate", Short: "Validate configuration syntax, version, and configured paths", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, cfg, err := loadConfig(options)
			if err != nil {
				return err
			}
			for _, candidate := range []struct{ name, path string }{
				{"project-zomboid.home-directory", cfg.ProjectZomboid.HomeDirectory},
				{"project-zomboid.install-directory", cfg.ProjectZomboid.InstallDirectory},
			} {
				if candidate.path == "" {
					continue
				}
				if _, err := os.Stat(candidate.path); err != nil {
					if _, printErr := fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s %q is unavailable: %v\n", candidate.name, candidate.path, err); printErr != nil {
						return printErr
					}
				}
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
				return configKeys(), cobra.ShellCompDirectiveNoFileComp
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
			if err := setConfigValue(&cfg, args[0], args[1]); err != nil {
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

func configKeys() []string {
	return []string{"editor", "project-zomboid.home-directory", "project-zomboid.install-directory", "project-zomboid.executable"}
}

func setConfigValue(cfg *config.Config, key, value string) error {
	switch key {
	case "editor":
		if value == "" {
			return errors.New("editor must not be empty")
		}
		cfg.Editor = value
	case "project-zomboid.home-directory":
		if value == "" {
			cfg.ProjectZomboid.HomeDirectory = ""
		} else {
			cfg.ProjectZomboid.HomeDirectory = filepath.Clean(value)
		}
	case "project-zomboid.install-directory":
		if value == "" {
			cfg.ProjectZomboid.InstallDirectory = ""
		} else {
			cfg.ProjectZomboid.InstallDirectory = filepath.Clean(value)
		}
	case "project-zomboid.executable":
		cfg.ProjectZomboid.Executable = value
	default:
		return fmt.Errorf("unknown config key %q (valid keys: %v)", key, configKeys())
	}
	return nil
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
