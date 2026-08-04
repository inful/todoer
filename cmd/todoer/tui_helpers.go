package main

import (
	"slices"
	"strings"

	"github.com/inful/todoer/pkg/core"
)

func (m *tuiModel) refreshItems() {
	m.items = make([]tuiItem, 0)
	// Always show items from all days. Today's items come first
	// (no day label), followed by carryover items from previous
	// days in reverse-chronological order (newest first, with a
	// date prefix). This way the user always sees every pending
	// todo, regardless of whether today has items, and can mark
	// any item as complete from the TUI.
	//
	// This replaces the earlier "carryover view" mode where carryover
	// items were only shown when today was empty. The old mode hid
	// items from previous days whenever the user had anything in
	// today, which made those items unreachable from the TUI.
	if m.todayDay != nil && len(m.todayDay.Items) > 0 {
		flattenTodoItems(m.todayDay.Items, 0, &m.items, m.todayDay)
	}
	for _, day := range slices.Backward(m.journal.Days) {
		if day != nil && day != m.todayDay && len(day.Items) > 0 {
			flattenTodoItems(day.Items, 0, &m.items, day)
		}
	}
	filtered := m.filteredItems()
	if m.selected >= len(filtered) {
		m.selected = max(0, len(filtered)-1)
	}
}

// pickInitialDisplayDay initializes the sticky displayDay mode
// marker on first load: today's section when it has items,
// otherwise the most recent non-empty day. The value persists
// until the next reload and is consulted by isCarryoverView() to
// choose the post-add status message wording.
func (m *tuiModel) pickInitialDisplayDay() *core.DaySection {
	if m.todayDay != nil && len(m.todayDay.Items) > 0 {
		return m.todayDay
	}
	for _, day := range slices.Backward(m.journal.Days) {
		if day != nil && len(day.Items) > 0 {
			return day
		}
	}
	return nil
}

// isCarryoverView reports whether displayDay is pinned to a
// previous (non-today) day. It does not change which items are
// rendered — the view always shows items from every day, and
// toggles/deletes always write back to the correct day in the
// journal. It is only consulted to choose the post-add status
// message wording in updateInputMode.
func (m *tuiModel) isCarryoverView() bool {
	return m.displayDay != m.todayDay
}

// emptyStateMessage returns the placeholder text shown in the
// body of the view when the filtered item list is empty. The
// view always shows items from all days, so a single generic
// message covers all cases.
func (m *tuiModel) emptyStateMessage() string {
	return "(No todos)"
}

func (m tuiModel) filteredItems() []tuiItem {
	if strings.TrimSpace(m.filterQuery) == "" {
		return m.items
	}

	needle := strings.ToLower(strings.TrimSpace(m.filterQuery))
	filtered := make([]tuiItem, 0, len(m.items))
	for _, entry := range m.items {
		if strings.Contains(strings.ToLower(entry.item.Text), needle) {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

func flattenTodoItems(items []*core.TodoItem, depth int, out *[]tuiItem, day *core.DaySection) {
	for _, item := range items {
		if item == nil {
			continue
		}
		*out = append(*out, tuiItem{item: item, depth: depth, day: day})
		flattenTodoItems(item.SubItems, depth+1, out, day)
	}
}
