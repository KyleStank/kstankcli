package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const RootEnvironmentVariable = "KSTANKCLI_ROOT"

type RootResolver struct {
	LookupEnv   func(string) (string, bool)
	UserHomeDir func() (string, error)
}

func DefaultRootResolver() RootResolver {
	return RootResolver{LookupEnv: os.LookupEnv, UserHomeDir: os.UserHomeDir}
}

func ResolveRoot() (string, error) {
	return DefaultRootResolver().Resolve()
}

func (r RootResolver) Resolve() (string, error) {
	lookupEnv := r.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	homeDir := r.UserHomeDir
	if homeDir == nil {
		homeDir = os.UserHomeDir
	}

	value, set := lookupEnv(RootEnvironmentVariable)
	if set && strings.TrimSpace(value) != "" {
		if value == "~" || strings.HasPrefix(value, "~/") || strings.HasPrefix(value, `~\`) {
			home, err := homeDir()
			if err != nil {
				return "", fmt.Errorf("expand %s: determine user home directory: %w", RootEnvironmentVariable, err)
			}
			if value == "~" {
				value = home
			} else {
				value = filepath.Join(home, value[2:])
			}
		}
		return filepath.Clean(value), nil
	}

	home, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("determine user home directory: %w", err)
	}
	return filepath.Join(home, ".kstankcli"), nil
}

func Path(root string) string {
	return filepath.Join(root, "config.json")
}
