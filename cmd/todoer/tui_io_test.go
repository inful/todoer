package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inful/todoer/pkg/core"
)

// TestHashBytes_Deterministic verifies the same input always produces
// the same hash (the contract saveToDisk / checkExternalChanges rely on
// to detect external file changes).
func TestHashBytes_Deterministic(t *testing.T) {
	a := hashBytes([]byte("hello"))
	b := hashBytes([]byte("hello"))
	if a != b {
		t.Errorf("expected deterministic hash, got %q and %q", a, b)
	}
	if len(a) != 64 { // sha256 hex = 64 chars
		t.Errorf("expected 64-char hex hash, got %d chars: %q", len(a), a)
	}
}

func TestHashBytes_DifferentInputs(t *testing.T) {
	a := hashBytes([]byte("hello"))
	b := hashBytes([]byte("world"))
	if a == b {
		t.Errorf("expected different hashes for different inputs, both = %q", a)
	}
}

func TestHashBytes_EmptyInput(t *testing.T) {
	// sha256("") has a well-known value; this guards against
	// accidentally returning "" for empty input.
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	got := hashBytes(nil)
	if got != want {
		t.Errorf("hashBytes(nil) = %q, want %q", got, want)
	}
}

// writeJournal creates a minimal journal file in a temp dir and
// returns the path. Used by the TUI IO tests below.
func writeJournal(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "2026-03-14.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write journal: %v", err)
	}
	return path
}

const sampleJournal = `---
title: 2026-03-14
---

# Daily Journal

## Todos

- [[2026-03-14]]
  - [ ] open item
  - [x] done item

## Notes

some notes here
`

// TestCheckExternalChanges_NoChangeLeavesModelUntouched verifies the
// happy path: when the on-disk content matches fileHash, nothing
// changes and no status message is set.
func TestCheckExternalChanges_NoChangeLeavesModelUntouched(t *testing.T) {
	path := writeJournal(t, sampleJournal)
	m, err := newTUIModel(path, "2026-03-14", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}
	originalStatus := m.status

	m.checkExternalChanges()

	if m.status != originalStatus {
		t.Errorf("status changed unexpectedly: %q -> %q", originalStatus, m.status)
	}
	if m.externalChanged {
		t.Error("externalChanged should be false when file is unchanged")
	}
}

// TestCheckExternalChanges_DirtySetsBlockedStatus verifies the
// "external change blocks save" behaviour: when the model is dirty and
// the file changed externally, the model records the conflict and
// does not silently reload (that would lose the user's edits).
func TestCheckExternalChanges_DirtySetsBlockedStatus(t *testing.T) {
	path := writeJournal(t, sampleJournal)
	m, err := newTUIModel(path, "2026-03-14", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}
	m.dirty = true
	m.status = "some prior status"

	// Mutate the file on disk.
	mutated := strings.Replace(sampleJournal, "- [ ] open item", "- [ ] changed externally", 1)
	if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
		t.Fatalf("mutate file: %v", err)
	}

	m.checkExternalChanges()

	if !m.externalChanged {
		t.Error("expected externalChanged = true")
	}
	if !strings.Contains(m.status, "External change detected") {
		t.Errorf("expected status to mention external change, got %q", m.status)
	}
	if !strings.Contains(m.status, "blocked") {
		t.Errorf("expected status to mention save blocked, got %q", m.status)
	}
}

// TestCheckExternalChanges_CleanReloadsFromDisk verifies the safe
// case: model is clean and the file changed externally, so we can
// reload without losing user work.
func TestCheckExternalChanges_CleanReloadsFromDisk(t *testing.T) {
	path := writeJournal(t, sampleJournal)
	m, err := newTUIModel(path, "2026-03-14", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}

	mutated := strings.Replace(sampleJournal, "- [ ] open item", "- [ ] changed externally", 1)
	if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
		t.Fatalf("mutate file: %v", err)
	}

	m.checkExternalChanges()

	if m.externalChanged {
		t.Error("expected externalChanged = false after clean reload")
	}
	if !strings.Contains(m.status, "External change detected and reloaded") {
		t.Errorf("expected reload status, got %q", m.status)
	}
	// After reload, the items should reflect the on-disk content.
	foundChanged := false
	for _, it := range m.items {
		if strings.Contains(it.item.Text, "changed externally") {
			foundChanged = true
			break
		}
	}
	if !foundChanged {
		t.Errorf("expected reloaded items to include the externally changed todo, got %d items", len(m.items))
	}
}

// TestCheckExternalChanges_MissingFileDoesNothing verifies the
// swallow-on-error behaviour: if the file disappeared (e.g. user
// deleted it from another shell), checkExternalChanges returns
// silently rather than panicking. The TUI keeps the in-memory state
// and the user can decide what to do.
func TestCheckExternalChanges_MissingFileDoesNothing(t *testing.T) {
	path := writeJournal(t, sampleJournal)
	m, err := newTUIModel(path, "2026-03-14", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}
	originalStatus := m.status

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove file: %v", err)
	}

	// Should not panic; should leave model in a sensible state.
	m.checkExternalChanges()

	if m.status != originalStatus {
		t.Errorf("status changed unexpectedly: %q -> %q", originalStatus, m.status)
	}
	if m.externalChanged {
		t.Error("externalChanged should be false when file is missing")
	}
}

// TestFlattenTodoItems verifies the recursive flattening with depth
// tracking, which feeds the TUI's flat item list from a hierarchical
// TodoItem tree.
func TestFlattenTodoItems(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		var out []tuiItem
		flattenTodoItems(nil, 0, &out, nil)
		if len(out) != 0 {
			t.Errorf("expected 0 items, got %d", len(out))
		}
	})

	t.Run("flat list of three items at depth 0", func(t *testing.T) {
		items := []*core.TodoItem{
			{Text: "a"},
			{Text: "b"},
			{Text: "c"},
		}
		var out []tuiItem
		flattenTodoItems(items, 0, &out, nil)
		if len(out) != 3 {
			t.Fatalf("expected 3 items, got %d", len(out))
		}
		for i, item := range out {
			if item.depth != 0 {
				t.Errorf("item %d: expected depth 0, got %d", i, item.depth)
			}
		}
	})

	t.Run("nested hierarchy preserves depth", func(t *testing.T) {
		items := []*core.TodoItem{
			{
				Text: "parent",
				SubItems: []*core.TodoItem{
					{
						Text: "child",
						SubItems: []*core.TodoItem{
							{Text: "grandchild"},
						},
					},
				},
			},
		}
		var out []tuiItem
		flattenTodoItems(items, 0, &out, nil)
		if len(out) != 3 {
			t.Fatalf("expected 3 flattened items, got %d", len(out))
		}
		wantDepths := []int{0, 1, 2}
		wantTexts := []string{"parent", "child", "grandchild"}
		for i, item := range out {
			if item.depth != wantDepths[i] {
				t.Errorf("item %d (%q): depth = %d, want %d", i, item.item.Text, item.depth, wantDepths[i])
			}
			if item.item.Text != wantTexts[i] {
				t.Errorf("item %d: text = %q, want %q", i, item.item.Text, wantTexts[i])
			}
		}
	})

	t.Run("nil entries in the slice are skipped", func(t *testing.T) {
		items := []*core.TodoItem{
			{Text: "a"},
			nil,
			{Text: "b"},
		}
		var out []tuiItem
		flattenTodoItems(items, 0, &out, nil)
		if len(out) != 2 {
			t.Errorf("expected 2 items (nil skipped), got %d", len(out))
		}
	})

	t.Run("starting depth is honoured", func(t *testing.T) {
		items := []*core.TodoItem{{Text: "x"}}
		var out []tuiItem
		flattenTodoItems(items, 5, &out, nil)
		if len(out) != 1 || out[0].depth != 5 {
			t.Errorf("expected depth 5, got %d", out[0].depth)
		}
	})
}

// TestNewTUIModel_RebuildsHashOnLoad ensures reloadFromDisk sets
// fileHash so a subsequent checkExternalChanges sees no diff.
func TestNewTUIModel_RebuildsHashOnLoad(t *testing.T) {
	path := writeJournal(t, sampleJournal)
	m, err := newTUIModel(path, "2026-03-14", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}
	if m.fileHash == "" {
		t.Error("expected fileHash to be set after newTUIModel")
	}
	if m.fileHash != hashBytes(mustRead(t, path)) {
		t.Error("fileHash should match the on-disk content")
	}
	if m.dirty {
		t.Error("expected dirty = false after fresh load")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

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
