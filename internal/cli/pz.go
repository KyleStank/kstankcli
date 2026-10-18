package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/KyleStank/kstankcli/internal/config"
	"github.com/KyleStank/kstankcli/internal/process"
	"github.com/KyleStank/kstankcli/internal/pz"
	"github.com/spf13/cobra"
)

func newPZCommand(options Options) *cobra.Command {
	command := &cobra.Command{Use: "pz", Short: "Manage Project Zomboid workflows", Args: cobra.NoArgs}
	command.AddCommand(newPZPathCommand(options))
	command.AddCommand(newPZOpenCommand(options))
	command.AddCommand(newPZRunCommand(options))
	command.AddCommand(newPZCopyCommand(options))
	return command
}

func newPZPathCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use: "path <home|install>", Short: "Print a configured Project Zomboid path", Args: cobra.ExactArgs(1),
		ValidArgs: []string{"home", "install"},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := loadConfig(options)
			if err != nil {
				return err
			}
			path, err := pzConfiguredPath(cfg, args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), path)
			return err
		},
	}
}

func newPZOpenCommand(options Options) *cobra.Command {
	return &cobra.Command{
		Use: "open <home|install>", Short: "Open a configured Project Zomboid directory", Args: cobra.ExactArgs(1),
		ValidArgs: []string{"home", "install"},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := loadConfig(options)
			if err != nil {
				return err
			}
			path, err := pzConfiguredPath(cfg, args[0])
			if err != nil {
				return err
			}
			if err := requireDirectory(path); err != nil {
				return err
			}
			var invocation process.Command
			switch options.GOOS {
			case "linux":
				invocation = process.Command{Name: "xdg-open", Args: []string{path}}
			case "windows":
				invocation = process.Command{Name: "explorer.exe", Args: []string{path}}
			default:
				return fmt.Errorf("pz open is unsupported on %s (supported: linux, windows)", options.GOOS)
			}
			return options.Runner.Run(cmd.Context(), invocation)
		},
	}
}

func newPZRunCommand(options Options) *cobra.Command {
	var debug, noSteam bool
	command := &cobra.Command{
		Use: "run", Short: "Launch Project Zomboid from its configured installation", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, cfg, err := loadConfig(options)
			if err != nil {
				return err
			}
			install, err := requirePZSetting(cfg.ProjectZomboid.InstallDirectory, "project-zomboid.install-directory")
			if err != nil {
				return err
			}
			if err := requireDirectory(install); err != nil {
				return err
			}
			executable := cfg.ProjectZomboid.Executable
			if executable == "" {
				if options.GOOS != "linux" {
					return errors.New("Project Zomboid executable is not configured; run: kstankcli config set project-zomboid.executable <path-or-name>")
				}
				executable = filepath.Join(install, "projectzomboid.sh")
			} else if !filepath.IsAbs(executable) {
				executable = filepath.Join(install, executable)
			}
			var arguments []string
			if debug {
				arguments = append(arguments, "-debug")
			}
			if noSteam {
				arguments = append(arguments, "-nosteam")
			}
			return options.Runner.Run(cmd.Context(), process.Command{Name: executable, Args: arguments, Dir: install})
		},
	}
	command.Flags().BoolVar(&debug, "debug", false, "enable Project Zomboid debug mode")
	command.Flags().BoolVar(&noSteam, "no-steam", false, "launch without Steam integration")
	return command
}

func newPZCopyCommand(options Options) *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use: "copy-local-mod <mod-name>", Short: "Copy a Workshop mod into the local mods directory", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := loadConfig(options)
			if err != nil {
				return err
			}
			home, err := requirePZSetting(cfg.ProjectZomboid.HomeDirectory, "project-zomboid.home-directory")
			if err != nil {
				return err
			}
			if err := pz.CopyLocalMod(home, args[0], force); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Copied %s into %s\n", args[0], filepath.Join(home, "mods"))
			return err
		},
	}
	command.Flags().BoolVar(&force, "force", false, "replace an existing local mod")
	return command
}

func pzConfiguredPath(cfg config.Config, name string) (string, error) {
	switch name {
	case "home":
		return requirePZSetting(cfg.ProjectZomboid.HomeDirectory, "project-zomboid.home-directory")
	case "install":
		return requirePZSetting(cfg.ProjectZomboid.InstallDirectory, "project-zomboid.install-directory")
	default:
		return "", fmt.Errorf("unknown Project Zomboid path %q (expected home or install)", name)
	}
}

func requirePZSetting(value, key string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("Project Zomboid setting %s is empty; run: kstankcli config set %s <path>", key, key)
	}
	return value, nil
}

func requireDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}
	return nil
}
