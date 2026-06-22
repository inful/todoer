package main

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inful/todoer/pkg/core"
)

// makeScrollItems creates count todo items with unique, non-overlapping
// text markers. The "task-%02d" format guarantees that strings.Contains
// for one task name never accidentally matches a different task name
// (e.g. "task-01" does not appear as a substring of "task-10" because
// the 6th character differs: '0' vs '1').
func makeScrollItems(count int) []tuiItem {
	items := make([]tuiItem, count)
	for i := range items {
		items[i] = tuiItem{item: &core.TodoItem{Text: fmt.Sprintf("task-%02d", i+1)}}
	}
	return items
}

// TestViewItems_RendersOnlyVisibleWindow pins the viewport
// scrolling contract: when the filtered list is longer than the
// terminal can show, viewItems renders only the slice
// [offset, offset+visibleCount), not the full list. Items outside
// the window must not appear in the output, and the cursor
// (selected) must always be inside the visible window.
//
// Without this, a 30-item list in a 20-line terminal produces a
// ~30-line View output. Bubble Tea then clips the output to fit
// the terminal, and because items are appended top-to-bottom, the
// bottom of the list is what the user sees — the top is clipped
// off and the user cannot reach the earlier items by scrolling.
func TestViewItems_RendersOnlyVisibleWindow(t *testing.T) {
	items := makeScrollItems(30)
	m := tuiModel{
		items:          items,
		selected:       20,
		offset:         15,
		viewportHeight: 20, // visibleCount = 20 - 9 = 11
	}
	view := stripANSI(m.viewItems())

	// Items at indices 15-25 (task-16..task-26) should be visible.
	for i := 15; i <= 25; i++ {
		needle := fmt.Sprintf("task-%02d", i+1)
		if !strings.Contains(view, needle) {
			t.Errorf("expected view to contain %q, got:\n%s", needle, view)
		}
	}

	// Items at indices 0-14 (task-01..task-15) should be scrolled
	// off the top.
	for i := range 15 {
		needle := fmt.Sprintf("task-%02d", i+1)
		if strings.Contains(view, needle) {
			t.Errorf("view should not contain %q (scrolled off), got:\n%s", needle, view)
		}
	}

	// Items at indices 26-29 (task-27..task-30) should be below
	// the viewport.
	for i := 26; i < 30; i++ {
		needle := fmt.Sprintf("task-%02d", i+1)
		if strings.Contains(view, needle) {
			t.Errorf("view should not contain %q (below viewport), got:\n%s", needle, view)
		}
	}

	// The cursor must appear at the selected item (index 20,
	// task-21). After stripping ANSI, the cursor is ">" at the
	// start of the line, followed by " [ ] task-21".
	if !strings.Contains(view, "> [ ] task-21") {
		t.Errorf("expected cursor at task-21, got:\n%s", view)
	}
}

// TestViewItems_EmptyFilteredReturnsEmptyState verifies that the
// empty-state placeholder is still returned when the filtered list
// is empty, regardless of offset/viewportHeight.
func TestViewItems_EmptyFilteredReturnsEmptyState(t *testing.T) {
	m := tuiModel{
		items:          nil,
		selected:       0,
		offset:         0,
		viewportHeight: 20,
		// filterQuery defaults to "" so filteredItems returns m.items
		// (nil) unchanged.
	}
	view := m.viewItems()
	if !strings.Contains(view, "(No todos)") {
		t.Errorf("expected empty-state placeholder, got:\n%s", view)
	}
}

// TestViewItems_ClampsOutOfRangeOffset verifies that viewItems
// gracefully handles an offset that is out of range (either
// negative or past the end of the filtered list). The output must
// still be valid and the cursor must be visible.
func TestViewItems_ClampsOutOfRangeOffset(t *testing.T) {
	t.Run("negative offset clamps to 0", func(t *testing.T) {
		items := makeScrollItems(10)
		m := tuiModel{
			items:          items,
			selected:       0,
			offset:         -5, // invalid
			viewportHeight: 15,
		}
		view := stripANSI(m.viewItems())
		if !strings.Contains(view, "task-01") {
			t.Errorf("expected first item visible after clamping, got:\n%s", view)
		}
		if strings.Contains(view, "task-08") {
			t.Errorf("view should not contain task-08, got:\n%s", view)
		}
	})

	t.Run("offset past end clamps to last page", func(t *testing.T) {
		items := makeScrollItems(10)
		m := tuiModel{
			items:          items,
			selected:       9,   // last item
			offset:         100, // way past the end
			viewportHeight: 15,
		}
		view := stripANSI(m.viewItems())
		if !strings.Contains(view, "task-10") {
			t.Errorf("expected last item visible after clamping, got:\n%s", view)
		}
		if !strings.Contains(view, "> [ ] task-10") {
			t.Errorf("expected cursor at last item, got:\n%s", view)
		}
	})
}

// TestEnsureSelectedVisible pins the scroll-into-view behaviour.
// After any change to m.selected or m.viewportHeight, the offset
// must be adjusted so the selected item is in the visible window.
func TestEnsureSelectedVisible(t *testing.T) {
	items := makeScrollItems(30)

	tests := []struct {
		name       string
		selected   int
		startOff   int
		viewportH  int
		wantOffset int
	}{
		{
			name:       "selected at top of window — no scroll",
			selected:   5,
			startOff:   0,
			viewportH:  20, // visibleCount = 11
			wantOffset: 0,
		},
		{
			name:       "selected in middle of window — no scroll",
			selected:   5,
			startOff:   0,
			viewportH:  20,
			wantOffset: 0,
		},
		{
			name:       "selected at bottom edge of window — no scroll",
			selected:   10,
			startOff:   0,
			viewportH:  20, // window is [0,11), 10 is still inside
			wantOffset: 0,
		},
		{
			name:       "selected just below window — scroll down by 1",
			selected:   11,
			startOff:   0,
			viewportH:  20,
			wantOffset: 1, // 11 - 11 + 1
		},
		{
			name:       "selected well below window — scroll down fully",
			selected:   25,
			startOff:   0,
			viewportH:  20,
			wantOffset: 15, // 25 - 11 + 1
		},
		{
			name:       "selected above window — scroll up",
			selected:   3,
			startOff:   10,
			viewportH:  20,
			wantOffset: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tuiModel{
				items:          items,
				selected:       tt.selected,
				offset:         tt.startOff,
				viewportHeight: tt.viewportH,
			}
			m.ensureSelectedVisible()
			if m.offset != tt.wantOffset {
				t.Errorf("offset = %d, want %d", m.offset, tt.wantOffset)
			}
		})
	}
}

// TestUpdateNormalMode_AdjustsOffsetAfterNavigation verifies that
// pressing j (down) or k (up) moves the cursor AND scrolls the
// viewport so the cursor stays visible. This is the user-facing
// behaviour: as the user navigates past the visible window, the
// list scrolls.
func TestUpdateNormalMode_AdjustsOffsetAfterNavigation(t *testing.T) {
	t.Run("pressing j past the bottom edge scrolls the viewport", func(t *testing.T) {
		items := makeScrollItems(30)
		m := tuiModel{
			items:          items,
			selected:       0,
			offset:         0,
			viewportHeight: 20, // visibleCount = 11
		}
		// Press j 15 times. After 11 presses, the offset should
		// start advancing. After 15 presses, selected=15 and
		// offset = 15 - 11 + 1 = 5.
		for range 15 {
			updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
			m = updated.(tuiModel)
		}
		if m.selected != 15 {
			t.Errorf("selected = %d, want 15", m.selected)
		}
		if m.offset != 5 {
			t.Errorf("offset = %d, want 5", m.offset)
		}
	})

	t.Run("pressing k from below the top edge scrolls up", func(t *testing.T) {
		items := makeScrollItems(30)
		m := tuiModel{
			items:          items,
			selected:       15,
			offset:         5,
			viewportHeight: 20,
		}
		// Press k 10 times. selected=5, and since 5 < offset=5
		// the offset should move to 5.
		for range 10 {
			updated, _ := m.updateNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
			m = updated.(tuiModel)
		}
		if m.selected != 5 {
			t.Errorf("selected = %d, want 5", m.selected)
		}
		if m.offset != 5 {
			t.Errorf("offset = %d, want 5", m.offset)
		}
	})
}

// TestUpdate_HandlesWindowSizeMessage verifies that Bubble Tea's
// initial tea.WindowSizeMsg sets viewportHeight and scrolls the
// viewport so the (already-selected) item is in view.
func TestUpdate_HandlesWindowSizeMessage(t *testing.T) {
	items := makeScrollItems(30)
	m := tuiModel{
		items:    items,
		selected: 20,
		offset:   0,
	}
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	if cmd != nil {
		t.Errorf("expected nil cmd, got %v", cmd)
	}
	got := updated.(tuiModel)
	if got.viewportHeight != 20 {
		t.Errorf("viewportHeight = %d, want 20", got.viewportHeight)
	}
	// visibleCount = 20-9 = 11, so offset = 20-11+1 = 10.
	if got.offset != 10 {
		t.Errorf("offset = %d, want 10", got.offset)
	}
}
