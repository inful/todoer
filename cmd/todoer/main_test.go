package main

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTempDir creates a temporary directory scoped to the test.
func setupTempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// tUnsetenv unsets an environment variable for the duration of the test,
// mirroring the t.Setenv naming convention. Prefer t.Setenv in new tests
// (available since Go 1.17); this helper exists for the few existing
// callers that need to control the unset timing.
func tUnsetenv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("cannot unset environment variable %s: %v", key, err)
	}
}

// createTestFile writes content to path, creating the parent directory
// if it does not exist.
func createTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}
