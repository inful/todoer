package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inful/todoer/pkg/core"
)

func TestTUICheckExternalChanges(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "2026-03-16.md")
	createTestFile(t, journalPath, "---\ntitle: 2026-03-16\n---\n\n# Journal\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a\n\n## Notes\n")

	m, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}

	// No change -> no status update.
	m.status = "stable"
	m.checkExternalChanges()
	if m.status != "stable" {
		t.Fatalf("expected status to remain stable, got %q", m.status)
	}

	// External change while dirty blocks save.
	m.dirty = true
	if err := os.WriteFile(journalPath, []byte("---\ntitle: 2026-03-16\n---\n\n# J\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a\n\n## Notes\n"), 0o644); err != nil {
		t.Fatalf("rewrite failed: %v", err)
	}
	m.checkExternalChanges()
	if !m.externalChanged {
		t.Fatalf("expected externalChanged to be set after external change")
	}
	if !strings.Contains(m.status, "External") {
		t.Fatalf("expected external-change status, got %q", m.status)
	}

	// External change while clean auto-reloads.
	m.dirty = false
	m.status = "clean"
	if err := os.WriteFile(journalPath, []byte("---\ntitle: 2026-03-16\n---\n\n# J\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] changed\n\n## Notes\n"), 0o644); err != nil {
		t.Fatalf("rewrite failed: %v", err)
	}
	m.checkExternalChanges()
	if m.dirty {
		t.Fatalf("expected dirty=false after clean auto-reload")
	}
	if !strings.Contains(m.status, "reloaded") {
		t.Fatalf("expected reload status, got %q", m.status)
	}
	if len(m.items) != 1 || m.items[0].item.Text != "changed" {
		t.Fatalf("expected reloaded items, got %+v", m.items)
	}

	// Missing file path -> no panic, no status change.
	m2, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel (second): %v", err)
	}
	if err := os.Remove(journalPath); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	prev := m2.status
	m2.checkExternalChanges()
	if m2.status != prev {
		t.Fatalf("expected status to be unchanged when file is missing, got %q", m2.status)
	}
}

func TestTUISaveToDisk(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "2026-03-16.md")
	createTestFile(t, journalPath, "---\ntitle: 2026-03-16\n---\n\n# Journal\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a\n\n## Notes\n")

	m, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}

	// No-op when not dirty.
	if err := m.saveToDisk(); err != nil {
		t.Fatalf("saveToDisk on clean model should be a no-op, got %v", err)
	}

	// Edit a todo, save, verify file is updated.
	m.items[0].item.Completed = true
	m.dirty = true
	if err := m.saveToDisk(); err != nil {
		t.Fatalf("saveToDisk: %v", err)
	}
	after, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("read after save: %v", err)
	}
	if !strings.Contains(string(after), "[x] a") {
		t.Fatalf("expected toggled item to be persisted, got:\n%s", string(after))
	}
	if m.dirty {
		t.Fatalf("expected dirty=false after save")
	}
	if m.externalChanged {
		t.Fatalf("expected externalChanged=false after save")
	}

	// Save blocked when the file changed externally.
	m.dirty = true
	if err := os.WriteFile(journalPath, []byte("---\ntitle: 2026-03-16\n---\n\n# J\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a\n\n## Notes\n"), 0o644); err != nil {
		t.Fatalf("rewrite failed: %v", err)
	}
	if err := m.saveToDisk(); err == nil {
		t.Fatalf("expected saveToDisk to fail after external change")
	}
	if !m.externalChanged {
		t.Fatalf("expected externalChanged=true after blocked save")
	}
}

func TestTUIFindDaySection(t *testing.T) {
	if got := core.FindDaySection(nil, "2026-03-16"); got != nil {
		t.Fatalf("expected nil journal to return nil")
	}
	day := &core.DaySection{Date: "2026-03-16", Items: []*core.TodoItem{}}
	journal := &core.TodoJournal{Days: []*core.DaySection{day}}
	if got := core.FindDaySection(journal, "2099-01-01"); got != nil {
		t.Fatalf("expected no-match to return nil")
	}
	if got := core.FindDaySection(journal, "2026-03-16"); got != day {
		t.Fatalf("expected match to return the day section")
	}
}
