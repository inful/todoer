package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inful/todoer/pkg/core"
)

func TestCmdNew(t *testing.T) {
	tempDir := setupTempDir(t)

	config := &Config{RootDir: tempDir}

	// Create a previous journal to use as source
	prevJournal := filepath.Join(tempDir, "2024/01/2024-01-01.md")
	createTestFile(t, prevJournal, `---
date: 2024-01-01
---

# Daily Journal

## Todos

- [ ] Previous task
- [x] Completed task

## Notes

Previous notes.
`)

	tests := []struct {
		name        string
		rootDir     string
		expectError bool
	}{
		{
			name:        "successful creation",
			rootDir:     tempDir,
			expectError: false,
		},
		{
			name:        "invalid root directory",
			rootDir:     "/nonexistent/path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(ModeQuiet)
			err := cmdNewWithOptions(tt.rootDir, "", false, false, config, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("cmdNewWithOptions() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("cmdNewWithOptions() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCmdNew_AlreadyExists(t *testing.T) {
	tempDir := setupTempDir(t)

	config := &Config{RootDir: tempDir}

	// Create journal for today
	today := time.Now().Format("2006-01-02")
	year := time.Now().Format("2006")
	month := time.Now().Format("01")
	expectedPath := filepath.Join(tempDir, year, month, today+".md")
	createTestFile(t, expectedPath, "existing content")

	// Should not error if file already exists
	logger := NewLogger(ModeQuiet)
	err := cmdNewWithOptions(tempDir, "", false, false, config, logger)
	if err != nil {
		t.Errorf("cmdNewWithOptions() unexpected error when file exists: %v", err)
	}
}

func TestCmdNewWithOptions_DisableSourceBackup(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	createTestFile(t, source, `---
date: `+yesterday+`
---

# Daily Journal

## Todos

- [[`+yesterday+`]]
  - [ ] Carryover item

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdNewWithOptions(tempDir, "", false, false, config, logger); err != nil {
		t.Fatalf("cmdNewWithOptions() unexpected error: %v", err)
	}

	target := buildJournalPath(tempDir, today)
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected today's journal to be created: %v", err)
	}

	if _, err := os.Stat(source + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file when preserveSourceBackup=false, got err=%v", err)
	}
}

func TestCmdNewWithOptions_DisableSourceBackup_PreservesCompletedAndCarriesUnfinished(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	createTestFile(t, source, `---
date: `+yesterday+`
---

# Daily Journal

## Todos

- [x] Done task
- [ ] Carry task

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdNewWithOptions(tempDir, "", false, false, config, logger); err != nil {
		t.Fatalf("cmdNewWithOptions() unexpected error: %v", err)
	}

	if _, err := os.Stat(source + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file when preserveSourceBackup=false, got err=%v", err)
	}

	sourceAfter, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("failed to read source after carryover: %v", err)
	}

	sourceAfterStr := string(sourceAfter)
	if !strings.Contains(sourceAfterStr, "Done task") {
		t.Fatalf("expected source to preserve completed task, got:\n%s", sourceAfterStr)
	}
	if strings.Contains(sourceAfterStr, "Carry task") {
		t.Fatalf("expected source to remove carried unfinished task, got:\n%s", sourceAfterStr)
	}

	target := buildJournalPath(tempDir, today)
	targetAfter, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read target after carryover: %v", err)
	}

	targetAfterStr := string(targetAfter)
	if !strings.Contains(targetAfterStr, "Carry task") {
		t.Fatalf("expected target to contain carried unfinished task, got:\n%s", targetAfterStr)
	}
	if strings.Contains(targetAfterStr, "Done task") {
		t.Fatalf("expected target to exclude completed task, got:\n%s", targetAfterStr)
	}
}

func TestCmdNew_DoesNotCreateBackupByDefault(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	createTestFile(t, source, `---
date: `+yesterday+`
---

# Daily Journal

## Todos

- [[`+yesterday+`]]
  - [ ] Carryover item

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdNewWithOptions(tempDir, "", false, false, config, logger); err != nil {
		t.Fatalf("cmdNewWithOptions() unexpected error: %v", err)
	}

	if _, err := os.Stat(source + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file from cmdNew by default, got err=%v", err)
	}
}

func TestCmdNewWithOptions_NoPreviousJournalCreatesEmpty(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)

	logger := NewLogger(ModeQuiet)
	if err := cmdNewWithOptions(tempDir, "", false, false, config, logger); err != nil {
		t.Fatalf("cmdNewWithOptions: %v", err)
	}

	after, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("expected today's journal to be created, got: %v", err)
	}
	if !strings.Contains(string(after), core.TodosHeader) {
		t.Fatalf("expected journal to contain the todos header, got:\n%s", string(after))
	}
}
