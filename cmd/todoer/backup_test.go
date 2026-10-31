package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBackupFile_KeepsCopiesInOneDirectory pins that previous versions live
// in ROOT/.backups, mirroring the journal tree, so the journal folder itself
// stays clean.
func TestBackupFile_KeepsCopiesInOneDirectory(t *testing.T) {
	root := t.TempDir()
	journal := filepath.Join(root, "2026", "10", "2026-10-31.md")
	if err := os.MkdirAll(filepath.Dir(journal), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(journal, []byte("first"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := backupFile(root, journal, DefaultBackupCount); err != nil {
		t.Fatalf("backupFile: %v", err)
	}

	backup := filepath.Join(root, ".backups", "2026", "10", "2026-10-31.md.1")
	content, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("expected a backup at %s, got: %v", backup, err)
	}
	if string(content) != "first" {
		t.Errorf("backup holds %q, want %q", content, "first")
	}

	// Nothing is written next to the journal itself.
	if _, err := os.Stat(journal + ".bak"); err == nil {
		t.Error("no .bak may be left next to the journal")
	}
}

// TestBackupFile_RotatesCopies pins that the copies rotate: the newest
// previous version is .1 and the older ones move down, so a writer that
// keeps breaking the file cannot destroy every good copy.
func TestBackupFile_RotatesCopies(t *testing.T) {
	root := t.TempDir()
	journal := filepath.Join(root, "2026", "10", "2026-10-31.md")
	if err := os.MkdirAll(filepath.Dir(journal), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	versions := []string{"v1", "v2", "v3", "v4"}
	for _, v := range versions {
		if err := os.WriteFile(journal, []byte(v), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := backupFile(root, journal, DefaultBackupCount); err != nil {
			t.Fatalf("backupFile: %v", err)
		}
	}

	dir := filepath.Join(root, ".backups", "2026", "10")
	want := map[string]string{
		"2026-10-31.md.1": "v4",
		"2026-10-31.md.2": "v3",
		"2026-10-31.md.3": "v2",
	}
	for name, content := range want {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != content {
			t.Errorf("%s holds %q, want %q", name, got, content)
		}
	}

	// Only three copies are kept: the oldest falls off.
	if _, err := os.Stat(filepath.Join(dir, "2026-10-31.md.4")); err == nil {
		t.Error("expected at most three copies")
	}
}

// TestBackupFile_OutsideTheTreeKeepsACopyNextToIt pins the fallback: a file
// that is not under the journals root has no tree to mirror.
func TestBackupFile_OutsideTheTreeKeepsACopyNextToIt(t *testing.T) {
	root := t.TempDir()
	elsewhere := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(elsewhere, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := backupFile(root, elsewhere, DefaultBackupCount); err != nil {
		t.Fatalf("backupFile: %v", err)
	}

	content, err := os.ReadFile(elsewhere + ".bak")
	if err != nil {
		t.Fatalf("expected a copy next to the file, got: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("copy holds %q, want %q", content, "hello")
	}
}

// TestBackupFile_CountZeroKeepsNothing pins that backups can be turned off.
func TestBackupFile_CountZeroKeepsNothing(t *testing.T) {
	root := t.TempDir()
	journal := filepath.Join(root, "2026", "10", "2026-10-31.md")
	if err := os.MkdirAll(filepath.Dir(journal), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(journal, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := backupFile(root, journal, 0); err != nil {
		t.Fatalf("backupFile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".backups")); err == nil {
		t.Error("no backup directory may be created when backups are off")
	}
}

// TestBackupDirectoryIsNotAJournal pins that the backup directory is never
// picked up as a journal: the files inside it are named like journals.
func TestBackupDirectoryIsNotAJournal(t *testing.T) {
	root := t.TempDir()
	journal := filepath.Join(root, "2026", "10", "2026-10-30.md")
	if err := os.MkdirAll(filepath.Dir(journal), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(journal, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := backupFile(root, journal, DefaultBackupCount); err != nil {
		t.Fatalf("backupFile: %v", err)
	}

	got, err := findClosestJournalFile(root, "2026-10-31")
	if err != nil {
		t.Fatalf("findClosestJournalFile: %v", err)
	}
	if got != journal {
		t.Errorf("expected the journal %s, got %s", journal, got)
	}
	if strings.Contains(got, BackupDirName) {
		t.Error("a file in the backup directory was taken for a journal")
	}
}
