package main

import (
	"bytes"
	"io"
	"os"
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

// TestSafeWriteFile_IsQuiet pins that a successful write prints nothing to
// stderr. The deferred cleanup closes the temp file that was already
// closed, and reporting that would print a warning on every write.
func TestSafeWriteFile_IsQuiet(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "journal.md")

	stderr := captureStderr(t, func() {
		if err := safeWriteFile(target, []byte("hello"), FilePermissions); err != nil {
			t.Fatalf("safeWriteFile: %v", err)
		}
	})

	if stderr != "" {
		t.Errorf("expected no output on stderr, got: %q", stderr)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

// captureStderr runs fn with os.Stderr redirected and returns what it wrote.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	w.Close()
	out := <-done
	r.Close()
	return out
}
