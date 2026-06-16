package main

import (
	"slices"
	"strings"

	"github.com/inful/todoer/pkg/core"
)

func (m *tuiModel) refreshItems() {
	m.items = make([]tuiItem, 0)
	if m.displayDay == nil {
		return
	}
	flattenTodoItems(m.displayDay.Items, 0, &m.items)
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

// isReadOnlyView reports whether the currently displayed day is not today's
// section. When the model falls back to showing a carryover day, edits to those
// items would silently mutate a different journal; the carryover view is
// read-only and the user is asked to run `new` to bring items into today.
func (m *tuiModel) isReadOnlyView() bool {
	return m.displayDay != m.todayDay
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

func flattenTodoItems(items []*core.TodoItem, depth int, out *[]tuiItem) {
	for _, item := range items {
		if item == nil {
			continue
		}
		*out = append(*out, tuiItem{item: item, depth: depth})
		flattenTodoItems(item.SubItems, depth+1, out)
	}
}
