package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inful/todoer/pkg/core"
)

func TestTUIQuitSavesDirtyChanges(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "2026-03-14.md")

	content := `---
title: 2026-03-14
---

# Daily Journal

## Todos

- [[2026-03-14]]
  - [ ] test item

## Notes
`

	if err := os.WriteFile(journalPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write journal file: %v", err)
	}

	model, err := newTUIModel(journalPath, "2026-03-14", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("failed to init tui model: %v", err)
	}

	if len(model.items) != 1 {
		t.Fatalf("expected one item, got %d", len(model.items))
	}

	model.items[0].item.Completed = true
	model.dirty = true

	updatedModel, cmd := model.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}

	updated := updatedModel.(tuiModel)
	if updated.dirty {
		t.Fatal("expected model to be clean after quit-save")
	}

	after, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatalf("failed to read journal file: %v", err)
	}

	if !strings.Contains(string(after), "- [x] test item") {
		t.Fatalf("expected toggled item to be persisted, got:\n%s", string(after))
	}
}

func TestTUIFilterMatchesTodoText(t *testing.T) {
	m := tuiModel{
		items: []tuiItem{
			{item: &core.TodoItem{Text: "review PR", Completed: false}},
			{item: &core.TodoItem{Text: "buy milk", Completed: false}},
			{item: &core.TodoItem{Text: "write tests", Completed: false}},
		},
		filterQuery: "pr",
	}

	filtered := m.filteredItems()
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered todo, got %d", len(filtered))
	}

	if filtered[0].item.Text != "review PR" {
		t.Fatalf("unexpected filtered todo text: %s", filtered[0].item.Text)
	}
}
