package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/inful/todoer/pkg/core"
)

func TestCmdAdd_DoesNotCreateBackupByDefault(t *testing.T) {
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

- [[`+yesterday+`]]
  - [ ] Carryover item

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdAdd(tempDir, "", "new todo", false, false, config, logger); err != nil {
		t.Fatalf("cmdAdd() unexpected error: %v", err)
	}

	if _, err := os.Stat(source + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file from cmdAdd by default, got err=%v", err)
	}

	target := buildJournalPath(tempDir, today)
	targetAfter, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read target: %v", err)
	}
	if !strings.Contains(string(targetAfter), "new todo") {
		t.Fatalf("expected target to contain the new todo, got:\n%s", string(targetAfter))
	}
}

func TestCmdAdd_WithBackupCreatesBackup(t *testing.T) {
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

- [[`+yesterday+`]]
  - [ ] Carryover item

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdAdd(tempDir, "", "new todo", false, true, config, logger); err != nil {
		t.Fatalf("cmdAdd() unexpected error: %v", err)
	}

	backupPath := source + ".bak"
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup file when backup=true, got err=%v", err)
	}

	target := buildJournalPath(tempDir, today)
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected target file to be created, got err=%v", err)
	}
}

func TestCmdAdd_PreservesCompletedUndatedTodos(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)

	// Pre-create today's journal with both completed and uncompleted undated
	// todos. The add path runs MoveUndatedTodosToCurrentDate internally, so
	// both classes of undated todos should land in today's section rather
	// than the completed ones being silently dropped.
	createTestFile(t, journalPath, `---
date: `+today+`
---

# Daily Journal

## Todos

- [x] Done last week
- [ ] Undated carry
- [x] Done yesterday

## Notes
`)

	logger := NewLogger(ModeQuiet)
	if err := cmdAdd(tempDir, "", "fresh today item", false, false, config, logger); err != nil {
		t.Fatalf("cmdAdd() unexpected error: %v", err)
	}

	after, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("failed to read today's journal: %v", err)
	}
	content := string(after)

	// The new todo must be present.
	if !strings.Contains(content, "fresh today item") {
		t.Fatalf("expected new todo to be appended, got:\n%s", content)
	}
	// The undated completed and uncompleted items must both be preserved.
	if !strings.Contains(content, "Done last week") {
		t.Fatalf("expected undated completed todo to be preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "Undated carry") {
		t.Fatalf("expected undated uncompleted todo to be preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "Done yesterday") {
		t.Fatalf("expected second undated completed todo to be preserved, got:\n%s", content)
	}
	// All four items must live under today's [[date]] section. The journal
	// should not have any undated day left at the top of the todos section.
	if !strings.Contains(content, "[["+today+"]]") {
		t.Fatalf("expected today's [[date]] section to exist, got:\n%s", content)
	}
}

func TestCmdAdd_EmptyTextReturnsError(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir}

	logger := NewLogger(ModeQuiet)
	if err := cmdAdd(tempDir, "", "   ", false, false, config, logger); err == nil {
		t.Fatalf("expected error on empty todo text")
	}
}

func TestCmdAdd_PrintPath(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)
	createTestFile(t, journalPath, "---\ndate: "+today+"\n---\n\n# J\n\n## Todos\n\n- [ ] existing\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	if err := cmdAdd(tempDir, "", "new", true, false, config, logger); err != nil {
		t.Fatalf("cmdAdd: %v", err)
	}
}

func TestFindOrCreateDaySection(t *testing.T) {
	existing := &core.DaySection{Date: "2026-03-16", Items: []*core.TodoItem{{Text: "x"}}}
	journal := &core.TodoJournal{Days: []*core.DaySection{existing}}

	// Finds existing.
	got := core.FindOrCreateDaySection(journal, "2026-03-16")
	if got != existing {
		t.Fatalf("expected to return the existing day section")
	}

	// Creates a new one when missing.
	got2 := core.FindOrCreateDaySection(journal, "2026-03-17")
	if got2 == nil || got2.Date != "2026-03-17" {
		t.Fatalf("expected a new day section for 2026-03-17, got %+v", got2)
	}
	if len(journal.Days) != 2 {
		t.Fatalf("expected journal to now have 2 day sections, got %d", len(journal.Days))
	}
}

func TestAppendTodoToJournal_NoExistingFile(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)

	if err := appendTodoToJournal(journalPath, today, "first", config); err == nil {
		t.Fatalf("expected error when journal file does not exist")
	}
}
