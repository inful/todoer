package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inful/todoer/pkg/core"
)

// TestWriteOrderTargetBeforeSource pins the ordering that keeps a failed
// target write from deleting the unfinished tasks: the target lands first,
// so a failure there leaves the source untouched.
func TestWriteOrderTargetBeforeSource(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)

	source := buildJournalPath(tempDir, yesterday)
	original := "---\ndate: " + yesterday + "\n---\n\n# J\n\n## Todos\n\n- [[" + yesterday +
		"]]\n  - [ ] IMPORTANT unfinished task\n  - [x] finished\n\n## Notes\n"
	createTestFile(t, source, original)

	// A directory where the target file should be makes the final rename
	// fail, which is how a full disk or a permissions problem behaves.
	blocked := filepath.Join(tempDir, "blocked.md")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
	err := processJournal(ProcessOptions{
		SourceFile:   source,
		TargetFile:   blocked,
		TemplateDate: today,
	}, config, NewLogger(ModeQuiet))
	if err == nil {
		t.Fatal("expected an error writing the target")
	}

	after, readErr := os.ReadFile(source)
	if readErr != nil {
		t.Fatalf("read source: %v", readErr)
	}
	if string(after) != original {
		t.Fatalf("source must be untouched when the target write fails.\n got:\n%s\nwant:\n%s", after, original)
	}
	if !strings.Contains(string(after), "IMPORTANT unfinished task") {
		t.Fatal("unfinished task was deleted from the source")
	}
}

// TestFindClosestJournalFile_IgnoresNonJournalFiles pins that carryover only
// ever picks a file where todoer itself writes journals. Anything else with a
// date-shaped name gets rewritten in place, and 'new' writes no backup.
func TestFindClosestJournalFile_IgnoresNonJournalFiles(t *testing.T) {
	tempDir := setupTempDir(t)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)

	// A date-named note in an unrelated subfolder, and one in a hidden dir.
	stray := filepath.Join(tempDir, "meetings", yesterday+".md")
	createTestFile(t, stray, "---\ntitle: "+yesterday+"\n---\n\n## Todos\n\n- [["+yesterday+
		"]]\n  - [ ] meeting follow-up\n\n## Notes\n")
	hidden := filepath.Join(tempDir, ".trash", "2025", "06", "2025-06-20.md")
	createTestFile(t, hidden, "---\ntitle: 2025-06-20\n---\n\n## Todos\n\n- [[2025-06-20]]\n  - [ ] deleted journal task\n\n## Notes\n")

	// The real journal, one day older, must win.
	journal := buildJournalPath(tempDir, "2025-06-19")
	createTestFile(t, journal, "---\ntitle: 2025-06-19\n---\n\n## Todos\n\n- [[2025-06-19]]\n  - [ ] real task\n\n## Notes\n")

	got, err := findClosestJournalFile(tempDir, yesterday)
	if err != nil {
		t.Fatalf("findClosestJournalFile: %v", err)
	}
	if got != journal {
		t.Fatalf("expected the real journal %s, got %s", journal, got)
	}
}

// TestCRLFJournalSurvivesAdd pins that a journal saved with Windows line
// endings is found, parsed and written back with the same endings.
func TestCRLFJournalSurvivesAdd(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)

	createTestFile(t, journalPath, "---\r\ntitle: "+today+"\r\n---\r\n\r\n# J\r\n\r\n## Todos\r\n\r\n- [["+today+
		"]]\r\n  - [ ] windows task\r\n\r\n## Notes\r\n\r\nkeep\r\n")

	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
	if err := cmdAdd(tempDir, "", "a todo", false, false, config, NewLogger(ModeQuiet)); err != nil {
		t.Fatalf("cmdAdd on a CRLF journal: %v", err)
	}

	after, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	got := string(after)
	if bare := strings.Count(strings.ReplaceAll(got, "\r\n", ""), "\n"); bare != 0 {
		t.Errorf("expected CRLF throughout, found %d bare LF:\n%q", bare, got)
	}
	for _, want := range []string{"windows task", "a todo", "keep"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%q", want, got)
		}
	}
}

// TestTUIOpensReadOnlyOnUnparseableJournal pins that a line the model cannot
// represent makes the TUI read-only instead of refusing to start — and that
// it refuses to write.
func TestTUIOpensReadOnlyOnUnparseableJournal(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "j.md")
	original := "---\ntitle: 2026-03-16\n---\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a task\n\nA loose note line\n\n## Notes\n"
	if err := os.WriteFile(journalPath, []byte(original), 0o644); err != nil {
		t.Fatalf("write journal: %v", err)
	}

	m, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("the TUI must still open, got: %v", err)
	}
	if !m.readOnly {
		t.Fatal("expected read-only mode")
	}
	if !strings.Contains(m.status, "unparseable line") {
		t.Errorf("status should explain why, got %q", m.status)
	}

	// Any write attempt is refused and the file is left alone.
	m.dirty = true
	if err := m.saveToDisk(); err == nil {
		t.Fatal("expected saveToDisk to refuse in read-only mode")
	}
	after, _ := os.ReadFile(journalPath)
	if string(after) != original {
		t.Fatalf("file changed in read-only mode:\n%s", after)
	}
}

// TestTUIQuitAlwaysQuits pins that both quit keys exit even when the save
// fails, so a conflict cannot trap the user in the TUI.
func TestTUIQuitAlwaysQuits(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "j.md")
	if err := os.WriteFile(journalPath, []byte("---\ntitle: 2026-03-16\n---\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a\n\n## Notes\n"), 0o644); err != nil {
		t.Fatalf("write journal: %v", err)
	}

	m, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}
	m.items[0].item.Completed = true
	m.dirty = true
	// Someone else edits the file underneath.
	if err := os.WriteFile(journalPath, []byte("---\ntitle: 2026-03-16\n---\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a\n  - [ ] b\n\n## Notes\n"), 0o644); err != nil {
		t.Fatalf("write journal: %v", err)
	}

	for _, key := range []tea.KeyMsg{{Type: tea.KeyCtrlC}, {Type: tea.KeyRunes, Runes: []rune{'q'}}} {
		_, cmd := m.updateNormalMode(key)
		if cmd == nil {
			t.Fatalf("key %v must quit even when the save fails", key)
		}
	}
}

// TestTUISaveKeepsBackup pins that the pre-save bytes survive next to the
// journal: the TUI rewrites the whole file, so this is the only undo.
func TestTUISaveKeepsBackup(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "j.md")
	original := "---\ntitle: 2026-03-16\n---\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a\n\n## Notes\n"
	if err := os.WriteFile(journalPath, []byte(original), 0o644); err != nil {
		t.Fatalf("write journal: %v", err)
	}

	m, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}
	m.items[0].item.Completed = true
	m.dirty = true
	if err := m.saveToDisk(); err != nil {
		t.Fatalf("saveToDisk: %v", err)
	}

	backup, err := os.ReadFile(journalPath + ".bak")
	if err != nil {
		t.Fatalf("expected a .bak next to the journal: %v", err)
	}
	if string(backup) != original {
		t.Fatalf("the .bak must hold the pre-save content, got:\n%s", backup)
	}
	updated, _ := os.ReadFile(journalPath)
	if strings.Contains(string(updated), "- [ ] a\n") {
		t.Fatalf("the toggle was not written:\n%s", updated)
	}
}

// TestPreviewDoesNotCreateDirectories pins that a read-only command has no
// filesystem side effects.
func TestPreviewDoesNotCreateDirectories(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "journals")

	config := &Config{RootDir: root, TodosHeader: core.TodosHeader}
	if err := cmdPreview("", "", "", "", "", config); err != nil {
		t.Fatalf("cmdPreview: %v", err)
	}
	if _, err := os.Stat(root); err == nil {
		t.Fatal("preview created the journals directory")
	}
}

// TestReadOnlyStatusIsHonest pins that the TUI never claims it reloaded or
// saved a journal it could not read.
func TestReadOnlyStatusIsHonest(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "j.md")
	original := "---\ntitle: 2026-03-16\n---\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a task\n\nA loose note line\n\n## Notes\n"
	if err := os.WriteFile(journalPath, []byte(original), 0o644); err != nil {
		t.Fatalf("write journal: %v", err)
	}

	m, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}
	if !m.readOnly {
		t.Fatal("expected read-only mode")
	}

	updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if status := updated.(tuiModel).status; status == "Reloaded from disk" {
		t.Errorf("reload must not claim success on a read-only journal, got %q", status)
	}

	updated, _ = m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if status := updated.(tuiModel).status; status == "Saved" {
		t.Errorf("save must not claim success on a read-only journal, got %q", status)
	}
}

// TestFindClosestJournalFile_OnlyJournalLayout pins that carryover only picks
// files where todoer writes journals: ROOT/YYYY/MM/YYYY-MM-DD.md.
func TestFindClosestJournalFile_OnlyJournalLayout(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)

	// The right file name, but in the wrong place.
	flat := filepath.Join(tempDir, yesterday+".md")
	createTestFile(t, flat, "---\ntitle: "+yesterday+"\n---\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] flat task\n\n## Notes\n")
	// The right place, but the wrong name.
	misplaced := filepath.Join(tempDir, "2025", "06", "notes.md")
	createTestFile(t, misplaced, "---\ntitle: 2025-06-20\n---\n\n## Todos\n\n- [[2025-06-20]]\n  - [ ] misplaced task\n\n## Notes\n")

	if _, err := findClosestJournalFile(tempDir, today); err == nil {
		t.Fatal("expected no journal to be found")
	}

	// The real layout is found.
	journal := buildJournalPath(tempDir, yesterday)
	createTestFile(t, journal, "---\ntitle: "+yesterday+"\n---\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] real task\n\n## Notes\n")
	got, err := findClosestJournalFile(tempDir, today)
	if err != nil {
		t.Fatalf("findClosestJournalFile: %v", err)
	}
	if got != journal {
		t.Fatalf("expected %s, got %s", journal, got)
	}
}
