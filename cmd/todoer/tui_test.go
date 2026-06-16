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

func TestTUIToggleBlockedOnCarryoverDay(t *testing.T) {
	item := &core.TodoItem{Text: "carryover item", Completed: false}
	carryoverDay := &core.DaySection{Date: "2026-03-15", Items: []*core.TodoItem{item}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{carryoverDay}},
		todayDay: nil,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	if len(m.items) != 1 {
		t.Fatalf("expected one visible carryover item, got %d", len(m.items))
	}
	if !m.isReadOnlyView() {
		t.Fatalf("expected view to be read-only (carryover fallback)")
	}

	updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m2 := updated.(tuiModel)

	if m2.status != "Cannot edit carryover items" {
		t.Fatalf("expected read-only status, got: %q", m2.status)
	}
	if item.Completed {
		t.Fatalf("carryover item should not have been toggled")
	}
	if m2.dirty {
		t.Fatalf("model should not be dirty after blocked edit")
	}
}

func TestTUIDeleteBlockedOnCarryoverDay(t *testing.T) {
	item := &core.TodoItem{Text: "carryover item", Completed: false}
	carryoverDay := &core.DaySection{Date: "2026-03-15", Items: []*core.TodoItem{item}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{carryoverDay}},
		todayDay: nil,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	if len(m.items) != 1 {
		t.Fatalf("expected one visible carryover item, got %d", len(m.items))
	}

	updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m2 := updated.(tuiModel)

	if m2.status != "Cannot edit carryover items" {
		t.Fatalf("expected read-only status, got: %q", m2.status)
	}
	if len(carryoverDay.Items) != 1 {
		t.Fatalf("carryover day items should be untouched, got %d items", len(carryoverDay.Items))
	}
	if m2.dirty {
		t.Fatalf("model should not be dirty after blocked edit")
	}
}

func TestTUIToggleWorksOnTodayDay(t *testing.T) {
	item := &core.TodoItem{Text: "today item", Completed: false}
	todayDay := &core.DaySection{Date: "2026-03-16", Items: []*core.TodoItem{item}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{todayDay}},
		todayDay: todayDay,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	if len(m.items) != 1 {
		t.Fatalf("expected one item, got %d", len(m.items))
	}
	if m.isReadOnlyView() {
		t.Fatalf("expected view to be editable (today's day)")
	}

	updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m2 := updated.(tuiModel)

	if !item.Completed {
		t.Fatalf("today item should have been toggled to completed")
	}
	if m2.status != "Todo toggled" {
		t.Fatalf("expected 'Todo toggled' status, got: %q", m2.status)
	}
	if !m2.dirty {
		t.Fatalf("model should be dirty after toggle")
	}
}

func TestTUIDeleteWorksOnTodayDay(t *testing.T) {
	item := &core.TodoItem{Text: "today item", Completed: false}
	todayDay := &core.DaySection{Date: "2026-03-16", Items: []*core.TodoItem{item}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{todayDay}},
		todayDay: todayDay,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	if len(m.items) != 1 {
		t.Fatalf("expected one item, got %d", len(m.items))
	}

	updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m2 := updated.(tuiModel)

	if len(todayDay.Items) != 0 {
		t.Fatalf("today item should have been removed, got %d items", len(todayDay.Items))
	}
	if m2.status != "Todo deleted" {
		t.Fatalf("expected 'Todo deleted' status, got: %q", m2.status)
	}
	if !m2.dirty {
		t.Fatalf("model should be dirty after delete")
	}
}

func TestTUIAddInCarryoverViewKeepsCarryoverVisible(t *testing.T) {
	carryoverItem := &core.TodoItem{Text: "carryover item", Completed: false}
	carryoverDay := &core.DaySection{Date: "2026-03-15", Items: []*core.TodoItem{carryoverItem}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{carryoverDay}},
		todayDay: nil,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	if len(m.items) != 1 {
		t.Fatalf("expected one visible carryover item, got %d", len(m.items))
	}
	if !m.isReadOnlyView() {
		t.Fatalf("expected initial view to be the carryover fallback")
	}

	// Enter input mode and add a new todo for today.
	m.inputMode = true
	m.inputText = "new today todo"
	updated, _ := m.updateInputMode(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(tuiModel)

	// The carryover item must still be visible: the new todo is in
	// today's section, the view stays on the carryover day.
	if len(m2.items) != 1 {
		t.Fatalf("expected the carryover item to remain visible after add, got %d items", len(m2.items))
	}
	if m2.items[0].item.Text != "carryover item" {
		t.Fatalf("expected the carryover item to be the visible one, got %q", m2.items[0].item.Text)
	}

	// todayDay is set and has the new item, but displayDay is still the carryover.
	if m2.todayDay == nil {
		t.Fatalf("expected todayDay to be created for the new item")
	}
	if len(m2.todayDay.Items) != 1 || m2.todayDay.Items[0].Text != "new today todo" {
		t.Fatalf("expected today's section to contain the new todo, got %+v", m2.todayDay.Items)
	}
	if m2.displayDay != carryoverDay {
		t.Fatalf("expected displayDay to remain the carryover day after add, got %p (want %p)", m2.displayDay, carryoverDay)
	}
	if !m2.dirty {
		t.Fatalf("model should be dirty after add")
	}
	if !strings.HasPrefix(m2.status, "Added to today") {
		t.Fatalf("expected add status to explain the view, got %q", m2.status)
	}
}

func TestTUIAddInTodayViewShowsNewTodo(t *testing.T) {
	existing := &core.TodoItem{Text: "existing today item", Completed: false}
	todayDay := &core.DaySection{Date: "2026-03-16", Items: []*core.TodoItem{existing}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{todayDay}},
		todayDay: todayDay,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	if len(m.items) != 1 {
		t.Fatalf("expected one existing item, got %d", len(m.items))
	}

	// Add a new todo in today view.
	m.inputMode = true
	m.inputText = "second today item"
	updated, _ := m.updateInputMode(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(tuiModel)

	if len(m2.items) != 2 {
		t.Fatalf("expected the new todo to be visible in today view, got %d items", len(m2.items))
	}
	if m2.items[1].item.Text != "second today item" {
		t.Fatalf("expected the new item to be the second one, got %q", m2.items[1].item.Text)
	}
	if m2.displayDay != todayDay {
		t.Fatalf("expected displayDay to stay on today, got %p (want %p)", m2.displayDay, todayDay)
	}
	if m2.status != "Todo added" {
		t.Fatalf("expected plain 'Todo added' status, got %q", m2.status)
	}
}
