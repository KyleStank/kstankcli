package pz

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func CopyLocalMod(homeDirectory, modName string, force bool) error {
	if err := validateModName(modName); err != nil {
		return err
	}
	if homeDirectory == "" {
		return errors.New("Project Zomboid home directory is not configured")
	}

	source := filepath.Join(homeDirectory, "Workshop", modName, "Contents", "mods", modName)
	destinationParent := filepath.Join(homeDirectory, "mods")
	destination := filepath.Join(destinationParent, modName)
	if err := ensureWithin(filepath.Join(homeDirectory, "Workshop"), source); err != nil {
		return err
	}
	if err := ensureWithin(destinationParent, destination); err != nil {
		return err
	}
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("inspect source mod %q: %w", source, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source mod %q is not a directory", source)
	}

	if _, err := os.Lstat(destination); err == nil && !force {
		return fmt.Errorf("destination mod %q already exists (use --force to replace it)", destination)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect destination mod %q: %w", destination, err)
	}
	if err := os.MkdirAll(destinationParent, 0o755); err != nil {
		return fmt.Errorf("create mods directory %q: %w", destinationParent, err)
	}

	temporary, err := os.MkdirTemp(destinationParent, "."+modName+"-copy-*")
	if err != nil {
		return fmt.Errorf("create temporary mod directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(temporary) }()
	if err := copyDirectory(source, temporary); err != nil {
		return fmt.Errorf("copy mod to temporary directory: %w", err)
	}

	var backup string
	if _, err := os.Lstat(destination); err == nil {
		backup, err = reserveBackupPath(destinationParent, modName)
		if err != nil {
			return err
		}
		if err := os.Rename(destination, backup); err != nil {
			return fmt.Errorf("back up existing mod %q: %w", destination, err)
		}
		defer func() { _ = os.RemoveAll(backup) }()
	}

	if err := os.Rename(temporary, destination); err != nil {
		if backup != "" {
			_ = os.Rename(backup, destination)
		}
		return fmt.Errorf("activate copied mod %q: %w", destination, err)
	}
	if backup != "" {
		if err := os.RemoveAll(backup); err != nil {
			return fmt.Errorf("remove replaced mod backup %q: %w", backup, err)
		}
	}
	return nil
}

func validateModName(name string) error {
	if name == "" || name == "." || name == ".." || filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid mod name %q: use one directory name without path separators", name)
	}
	return nil
}

func ensureWithin(parent, child string) error {
	relative, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil {
		return fmt.Errorf("compare paths: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("path %q escapes expected directory %q", child, parent)
	}
	return nil
}

func reserveBackupPath(parent, modName string) (string, error) {
	path, err := os.MkdirTemp(parent, "."+modName+"-backup-*")
	if err != nil {
		return "", fmt.Errorf("reserve mod backup path: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("prepare mod backup path: %w", err)
	}
	return path, nil
}

func copyDirectory(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if entry.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported file type at %q", path)
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(source, destination string, mode fs.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}
