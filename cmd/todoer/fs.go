package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// safeWriteFile atomically writes data to filename using a temp file and rename.
func safeWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(filename)+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	defer func() {
		// Best effort: the file was either renamed (the remove is a
		// no-op) or abandoned. Neither failure can be acted on here.
		if err := tmpFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "todoer: warning: closing temp file: %v\n", err)
		}
		if err := os.Remove(tmpFile.Name()); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "todoer: warning: removing temp file %s: %v\n", tmpFile.Name(), err)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), filename); err != nil {
		return fmt.Errorf("failed to move temporary file to target: %w", err)
	}

	if err := os.Chmod(filename, perm); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
