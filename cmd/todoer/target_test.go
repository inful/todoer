package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inful/todoer/pkg/core"
)

// TestProcessBacksUpTheTargetItOverwrites pins that 'process' keeps a copy
// of the target before replacing it. The target may be the user's real
// journal, and the old content was unrecoverable.
func TestProcessBacksUpTheTargetItOverwrites(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)

	source := filepath.Join(tempDir, "src.md")
	target := filepath.Join(tempDir, "target.md")
	createTestFile(t, source, "---\ndate: 2025-06-20\n---\n\n## Todos\n\n- [[2025-06-20]]\n  - [ ] carried task\n\n## Notes\n")
	original := "---\ntitle: " + today + "\n---\n\n# My journal\n\n## Todos\n\n- [[" + today +
		"]]\n  - [ ] something I wrote today\n\n## Notes\n\nIMPORTANT personal notes\n"
	createTestFile(t, target, original)

	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
	if err := processJournal(ProcessOptions{
		SourceFile:   source,
		TargetFile:   target,
		TemplateDate: today,
	}, config, NewLogger(ModeQuiet)); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	backup, err := os.ReadFile(backupPath(tempDir, target))
	if err != nil {
		t.Fatalf("expected a backup of the target, got: %v", err)
	}
	if string(backup) != original {
		t.Errorf("the backup must hold the old target content:\ngot:\n%s\nwant:\n%s", backup, original)
	}

	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !strings.Contains(string(after), "carried task") {
		t.Errorf("the target must hold the processed content:\n%s", after)
	}
}

// TestNewDoesNotBackUpTheTarget pins that the merge path, which keeps the
// target's content, does not litter it with a .bak.
func TestNewDoesNotBackUpTheTarget(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)

	source := filepath.Join(tempDir, "src.md")
	target := filepath.Join(tempDir, "target.md")
	createTestFile(t, source, "---\ndate: 2025-06-20\n---\n\n## Todos\n\n- [[2025-06-20]]\n  - [ ] carried task\n\n## Notes\n")
	createTestFile(t, target, "---\ntitle: "+today+"---\n\n## Todos\n\n## Notes\n\nkeep me\n")

	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
	if err := processJournal(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, NewLogger(ModeQuiet)); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	if _, err := os.Stat(target + ".bak"); err == nil {
		t.Error("the merge path keeps the target's content, so it needs no .bak")
	}
}

// TestAddToEmptyJournal pins that an empty journal file gets a todos
// section instead of an error.
func TestAddToEmptyJournal(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)
	createTestFile(t, journalPath, "")

	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
	if err := cmdAdd(tempDir, "", "first todo", false, config, NewLogger(ModeQuiet)); err != nil {
		t.Fatalf("cmdAdd on an empty journal: %v", err)
	}

	after, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	got := string(after)
	if !strings.Contains(got, core.TodosHeader) {
		t.Errorf("expected a todos section header, got:\n%q", got)
	}
	if !strings.Contains(got, "first todo") {
		t.Errorf("expected the todo to be added, got:\n%q", got)
	}
}

// TestAddKeepsDaysInOrder pins that 'add' writes the day sections in date
// order, like the TUI does.
func TestAddKeepsDaysInOrder(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)

	// A later day comes first in the file.
	earlier := time.Now().AddDate(0, 0, -2).Format(core.DateFormat)
	later := time.Now().AddDate(0, 0, 8).Format(core.DateFormat)
	createTestFile(t, journalPath, "---\ntitle: "+today+"\n---\n\n## Todos\n\n"+
		"- [["+later+"]]\n  - [ ] later task\n\n"+
		"- [["+earlier+"]]\n  - [ ] earlier task\n\n## Notes\n")

	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
	if err := cmdAdd(tempDir, "", "new todo", false, config, NewLogger(ModeQuiet)); err != nil {
		t.Fatalf("cmdAdd: %v", err)
	}

	after, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	got := string(after)
	// Match the day headers, not the date in the frontmatter.
	earlierIdx := strings.Index(got, "- [["+earlier+"]]")
	todayIdx := strings.Index(got, "- [["+today+"]]")
	laterIdx := strings.Index(got, "- [["+later+"]]")
	if earlierIdx < 0 || laterIdx < 0 || todayIdx < 0 {
		t.Fatalf("missing a day section in:\n%s", got)
	}
	if earlierIdx >= todayIdx || todayIdx >= laterIdx {
		t.Errorf("day sections are not in date order:\n%s", got)
	}
}

// TestAddErrorNamesTheNextStep pins that a journal without a todos section
// says what to do about it.
func TestAddErrorNamesTheNextStep(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)
	createTestFile(t, journalPath, "---\ntitle: "+today+"\n---\n\n# Journal without a todos section\n")

	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
	err := cmdAdd(tempDir, "", "a todo", false, config, NewLogger(ModeQuiet))
	if err == nil {
		t.Fatal("expected an error for a journal without a todos section")
	}
	if !strings.Contains(err.Error(), "todos_header") {
		t.Errorf("the error should say what to do, got: %v", err)
	}
}

// TestProcessRotatesTargetBackups pins that a second run does not throw
// away the copy of the file the user actually had: the backups rotate, so
// the older copy survives.
func TestProcessRotatesTargetBackups(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)

	source := filepath.Join(tempDir, "src.md")
	target := filepath.Join(tempDir, "target.md")
	createTestFile(t, source, "---\ndate: 2025-06-20\n---\n\n## Todos\n\n- [[2025-06-20]]\n  - [ ] carried task\n\n## Notes\n")
	original := "---\ntitle: " + today + "\n---\n\n## Notes\n\nMY ORIGINAL CONTENT\n"
	createTestFile(t, target, original)

	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
	for range 2 {
		if err := processJournal(ProcessOptions{
			SourceFile:   source,
			TargetFile:   target,
			TemplateDate: today,
		}, config, NewLogger(ModeQuiet)); err != nil {
			t.Fatalf("processJournal: %v", err)
		}
	}

	// The newest backup holds the target as it was before the last run,
	// the one before it the original.
	newest, err := os.ReadFile(backupPath(tempDir, target))
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if strings.Contains(string(newest), "MY ORIGINAL CONTENT") {
		t.Errorf("the newest backup must hold the version before the last run, got:\n%s", newest)
	}

	older := strings.TrimSuffix(backupPath(tempDir, target), ".1") + ".2"
	content, err := os.ReadFile(older)
	if err != nil {
		t.Fatalf("read the older backup: %v", err)
	}
	if !strings.Contains(string(content), "MY ORIGINAL CONTENT") {
		t.Errorf("the older backup must still hold the original, got:\n%s", content)
	}
}

// TestNewLeavesNoTempFileInTheJournalFolder pins that the placeholder
// journal used when there is no previous journal is not written into the
// user's journal tree.
func TestNewLeavesNoTempFileInTheJournalFolder(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	if err := cmdNewWithOptions(tempDir, "", false, config, NewLogger(ModeQuiet)); err != nil {
		t.Fatalf("cmdNewWithOptions: %v", err)
	}

	var strays []string
	if err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.Contains(filepath.Base(path), "empty-journal") {
			strays = append(strays, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(strays) > 0 {
		t.Errorf("temp files left in the journal folder: %v", strays)
	}
}
