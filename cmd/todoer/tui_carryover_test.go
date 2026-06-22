package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inful/todoer/pkg/core"
)

// TestCarryoverView_ShowsItemsFromAllDays pins the contract that
// when today is empty, the carryover view shows items from ALL
// days in the journal — not just the most recent non-empty one.
// This is the fix for the multi-day TUI visibility bug: previously
// items from days before the most-recent were completely hidden.
func TestCarryoverView_ShowsItemsFromAllDays(t *testing.T) {
	// Three days, all with items. Today (2026-03-16) is empty.
	day15 := &core.DaySection{Date: "2026-03-15", Items: []*core.TodoItem{
		{Text: "task A"},
	}}
	day14 := &core.DaySection{Date: "2026-03-14", Items: []*core.TodoItem{
		{Text: "task B"},
		{Text: "task C"},
	}}
	day13 := &core.DaySection{Date: "2026-03-13", Items: []*core.TodoItem{
		{Text: "task D"},
	}}
	m := tuiModel{
		today: "2026-03-16",
		journal: &core.TodoJournal{Days: []*core.DaySection{
			day15, day14, day13,
		}},
		todayDay: nil,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	// All four items from all three days should be visible.
	if len(m.items) != 4 {
		t.Fatalf("expected 4 items from all 3 days, got %d", len(m.items))
	}

	stripped := stripANSI(m.View())
	for _, text := range []string{"task A", "task B", "task C", "task D"} {
		if !strings.Contains(stripped, text) {
			t.Errorf("expected view to contain %q, got:\n%s", text, stripped)
		}
	}
}

// TestCarryoverView_ShowsDayLabels pins the contract that items
// in the carryover view are labelled with their date so the user
// knows which day each item is from. Today view does not show day
// labels (all items are from today).
func TestCarryoverView_ShowsDayLabels(t *testing.T) {
	day15 := &core.DaySection{Date: "2026-03-15", Items: []*core.TodoItem{
		{Text: "task A"},
	}}
	day14 := &core.DaySection{Date: "2026-03-14", Items: []*core.TodoItem{
		{Text: "task B"},
	}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{day15, day14}},
		todayDay: nil,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	stripped := stripANSI(m.View())
	// Each item should be prefixed with its date.
	if !strings.Contains(stripped, "(2026-03-15) task A") {
		t.Errorf("expected day label for task A, got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "(2026-03-14) task B") {
		t.Errorf("expected day label for task B, got:\n%s", stripped)
	}
}

// TestCarryoverView_AllowsToggleOnOldDay verifies that pressing
// space/x in the carryover view toggles the item's Completed
// flag and writes the change back to the correct day in the
// journal. The change must NOT be silently dropped.
func TestCarryoverView_AllowsToggleOnOldDay(t *testing.T) {
	item := &core.TodoItem{Text: "old task", Completed: false}
	oldDay := &core.DaySection{Date: "2026-03-15", Items: []*core.TodoItem{item}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{oldDay}},
		todayDay: nil,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	if !m.isCarryoverView() {
		t.Fatalf("expected carryover view, got today view")
	}
	if item.Completed {
		t.Fatalf("item should start uncompleted")
	}

	updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m2 := updated.(tuiModel)

	if !item.Completed {
		t.Errorf("item in carryover day should have been toggled, got Completed=%v", item.Completed)
	}
	if m2.status == "Cannot edit carryover items" {
		t.Errorf("carryover view should not block edits, got status=%q", m2.status)
	}
	if !m2.dirty {
		t.Errorf("model should be dirty after toggling in carryover view")
	}
}

// TestCarryoverView_AllowsDeleteOnOldDay verifies that pressing
// 'd' in the carryover view removes the item from the correct
// day in the journal.
func TestCarryoverView_AllowsDeleteOnOldDay(t *testing.T) {
	item := &core.TodoItem{Text: "old task", Completed: false}
	oldDay := &core.DaySection{Date: "2026-03-15", Items: []*core.TodoItem{item}}
	m := tuiModel{
		today:    "2026-03-16",
		journal:  &core.TodoJournal{Days: []*core.DaySection{oldDay}},
		todayDay: nil,
	}
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()

	updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m2 := updated.(tuiModel)

	if len(oldDay.Items) != 0 {
		t.Errorf("item should have been removed from carryover day, got %d items", len(oldDay.Items))
	}
	if m2.status == "Cannot edit carryover items" {
		t.Errorf("carryover view should not block deletes, got status=%q", m2.status)
	}
	if !m2.dirty {
		t.Errorf("model should be dirty after deleting in carryover view")
	}
}
