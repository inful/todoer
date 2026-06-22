package main

import (
	"path/filepath"
	"testing"
)

func TestSafeWriteFile_BadTarget(t *testing.T) {
	tempDir := t.TempDir()

	// Target inside a non-existent directory should fail.
	bad := filepath.Join(tempDir, "no-such-dir", "out.md")
	if err := safeWriteFile(bad, []byte("hi"), 0o644); err == nil {
		t.Fatalf("expected error when target directory does not exist")
	}
}
