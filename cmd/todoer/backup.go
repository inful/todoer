package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultBackupCount is how many previous versions of a file are kept.
const DefaultBackupCount = 3

// backupFile saves the current content of path and rotates the older
// copies, so a writer that keeps producing broken files cannot destroy
// every good copy.
//
// Backups live in one place — ROOT/.backups/, mirroring the journal tree
// — so the journal folder stays clean. A file outside ROOT has no tree to
// mirror, so it keeps a single .bak next to it.
//
// A count of 0 keeps no backups at all.
func backupFile(rootDir, path string, count int) error {
	if count <= 0 {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to back up
		}
		return fmt.Errorf("failed to read %s for backup: %w", path, err)
	}

	if rel, ok := pathUnder(rootDir, path); ok {
		return rotateBackup(
			filepath.Join(rootDir, BackupDirName, filepath.Dir(rel)),
			filepath.Base(rel),
			content,
			count,
		)
	}

	// Outside the journal tree: one copy beside the file.
	return safeWriteFile(path+".bak", content, FilePermissions)
}

// rotateBackup writes content as copy 1 of name in dir and moves the
// older copies down, dropping the oldest one.
func rotateBackup(dir, name string, content []byte, count int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create backup directory %s: %w", dir, err)
	}

	copyName := func(n int) string {
		return filepath.Join(dir, name+"."+strconv.Itoa(n))
	}

	// The oldest copy goes away first, then the others move down.
	if err := os.Remove(copyName(count)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old backup %s: %w", copyName(count), err)
	}
	for i := count - 1; i >= 1; i-- {
		if _, err := os.Stat(copyName(i)); err != nil {
			continue // no such copy
		}
		if err := os.Rename(copyName(i), copyName(i+1)); err != nil {
			return fmt.Errorf("failed to rotate backup %s: %w", copyName(i), err)
		}
	}

	if err := safeWriteFile(copyName(1), content, FilePermissions); err != nil {
		return fmt.Errorf("failed to write backup %s: %w", copyName(1), err)
	}
	return nil
}

// pathUnder reports whether path is inside rootDir and returns its path
// relative to rootDir.
func pathUnder(rootDir, path string) (string, bool) {
	if rootDir == "" {
		return "", false
	}
	rel, err := filepath.Rel(rootDir, path)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

// backupPath returns the path of the newest backup copy of path, for
// messages and for the tests.
func backupPath(rootDir, path string) string {
	if rel, ok := pathUnder(rootDir, path); ok {
		return filepath.Join(rootDir, BackupDirName, filepath.Dir(rel), filepath.Base(rel)+".1")
	}
	return path + ".bak"
}
