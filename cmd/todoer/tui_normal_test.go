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

// Note: the previous TestTUIToggleBlockedOnCarryoverDay and
// TestTUIDeleteBlockedOnCarryoverDay tests asserted that the
// carryover view was read-only. The carryover view is now
// editable: the user can toggle and delete items from any
// previous day, and the change is written back to the correct
// day in the journal. See cmd/todoer/tui_carryover_test.go
// for the new tests that pin this behaviour.

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
	if m.isCarryoverView() {
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
	if !m.isCarryoverView() {
		t.Fatalf("expected initial view to be the carryover fallback")
	}

	// Enter input mode and add a new todo for today.
	m.inputMode = true
	m.inputText = "new today todo"
	updated, _ := m.updateInputMode(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(tuiModel)

	// The carryover item must still be visible AND the new today
	// item is also visible (the carryover view shows items from
	// all days, including today). The display day is sticky on
	// the carryover day.
	if len(m2.items) != 2 {
		t.Fatalf("expected carryover + new today items visible after add, got %d items", len(m2.items))
	}
	got := make(map[string]bool)
	for _, it := range m2.items {
		got[it.item.Text] = true
	}
	if !got["carryover item"] {
		t.Errorf("expected the carryover item to remain visible, got %+v", m2.items)
	}
	if !got["new today todo"] {
		t.Errorf("expected the new today item to be visible, got %+v", m2.items)
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
