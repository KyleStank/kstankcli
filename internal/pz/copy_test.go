package pz

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyLocalModAndForceReplacement(t *testing.T) {
	home := t.TempDir()
	name := "ExampleMod"
	source := filepath.Join(home, "Workshop", name, "Contents", "mods", name)
	if err := os.MkdirAll(filepath.Join(source, "media"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "media", "data.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyLocalMod(home, name, false); err != nil {
		t.Fatal(err)
	}
	destinationFile := filepath.Join(home, "mods", name, "media", "data.txt")
	if data, err := os.ReadFile(destinationFile); err != nil || string(data) != "new" {
		t.Fatalf("copied data = %q, err = %v", data, err)
	}

	if err := CopyLocalMod(home, name, false); err == nil {
		t.Fatal("expected overwrite refusal")
	}
	if err := os.WriteFile(filepath.Join(source, "media", "data.txt"), []byte("replacement"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyLocalMod(home, name, true); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(destinationFile); err != nil || string(data) != "replacement" {
		t.Fatalf("replaced data = %q, err = %v", data, err)
	}
}

func TestCopyLocalModRejectsUnsafeNames(t *testing.T) {
	for _, name := range []string{"", ".", "..", "nested/mod", `nested\mod`} {
		if err := CopyLocalMod(t.TempDir(), name, false); err == nil {
			t.Errorf("expected %q to be rejected", name)
		}
	}
}
