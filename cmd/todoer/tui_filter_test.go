package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inful/todoer/pkg/core"
)

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

func TestTUIUpdateFilterMode_BackspaceRuneSafe(t *testing.T) {
	m := tuiModel{filterMode: true}

	updated, _ := m.updateFilterMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'é'}})
	m2 := updated.(tuiModel)
	if m2.filterQuery != "é" {
		t.Fatalf("expected filterQuery 'é', got %q", m2.filterQuery)
	}

	updated, _ = m2.updateFilterMode(tea.KeyMsg{Type: tea.KeyBackspace})
	m3 := updated.(tuiModel)
	if m3.filterQuery != "" {
		t.Fatalf("expected backspace to drop the whole rune, got %q (% x)", m3.filterQuery, []byte(m3.filterQuery))
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
	// Esc cancels the in-progress query. To keep a filter the user
	// presses Enter.
	if m4.filterQuery != "" {
		t.Fatalf("expected filterQuery to be cleared on esc, got %q", m4.filterQuery)
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
