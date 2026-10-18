package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteLoadAndStrictDecoding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	cfg := Default()
	cfg.ProjectZomboid.HomeDirectory = "/games/Zomboid"
	if err := Write(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != cfg {
		t.Fatalf("got %#v, want %#v", loaded, cfg)
	}

	if err := os.WriteFile(path, []byte(`{"version":1,"editor":"code","projectZomboid":{},"typo":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field error, got %v", err)
	}
}

func TestInitDoesNotOverwriteWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := Init(path, false); err != nil {
		t.Fatal(err)
	}
	if err := Init(path, false); err == nil {
		t.Fatal("expected existing-config error")
	}
	if err := Init(path, true); err != nil {
		t.Fatal(err)
	}
}

func TestUnsupportedVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"version":2,"editor":"code","projectZomboid":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "unsupported config version") {
		t.Fatalf("expected version error, got %v", err)
	}
}
