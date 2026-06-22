package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/inful/todoer/pkg/core"
)

func (m *tuiModel) refreshItems() {
	m.items = make([]tuiItem, 0)
	// In the carryover view (today is empty, display is a previous
	// day), flatten items from ALL days so the user can see and
	// mark complete todos from every previous day — not just the
	// most recent one. Each item carries a reference to its day
	// so the carryover view can render day labels and so toggle
	// and delete write the change back to the correct day.
	if m.isCarryoverView() {
		for _, day := range m.journal.Days {
			if day != nil {
				flattenTodoItems(day.Items, 0, &m.items, day)
			}
		}
	} else {
		if m.displayDay != nil {
			flattenTodoItems(m.displayDay.Items, 0, &m.items, m.displayDay)
		}
	}
	filtered := m.filteredItems()
	if m.selected >= len(filtered) {
		m.selected = max(0, len(filtered)-1)
	}
}

// pickInitialDisplayDay chooses which day the model should show on first
// load: today's section when it has items, otherwise the most recent
// non-empty day. The result is sticky for the lifetime of the model until
// the next reload, which is what keeps the carryover view visible after
// the user adds an item to today's section.
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

// isCarryoverView reports whether the model is showing a previous
// day (the carryover fallback) rather than today's section. The
// carryover view shows items from ALL days (not just the most
// recent) and is editable: the user can mark items complete or
// delete them, and the change is written back to the correct day
// in the journal. New todos still go to today.
func (m *tuiModel) isCarryoverView() bool {
	return m.displayDay != m.todayDay
}

// emptyStateMessage returns the placeholder text shown in the body of
// the view when the filtered item list is empty. The wording is
// section-relative: the carryover view names the carryover day
// (so the user knows they are looking at a previous day, not
// today), the today view names today's section, and the degenerate
// no-display-day state falls back to a generic message.
func (m *tuiModel) emptyStateMessage() string {
	switch {
	case m.displayDay == nil:
		return "(No todos)"
	case m.isCarryoverView():
		return fmt.Sprintf("(No items in [[%s]])", m.displayDay.Date)
	default:
		return "(No todos in today's section)"
	}
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
