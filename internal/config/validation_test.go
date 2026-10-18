package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAllFieldsAcceptsValidConfiguration(t *testing.T) {
	cfg := validTestConfig(t)
	if err := ValidateAllFields(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAllFieldsAggregatesFailures(t *testing.T) {
	cfg := Default()
	cfg.Editor = "vim"
	cfg.ProjectZomboid.InstallDirectory = "relative"
	cfg.ProjectZomboid.Executable = t.TempDir()

	err := ValidateAllFields(cfg)
	if err == nil {
		t.Fatal("expected validation failure")
	}
	message := err.Error()
	for _, expected := range []string{
		"editor (oneOf)",
		"projectZomboid.homeDirectory (required)",
		"projectZomboid.installDirectory (absolute)",
		"projectZomboid.executable (file)",
	} {
		if !strings.Contains(message, expected) {
			t.Errorf("error %q does not contain %q", message, expected)
		}
	}
}

func TestValidateFieldsChecksOnlyRequestedFields(t *testing.T) {
	cfg := Default()
	if err := ValidateFields(cfg, "editor"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFields(cfg, "projectZomboid.homeDirectory"); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required-path error, got %v", err)
	}
}

func TestValidateFieldsDistinguishesFilesAndDirectories(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	cfg.ProjectZomboid.HomeDirectory = file
	cfg.ProjectZomboid.Executable = directory
	err := ValidateFields(cfg, "projectZomboid.homeDirectory", "projectZomboid.executable")
	if err == nil {
		t.Fatal("expected file-type validation failures")
	}
	for _, expected := range []string{
		"projectZomboid.homeDirectory (directory)",
		"projectZomboid.executable (file)",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error %q does not contain %q", err, expected)
		}
	}
}

func validTestConfig(t *testing.T) Config {
	t.Helper()
	home := t.TempDir()
	install := t.TempDir()
	executable := filepath.Join(install, "ProjectZomboid")
	if err := os.WriteFile(executable, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	cfg.ProjectZomboid.HomeDirectory = home
	cfg.ProjectZomboid.InstallDirectory = install
	cfg.ProjectZomboid.Executable = executable
	return cfg
}
