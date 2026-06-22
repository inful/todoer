package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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

func TestTUIUpdateInputMode_BackspaceRuneSafe(t *testing.T) {
	m := tuiModel{inputMode: true}

	// Type a multi-byte rune.
	updated, _ := m.updateInputMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'é'}})
	m2 := updated.(tuiModel)
	if m2.inputText != "é" {
		t.Fatalf("expected inputText 'é', got %q", m2.inputText)
	}

	// Backspace should drop the whole rune, not leave a broken
	// UTF-8 sequence in the string.
	updated, _ = m2.updateInputMode(tea.KeyMsg{Type: tea.KeyBackspace})
	m3 := updated.(tuiModel)
	if m3.inputText != "" {
		t.Fatalf("expected backspace to drop the whole rune, got %q (% x)", m3.inputText, []byte(m3.inputText))
	}

	// Multi-byte + ASCII mixed.
	updated, _ = m3.updateInputMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	updated, _ = updated.(tuiModel).updateInputMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'é'}})
	updated, _ = updated.(tuiModel).updateInputMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m4 := updated.(tuiModel)
	if m4.inputText != "aéb" {
		t.Fatalf("expected 'aéb', got %q", m4.inputText)
	}

	// Drop the trailing ASCII 'b'.
	updated, _ = m4.updateInputMode(tea.KeyMsg{Type: tea.KeyBackspace})
	m5 := updated.(tuiModel)
	if m5.inputText != "aé" {
		t.Fatalf("expected 'aé' after dropping 'b', got %q", m5.inputText)
	}

	// Drop the multi-byte 'é' cleanly.
	updated, _ = m5.updateInputMode(tea.KeyMsg{Type: tea.KeyBackspace})
	m6 := updated.(tuiModel)
	if m6.inputText != "a" {
		t.Fatalf("expected 'a' after dropping 'é', got %q (% x)", m6.inputText, []byte(m6.inputText))
	}
}
