package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inful/todoer/pkg/core"
)

// TestView_StateMarkers captures the state-line markers in the View
// output. These are stable, easy to assert on, and survive the
// viewHeader / viewItems / viewStatus decomposition in Phase 2b.
// We deliberately use markers rather than full-string equality so the
// tests survive whitespace and colour-rendering changes.
func TestView_StateMarkers(t *testing.T) {
	const today = "2026-03-14"

	newModel := func(t *testing.T) tuiModel {
		t.Helper()
		path := writeJournal(t, sampleJournal)
		m, err := newTUIModel(path, today, &Config{TodosHeader: core.TodosHeader})
		if err != nil {
			t.Fatalf("newTUIModel: %v", err)
		}
		return m
	}

	t.Run("clean and no external change shows State: clean", func(t *testing.T) {
		m := newModel(t)
		view := m.View()
		if !strings.Contains(view, "State:") {
			t.Fatalf("expected State: line in view, got:\n%s", view)
		}
		// The state is rendered as a styled line; strip ANSI escapes
		// before checking the marker.
		if !strings.Contains(stripANSI(view), "clean") {
			t.Errorf("expected 'clean' marker in view, got:\n%s", stripANSI(view))
		}
		if strings.Contains(stripANSI(view), "dirty") {
			t.Errorf("did not expect 'dirty' marker in clean state, got:\n%s", stripANSI(view))
		}
	})

	t.Run("dirty model shows State: dirty", func(t *testing.T) {
		m := newModel(t)
		m.dirty = true
		view := m.View()
		if !strings.Contains(stripANSI(view), "dirty") {
			t.Errorf("expected 'dirty' marker, got:\n%s", stripANSI(view))
		}
	})

	t.Run("externalChanged appends to state", func(t *testing.T) {
		m := newModel(t)
		m.externalChanged = true
		view := m.View()
		if !strings.Contains(stripANSI(view), "external-changed") {
			t.Errorf("expected 'external-changed' marker, got:\n%s", stripANSI(view))
		}
	})

	t.Run("filter line is present in header", func(t *testing.T) {
		m := newModel(t)
		view := m.View()
		if !strings.Contains(view, "Filter:") {
			t.Errorf("expected 'Filter:' line in view, got:\n%s", view)
		}
	})

	t.Run("input mode shows prompt", func(t *testing.T) {
		m := newModel(t)
		m.inputMode = true
		m.inputText = "new todo"
		view := m.View()
		if !strings.Contains(stripANSI(view), "Add todo:") {
			t.Errorf("expected 'Add todo:' prompt, got:\n%s", stripANSI(view))
		}
		if !strings.Contains(stripANSI(view), "new todo") {
			t.Errorf("expected typed text in prompt, got:\n%s", stripANSI(view))
		}
	})

	t.Run("filter mode shows filter prompt", func(t *testing.T) {
		m := newModel(t)
		m.filterMode = true
		m.filterQuery = "abc"
		view := m.View()
		if !strings.Contains(stripANSI(view), "Filter todos:") {
			t.Errorf("expected 'Filter todos:' prompt, got:\n%s", stripANSI(view))
		}
		if !strings.Contains(stripANSI(view), "abc") {
			t.Errorf("expected typed filter in prompt, got:\n%s", stripANSI(view))
		}
	})

	t.Run("normal mode shows help text", func(t *testing.T) {
		m := newModel(t)
		// Verify the help line drives from tuiKeymap (drift-proof).
		view := m.View()
		for _, k := range tuiKeymap {
			if !strings.Contains(stripANSI(view), k.label) {
				t.Errorf("expected help line to contain key %q label, got:\n%s", k.label, stripANSI(view))
			}
		}
	})

	t.Run("status with failed or error uses error style and is still rendered", func(t *testing.T) {
		// We can't easily inspect the chosen style; assert the text
		// is in the view and the Status: marker is present.
		m := newModel(t)
		m.status = "save failed: disk full"
		view := m.View()
		if !strings.Contains(view, "Status:") {
			t.Errorf("expected 'Status:' line, got:\n%s", view)
		}
		if !strings.Contains(stripANSI(view), "save failed: disk full") {
			t.Errorf("expected status text, got:\n%s", stripANSI(view))
		}
	})

	t.Run("status with external or blocked uses warn style", func(t *testing.T) {
		m := newModel(t)
		m.status = "External change detected. Save blocked."
		view := m.View()
		if !strings.Contains(stripANSI(view), "Save blocked") {
			t.Errorf("expected status text, got:\n%s", stripANSI(view))
		}
	})

	t.Run("empty items shows placeholder", func(t *testing.T) {
		m := newModel(t)
		// Force empty by clearing journal.
		m.journal = &core.TodoJournal{Days: []*core.DaySection{}}
		m.todayDay = nil
		m.displayDay = nil
		m.refreshItems()
		view := m.View()
		stripped := stripANSI(view)
		if !strings.Contains(stripped, "No todos") {
			t.Errorf("expected 'No todos' placeholder, got:\n%s", stripped)
		}
	})
}

// stripANSI removes ANSI escape sequences so the test assertions can
// match on text content rather than rendered styles. lipgloss adds
// colour codes that change with theme tweaks; the underlying text is
// what we care about.
func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEscape = true
		case inEscape:
			if r == 'm' {
				inEscape = false
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// _ keeps the bubbletea import used elsewhere in this package.
var _ tea.KeyMsg
