package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSettableKeysUseJSONFieldNames(t *testing.T) {
	want := []string{
		"editor",
		"projectZomboid.executable",
		"projectZomboid.homeDirectory",
		"projectZomboid.installDirectory",
	}
	if got := SettableKeys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("keys = %#v, want %#v", got, want)
	}
}

func TestConfigOptionsAreCommaSeparated(t *testing.T) {
	type taggedConfig struct {
		Path string `config:"path,mustExist"`
	}
	field, ok := reflect.TypeOf(taggedConfig{}).FieldByName("Path")
	if !ok {
		t.Fatal("Path field not found")
	}
	for _, option := range []string{"path", "mustExist"} {
		if !hasConfigOption(field, option) {
			t.Fatalf("expected option %q in %q", option, field.Tag.Get("config"))
		}
	}
	if hasConfigOption(field, "readonly") {
		t.Fatal("unexpected readonly option")
	}
}

func TestSetValueUsesJSONPaths(t *testing.T) {
	cfg := Default()
	if err := SetValue(&cfg, "editor", "code"); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Editor, "code"; got != want {
		t.Fatalf("editor = %q, want %q", got, want)
	}
	home := t.TempDir()
	if err := SetValue(&cfg, "projectZomboid.homeDirectory", home); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.ProjectZomboid.HomeDirectory, filepath.Clean(home); got != want {
		t.Fatalf("homeDirectory = %q, want %q", got, want)
	}
	executable := filepath.Join(t.TempDir(), "ProjectZomboid64.exe")
	if err := os.WriteFile(executable, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SetValue(&cfg, "projectZomboid.executable", executable); err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.ProjectZomboid.Executable, executable; got != want {
		t.Fatalf("executable = %q, want %q", got, want)
	}
}

func TestSetValueRejectsNonJSONAndReadOnlyPaths(t *testing.T) {
	for _, key := range []string{
		"project-zomboid.home-directory",
		"ProjectZomboid.homeDirectory",
		"projectZomboid.HomeDirectory",
		"version",
		"projectZomboid",
	} {
		t.Run(key, func(t *testing.T) {
			cfg := Default()
			err := SetValue(&cfg, key, "value")
			if err == nil || !strings.Contains(err.Error(), "unknown config key") {
				t.Fatalf("expected unknown-key error, got %v", err)
			}
		})
	}
}

func TestSetValueRollsBackInvalidConfig(t *testing.T) {
	cfg := Default()
	err := SetValue(&cfg, "editor", "vim")
	if err == nil || !strings.Contains(err.Error(), "must be one of: code") {
		t.Fatalf("expected editor validation error, got %v", err)
	}
	if got, want := cfg, Default(); !reflect.DeepEqual(got, want) {
		t.Fatalf("config = %#v, want %#v", got, want)
	}
}

func TestSetValueRejectsInvalidPathAndRollsBack(t *testing.T) {
	cfg := Default()
	missing := filepath.Join(t.TempDir(), "missing")
	err := SetValue(&cfg, "projectZomboid.installDirectory", missing)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected missing-path error, got %v", err)
	}
	if got := cfg.ProjectZomboid.InstallDirectory; got != "" {
		t.Fatalf("installDirectory = %q after rollback, want empty", got)
	}
}

func TestSetValueIgnoresInvalidUnrelatedFields(t *testing.T) {
	cfg := Default()
	cfg.Editor = "vim"
	home := t.TempDir()
	if err := SetValue(&cfg, "projectZomboid.homeDirectory", home); err != nil {
		t.Fatal(err)
	}
	if cfg.ProjectZomboid.HomeDirectory != home {
		t.Fatalf("homeDirectory = %q, want %q", cfg.ProjectZomboid.HomeDirectory, home)
	}
}
