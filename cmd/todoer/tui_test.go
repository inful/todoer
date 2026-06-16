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

func TestTUIUpdateInputMode_BackspaceAndRunes(t *testing.T) {
	m := tuiModel{inputMode: true}

	updated, _ := m.updateInputMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a', 'b'}})
	m2 := updated.(tuiModel)
	if m2.inputText != "ab" {
		t.Fatalf("expected inputText 'ab', got %q", m2.inputText)
	}

	updated, _ = m2.updateInputMode(tea.KeyMsg{Type: tea.KeyBackspace})
	m3 := updated.(tuiModel)
	if m3.inputText != "a" {
		t.Fatalf("expected backspace to drop last char, got %q", m3.inputText)
	}

	// Backspace on empty input is a no-op.
	updated, _ = m3.updateInputMode(tea.KeyMsg{Type: tea.KeyBackspace})
	m4 := updated.(tuiModel)
	updated, _ = m4.updateInputMode(tea.KeyMsg{Type: tea.KeyBackspace})
	m5 := updated.(tuiModel)
	if m5.inputText != "" {
		t.Fatalf("expected inputText to stay empty, got %q", m5.inputText)
	}
}

func TestTUIUpdateInputMode_EscCancels(t *testing.T) {
	m := tuiModel{inputMode: true, inputText: "draft"}
	updated, _ := m.updateInputMode(tea.KeyMsg{Type: tea.KeyEscape})
	m2 := updated.(tuiModel)

	if m2.inputMode {
		t.Fatalf("expected input mode to be cleared by esc")
	}
	if m2.inputText != "" {
		t.Fatalf("expected input text to be cleared by esc, got %q", m2.inputText)
	}
	if !strings.Contains(m2.status, "cancelled") {
		t.Fatalf("expected cancel status, got %q", m2.status)
	}
}

func TestTUIUpdateNormalMode_QuitCleanAndDirty(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "2026-03-16.md")
	createTestFile(t, journalPath, "---\ntitle: 2026-03-16\n---\n\n# Journal\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a\n  - [ ] b\n\n## Notes\n")

	// Clean quit: no save expected.
	m, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}
	updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	mClean := updated.(tuiModel)
	if mClean.dirty {
		t.Fatalf("clean model should not be dirty after q")
	}
}

func TestTUIUpdateNormalMode_NavigationAndFilter(t *testing.T) {
	day := &core.DaySection{Date: "2026-03-16", Items: []*core.TodoItem{
		{Text: "alpha"},
		{Text: "beta"},
		{Text: "gamma"},
	}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{day}},
		todayDay: day,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	// j moves down
	updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m2 := updated.(tuiModel)
	if m2.selected != 1 {
		t.Fatalf("expected selected=1 after j, got %d", m2.selected)
	}

	// down arrow also moves down
	updated, _ = m2.updateNormalMode(tea.KeyMsg{Type: tea.KeyDown})
	m3 := updated.(tuiModel)
	if m3.selected != 2 {
		t.Fatalf("expected selected=2 after down, got %d", m3.selected)
	}

	// j at the bottom clamps
	updated, _ = m3.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m4 := updated.(tuiModel)
	if m4.selected != 2 {
		t.Fatalf("expected selected to stay at 2, got %d", m4.selected)
	}

	// k moves up
	updated, _ = m4.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m5 := updated.(tuiModel)
	if m5.selected != 1 {
		t.Fatalf("expected selected=1 after k, got %d", m5.selected)
	}

	// k at the top clamps
	updated, _ = m5.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated, _ = updated.(tuiModel).updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m6 := updated.(tuiModel)
	if m6.selected != 0 {
		t.Fatalf("expected selected to stay at 0, got %d", m6.selected)
	}

	// / enters filter mode
	updated, _ = m6.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m7 := updated.(tuiModel)
	if !m7.filterMode {
		t.Fatalf("expected filter mode to be active after /")
	}

	// type a filter and apply
	updated, _ = m7.updateFilterMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	updated, _ = updated.(tuiModel).updateFilterMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	updated, _ = updated.(tuiModel).updateFilterMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	updated, _ = updated.(tuiModel).updateFilterMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	updated, _ = updated.(tuiModel).updateFilterMode(tea.KeyMsg{Type: tea.KeyEnter})
	m8 := updated.(tuiModel)
	if m8.filterMode {
		t.Fatalf("expected filter mode to be off after enter")
	}
	if !strings.Contains(m8.status, "Filter applied") {
		t.Fatalf("expected filter applied status, got %q", m8.status)
	}
	if got := m8.filterQuery; got != "beta" {
		t.Fatalf("expected filterQuery 'beta', got %q", got)
	}

	// c clears the filter
	updated, _ = m8.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m9 := updated.(tuiModel)
	if m9.filterQuery != "" {
		t.Fatalf("expected filterQuery to be cleared, got %q", m9.filterQuery)
	}
	if !strings.Contains(m9.status, "Filter cleared") {
		t.Fatalf("expected filter cleared status, got %q", m9.status)
	}
}

func TestTUIUpdateFilterMode_EscBackspaceRunes(t *testing.T) {
	m := tuiModel{filterMode: true}

	updated, _ := m.updateFilterMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x', 'y'}})
	m2 := updated.(tuiModel)
	if m2.filterQuery != "xy" {
		t.Fatalf("expected filterQuery 'xy', got %q", m2.filterQuery)
	}

	updated, _ = m2.updateFilterMode(tea.KeyMsg{Type: tea.KeyBackspace})
	m3 := updated.(tuiModel)
	if m3.filterQuery != "x" {
		t.Fatalf("expected filterQuery 'x' after backspace, got %q", m3.filterQuery)
	}

	updated, _ = m3.updateFilterMode(tea.KeyMsg{Type: tea.KeyEscape})
	m4 := updated.(tuiModel)
	if m4.filterMode {
		t.Fatalf("expected filter mode to clear on esc")
	}
	if !strings.Contains(m4.status, "cancelled") {
		t.Fatalf("expected cancel status, got %q", m4.status)
	}
}

func TestTUIUpdateFilterMode_EnterWithEmptyQuery(t *testing.T) {
	m := tuiModel{filterMode: true, filterQuery: ""}
	updated, _ := m.updateFilterMode(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(tuiModel)
	if m2.filterMode {
		t.Fatalf("expected filter mode to be off after enter")
	}
	if !strings.Contains(m2.status, "cleared") {
		t.Fatalf("expected cleared status, got %q", m2.status)
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

func TestTUIViewContainsKeyState(t *testing.T) {
	day := &core.DaySection{Date: "2026-03-16", Items: []*core.TodoItem{
		{Text: "first", Completed: false},
		{Text: "second", Completed: true},
	}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{day}},
		todayDay: day,
		status:   "test-status",
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	view := m.View()
	if !strings.Contains(view, "test-status") {
		t.Fatalf("expected status to appear in view, got:\n%s", view)
	}
	if !strings.Contains(view, "first") || !strings.Contains(view, "second") {
		t.Fatalf("expected items to appear in view, got:\n%s", view)
	}
	if !strings.Contains(view, "[x]") {
		t.Fatalf("expected completed marker in view, got:\n%s", view)
	}
}

func TestTUIViewEmptyAndExternalChanged(t *testing.T) {
	m := tuiModel{
		today:   "2026-03-16",
		journal: &core.TodoJournal{Days: []*core.DaySection{}},
		status:  "ready",
	}
	// Force empty view.
	m.displayDay = nil
	m.refreshItems()

	view := m.View()
	if !strings.Contains(view, "No todos") {
		t.Fatalf("expected empty placeholder in view, got:\n%s", view)
	}
	if !strings.Contains(view, "ready") {
		t.Fatalf("expected status in view, got:\n%s", view)
	}

	// dirty + external-changed should show both states.
	m.dirty = true
	m.externalChanged = true
	view2 := m.View()
	if !strings.Contains(view2, "dirty") || !strings.Contains(view2, "external-changed") {
		t.Fatalf("expected dirty + external-changed markers, got:\n%s", view2)
	}
}

func TestTUIEmptyStateMessage(t *testing.T) {
	// No display day at all -> generic placeholder.
	m := tuiModel{today: "2026-03-16"}
	if got := m.emptyStateMessage(); got != "(No todos)" {
		t.Fatalf("expected generic 'No todos', got %q", got)
	}

	// Today view (displayDay == todayDay) -> today's section wording.
	todayDay := &core.DaySection{Date: "2026-03-16", Items: nil}
	m = tuiModel{
		today:    "2026-03-16",
		todayDay: todayDay,
	}
	m.displayDay = todayDay
	if got := m.emptyStateMessage(); got != "(No todos in today's section)" {
		t.Fatalf("expected today's section wording, got %q", got)
	}

	// Carryover view (displayDay != todayDay) -> names the carryover
	// day so the user knows they are not looking at today.
	carryoverDay := &core.DaySection{Date: "2026-03-15", Items: nil}
	m = tuiModel{
		today:    "2026-03-16",
		todayDay: todayDay,
	}
	m.displayDay = carryoverDay
	if got := m.emptyStateMessage(); got != "(No items in [[2026-03-15]])" {
		t.Fatalf("expected carryover-day wording, got %q", got)
	}
}

func TestTUIViewEmptyInCarryoverView(t *testing.T) {
	// End-to-end: construct a model where the display day is a
	// carryover day with no items, render the view, and assert the
	// view does not claim 'today's section'.
	todayDay := &core.DaySection{Date: "2026-03-16", Items: nil}
	carryoverDay := &core.DaySection{Date: "2026-03-15", Items: nil}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{carryoverDay, todayDay}},
		todayDay: todayDay,
	}
	m.displayDay = carryoverDay
	m.refreshItems()

	view := m.View()
	if strings.Contains(view, "today's section") {
		t.Fatalf("carryover view should not mention 'today's section', got:\n%s", view)
	}
	if !strings.Contains(view, "No items in [[2026-03-15]]") {
		t.Fatalf("expected carryover-day wording in view, got:\n%s", view)
	}
}

func TestTUIViewInputAndFilterModes(t *testing.T) {
	m := tuiModel{
		inputMode:  true,
		inputText:  "draft",
		filterMode: false,
	}
	view := m.View()
	if !strings.Contains(view, "Add todo") {
		t.Fatalf("expected add-todo label, got:\n%s", view)
	}
	if !strings.Contains(view, "draft") {
		t.Fatalf("expected input text in view, got:\n%s", view)
	}

	m.inputMode = false
	m.filterMode = true
	m.filterQuery = "needle"
	view2 := m.View()
	if !strings.Contains(view2, "Filter todos") {
		t.Fatalf("expected filter-todos label, got:\n%s", view2)
	}
	if !strings.Contains(view2, "needle") {
		t.Fatalf("expected filter query in view, got:\n%s", view2)
	}
}

func TestTUIUpdateNormalMode_SaveAndReload(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "2026-03-16.md")
	createTestFile(t, journalPath, "---\ntitle: 2026-03-16\n---\n\n# J\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a\n\n## Notes\n")

	m, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}

	// s on a clean model: status flips to "Saved" but nothing is written.
	updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	mClean := updated.(tuiModel)
	if mClean.status != "Saved" {
		t.Fatalf("expected status 'Saved' on clean save, got %q", mClean.status)
	}

	// s on a dirty model: writes the file and clears dirty.
	mClean.dirty = true
	mClean.items[0].item.Completed = true
	updated, _ = mClean.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	mSaved := updated.(tuiModel)
	if mSaved.dirty {
		t.Fatalf("expected dirty=false after save")
	}
	after, _ := os.ReadFile(journalPath)
	if !strings.Contains(string(after), "[x] a") {
		t.Fatalf("expected save to persist the toggle, got:\n%s", string(after))
	}

	// s failure -> status shows the error.
	mSaved.dirty = true
	if err := os.Chmod(journalPath, 0o000); err != nil {
		t.Skipf("chmod not supported in this environment: %v", err)
	}
	updated, _ = mSaved.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	mErr := updated.(tuiModel)
	if !strings.Contains(mErr.status, "Save failed") {
		t.Fatalf("expected Save failed status, got %q", mErr.status)
	}
	// Restore perms for the reload test.
	_ = os.Chmod(journalPath, 0o644)

	// r reloads from disk on success.
	mErr.dirty = false
	updated, _ = mErr.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	mReload := updated.(tuiModel)
	if mReload.status != "Reloaded from disk" {
		t.Fatalf("expected Reloaded status, got %q", mReload.status)
	}
}

func TestTUIUpdateNormalMode_CtrlCAndQuitDirty(t *testing.T) {
	tempDir := t.TempDir()
	journalPath := filepath.Join(tempDir, "2026-03-16.md")
	createTestFile(t, journalPath, "---\ntitle: 2026-03-16\n---\n\n# J\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] a\n\n## Notes\n")

	m, err := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	if err != nil {
		t.Fatalf("newTUIModel: %v", err)
	}
	m.dirty = true
	m.items[0].item.Completed = true

	// q on a dirty model should save before quitting.
	updated, cmd := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m2 := updated.(tuiModel)
	if cmd == nil {
		t.Fatalf("expected quit command on q")
	}
	if m2.dirty {
		t.Fatalf("expected dirty=false after q-save")
	}
	after, _ := os.ReadFile(journalPath)
	if !strings.Contains(string(after), "[x] a") {
		t.Fatalf("expected q to persist the toggle, got:\n%s", string(after))
	}

	// ctrl+c quits without saving.
	m3, _ := newTUIModel(journalPath, "2026-03-16", &Config{TodosHeader: core.TodosHeader})
	m3.dirty = true
	updated, cmd = m3.updateNormalMode(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatalf("expected quit command on ctrl+c")
	}
	mCtrlC := updated.(tuiModel)
	if !mCtrlC.dirty {
		t.Fatalf("expected dirty to remain true on ctrl+c (no save)")
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
