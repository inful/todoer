package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inful/todoer/pkg/core"
)

func TestTUIModelDisplaysCarryoverWhenTodaySectionMissing(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "2026-03-16.md")

	content := `---
title: 2026-03-16
---

# Daily Journal

## Todos

- [[2026-03-15]]
  - [ ] carryover item

## Notes
`

	if err := os.WriteFile(journalPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write journal file: %v", err)
	}

	model, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("failed to init tui model: %v", err)
	}

	if len(model.items) != 1 {
		t.Fatalf("expected one visible carryover item, got %d", len(model.items))
	}

	if model.items[0].item.Text != "carryover item" {
		t.Fatalf("unexpected todo text: %s", model.items[0].item.Text)
	}
}

func TestTUIModelPicksLatestCarryoverRegardlessOfFileOrder(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "2026-03-20.md")

	// Today's section is missing. Two carryover days appear in
	// REVERSE chronological order in the file, so the most recent
	// is not the last entry. reloadFromDisk sorts the day
	// sections, so the carryover fallback picks the latest by date,
	// not the last in file order.
	content := `---
title: 2026-03-20
---

# Daily Journal

## Todos

- [[2026-03-19]]
  - [ ] yesterday
- [[2026-03-18]]
  - [ ] day before yesterday
- [[2026-03-15]]
  - [ ] last week

## Notes
`

	if err := os.WriteFile(journalPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write journal file: %v", err)
	}

	model, err := newTUIModel(journalPath, "2026-03-20", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("failed to init tui model: %v", err)
	}

	// The carryover fallback should show the latest day by date,
	// which is 2026-03-19.
	if model.displayDay == nil || model.displayDay.Date != "2026-03-19" {
		t.Fatalf("expected displayDay to be 2026-03-19, got %+v", model.displayDay)
	}
	if len(model.items) != 1 || model.items[0].item.Text != "yesterday" {
		t.Fatalf("expected the 2026-03-19 item to be shown, got %+v", model.items)
	}
}

func TestTUIModelDisplaysCarryoverWhenTodaySectionEmpty(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "2026-03-16.md")

	// Today has a section but it is empty; carryover has items. The view
	// should fall back to the carryover day rather than render an empty list.
	content := `---
title: 2026-03-16
---

# Daily Journal

## Todos

- [[2026-03-15]]
  - [ ] carryover item

- [[2026-03-16]]

## Notes
`

	if err := os.WriteFile(journalPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write journal file: %v", err)
	}

	model, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("failed to init tui model: %v", err)
	}

	if len(model.items) != 1 {
		t.Fatalf("expected one visible carryover item, got %d", len(model.items))
	}

	if model.items[0].item.Text != "carryover item" {
		t.Fatalf("unexpected todo text: %s", model.items[0].item.Text)
	}

	if !model.isReadOnlyView() {
		t.Fatalf("expected read-only view when today section is empty and carryover has items")
	}
}

func TestTUIInitReturnsTickCommand(t *testing.T) {
	m := tuiModel{}
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected Init to return a non-nil tick command")
	}
}

func TestTUIUpdateRoutesByMode(t *testing.T) {
	m := tuiModel{inputMode: true, inputText: "x"}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m2 := updated.(tuiModel)
	if m2.inputText != "xy" {
		t.Fatalf("expected inputText 'xy' via Update dispatch, got %q", m2.inputText)
	}
}

func TestTUIRemoveItemFromDays(t *testing.T) {
	target := &core.TodoItem{Text: "target"}
	other := &core.TodoItem{Text: "other"}
	sub := &core.TodoItem{Text: "subtarget"}
	day := &core.DaySection{
		Date: "2026-03-16",
		Items: []*core.TodoItem{
			other,
			{Text: "parent", SubItems: []*core.TodoItem{sub, {Text: "subother"}}},
		},
	}
	days := []*core.DaySection{day}

	// Remove a nested item.
	updated := core.RemoveItemFromDays(days, sub)
	if len(updated) != 1 {
		t.Fatalf("expected one day, got %d", len(updated))
	}
	if len(updated[0].Items) != 2 {
		t.Fatalf("expected parent + other to remain, got %d", len(updated[0].Items))
	}
	if len(updated[0].Items[1].SubItems) != 1 {
		t.Fatalf("expected one subitem left, got %d", len(updated[0].Items[1].SubItems))
	}

	// Remove a top-level item.
	updated2 := core.RemoveItemFromDays(updated, other)
	if len(updated2[0].Items) != 1 {
		t.Fatalf("expected one item left, got %d", len(updated2[0].Items))
	}

	// Remove a missing target is a no-op.
	updated3 := core.RemoveItemFromDays(updated2, target)
	if len(updated3[0].Items) != 1 {
		t.Fatalf("expected unchanged items, got %d", len(updated3[0].Items))
	}

	// Nil days and nil items in the slice are tolerated.
	updated4 := core.RemoveItemFromDays([]*core.DaySection{nil, day}, target)
	if len(updated4) != 2 {
		t.Fatalf("expected nil-day pass-through, got %d", len(updated4))
	}
}

func TestTUIUpdateTickMsg(t *testing.T) {
	m := tuiModel{}
	_, cmd := m.Update(tuiTickMsg{})
	if cmd == nil {
		t.Fatalf("expected tick message to schedule next tick")
	}
}

func TestTUIUpdateNoMsg(t *testing.T) {
	m := tuiModel{}
	_, cmd := m.Update(struct{}{})
	if cmd != nil {
		t.Fatalf("expected nil cmd for unknown message type")
	}
}

func TestTUITickCmd(t *testing.T) {
	cmd := tuiTickCmd()
	if cmd == nil {
		t.Fatalf("expected tuiTickCmd to return a non-nil command")
	}
}

func TestTUIPickInitialDisplayDay_Empty(t *testing.T) {
	m := tuiModel{journal: &core.TodoJournal{Days: []*core.DaySection{}}}
	if got := m.pickInitialDisplayDay(); got != nil {
		t.Fatalf("expected nil for empty journal")
	}
}

func TestTUIRemoveItemRecursive_NilSafety(t *testing.T) {
	items, removed := core.RemoveItemRecursive(nil, &core.TodoItem{Text: "x"})
	if removed {
		t.Fatalf("expected no removal on nil items")
	}
	if items != nil {
		t.Fatalf("expected nil items to be returned as-is")
	}

	items, removed = core.RemoveItemRecursive([]*core.TodoItem{nil}, &core.TodoItem{Text: "x"})
	_ = items
	if removed {
		t.Fatalf("expected no removal when only nil items are present")
	}
}
