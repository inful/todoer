package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// safeWriteFile writes data to a file safely with atomic operations.
func safeWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(filename)+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
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

// getConfigValue returns the CLI value if provided, otherwise falls back to config value.
func getConfigValue(cliValue, configValue string) string {
	if cliValue != "" {
		return cliValue
	}
	return configValue
}

// fatalError logs an error and exits with code 1.
func fatalError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}
