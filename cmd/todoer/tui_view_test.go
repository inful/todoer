package main

import (
	"strings"
	"testing"

	"github.com/inful/todoer/pkg/core"
)

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
	// The view always shows items from all days, so the empty-state
	// message is a single generic placeholder. This used to have
	// three cases (no display day, today view, carryover view) but
	// the distinction no longer matters: the view always shows all
	// days, and if no days have items, the generic message covers
	// all cases.
	tests := []struct {
		name     string
		today    string
		todayDay *core.DaySection
		display  *core.DaySection
	}{
		{"no display day", "2026-03-16", nil, nil},
		{"today view, empty", "2026-03-16", &core.DaySection{Date: "2026-03-16"}, &core.DaySection{Date: "2026-03-16"}},
		{"carryover view, empty", "2026-03-16", &core.DaySection{Date: "2026-03-16"}, &core.DaySection{Date: "2026-03-15"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tuiModel{today: tt.today, todayDay: tt.todayDay, displayDay: tt.display}
			if got := m.emptyStateMessage(); got != "(No todos)" {
				t.Errorf("expected '(No todos)', got %q", got)
			}
		})
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
	// The empty-state message is now generic ("(No todos)") because
	// the view always shows items from all days, so the section-
	// specific message no longer applies.
	if !strings.Contains(view, "(No todos)") {
		t.Fatalf("expected generic empty-state message, got:\n%s", view)
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

func TestTUIHelpText_DriftGuard(t *testing.T) {
	help := tuiHelpText()
	init := tuiInitStatus()

	// Invariant 1: init status embeds the help text body.
	if !strings.Contains(init, strings.TrimPrefix(help, "Keys: ")) {
		t.Errorf("init status does not contain help text: init=%q help=%q", init, help)
	}

	// Invariant 2: the primary keys from the keymap appear in the
	// help text. This is the drift guard for the user-facing text.
	required := []string{"j", "k", "space", "/", "c", "a", "d", "s", "r", "q"}
	for _, key := range required {
		if !strings.Contains(help, key) {
			t.Errorf("help text is missing advertised key %q: %q", key, help)
		}
	}

	// Invariant 3: the rendered help text's " | "-separated segment
	// count matches the keymap length. This catches both stale
	// entries (a keymap row removed but the test still passes) and
	// accidental keymap duplication (more rows than rendered).
	segments := strings.Split(strings.TrimPrefix(help, "Keys: "), " | ")
	if got, want := len(segments), len(tuiKeymap); got != want {
		t.Errorf("help text has %d segments; expected %d (matches tuiKeymap): %q", got, want, help)
	}
}

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
