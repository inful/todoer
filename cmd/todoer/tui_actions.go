package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inful/todoer/pkg/core"
)

func (m tuiModel) updateInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputMode = false
		m.inputText = ""
		m.status = "Add cancelled"
	case "enter":
		text := strings.TrimSpace(m.inputText)
		if text == "" {
			m.status = "Cannot add empty todo"
			return m, nil
		}
		wasCarryoverView := m.isReadOnlyView()
		if m.todayDay == nil {
			m.todayDay = core.FindOrCreateDaySection(m.journal, m.today)
		}
		m.todayDay.Items = append(m.todayDay.Items, &core.TodoItem{
			Completed:   false,
			Text:        text,
			SubItems:    []*core.TodoItem{},
			BulletLines: []string{},
		})
		// Keep the display day sticky. If the user was in a carryover view
		// they should still see the carryover items; the new todo is in
		// today's section and will appear after the next save+reload.
		// If the user was already on today, the new todo is appended to
		// the displayed day and is visible immediately.
		if !wasCarryoverView {
			m.displayDay = m.todayDay
		}
		m.refreshItems()
		if len(m.items) > 0 {
			m.selected = len(m.items) - 1
			m.ensureSelectedVisible()
		}
		m.dirty = true
		m.inputMode = false
		m.inputText = ""
		if wasCarryoverView {
			m.status = "Added to today (carryover still shown); press r after save to see today"
		} else {
			m.status = "Todo added"
		}
	case "backspace":
		if len(m.inputText) > 0 {
			// Remove the last rune (not the last byte) so multi-byte
			// characters like accented letters, CJK, and emoji are
			// dropped cleanly. utf8.DecodeLastRuneInString returns
			// size=0 only for the empty string; we have already
			// guarded on len > 0 so size is at least 1.
			_, size := utf8.DecodeLastRuneInString(m.inputText)
			m.inputText = m.inputText[:len(m.inputText)-size]
		}
	default:
		if len(msg.Runes) > 0 {
			m.inputText += string(msg.Runes)
		}
	}

	return m, nil
}

func (m tuiModel) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredItems()

	switch msg.String() {
	case "ctrl+c", "q":
		if msg.String() == "q" && m.dirty {
			if err := m.saveToDisk(); err != nil {
				m.status = fmt.Sprintf("Save failed: %v", err)
				return m, nil
			}
			m.status = "Saved"
		}
		return m, tea.Quit
	case "j", "down":
		if m.selected < len(filtered)-1 {
			m.selected++
			m.ensureSelectedVisible()
		}
	case "k", "up":
		if m.selected > 0 {
			m.selected--
			m.ensureSelectedVisible()
		}
	case "/":
		m.filterMode = true
		m.status = "Type filter text, Enter to apply, Esc to cancel"
	case " ", "x":
		if len(filtered) == 0 {
			m.status = "No todo selected"
			return m, nil
		}
		if m.isReadOnlyView() {
			m.status = "Cannot edit carryover items"
			return m, nil
		}
		entry := filtered[m.selected]
		entry.item.Completed = !entry.item.Completed
		m.dirty = true
		m.status = "Todo toggled"
	case "d":
		if len(filtered) == 0 {
			m.status = "No todo selected"
			return m, nil
		}
		if m.isReadOnlyView() {
			m.status = "Cannot edit carryover items"
			return m, nil
		}
		target := filtered[m.selected].item
		m.journal.Days = core.RemoveItemFromDays(m.journal.Days, target)
		m.todayDay = core.FindDaySection(m.journal, m.today)
		m.refreshItems()
		filtered = m.filteredItems()
		if m.selected >= len(filtered) && len(filtered) > 0 {
			m.selected = len(filtered) - 1
		}
		if len(filtered) == 0 {
			m.selected = 0
		}
		m.dirty = true
		m.status = "Todo deleted"
		m.ensureSelectedVisible()
	case "a":
		m.inputMode = true
		m.inputText = ""
		m.status = "Type todo text, Enter to add, Esc to cancel"
	case "s":
		if err := m.saveToDisk(); err != nil {
			m.status = fmt.Sprintf("Save failed: %v", err)
		} else {
			m.status = "Saved"
		}
	case "r":
		if err := m.reloadFromDisk(); err != nil {
			m.status = fmt.Sprintf("Reload failed: %v", err)
		} else {
			m.status = "Reloaded from disk"
		}
	case "c":
		m.filterQuery = ""
		m.selected = 0
		m.status = "Filter cleared"
	}

	return m, nil
}

func (m tuiModel) updateFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterMode = false
		// Cancel discards the in-progress query so the user is
		// not surprised by a filter they thought they had
		// abandoned. To keep a filter, the user presses Enter
		// instead.
		m.filterQuery = ""
		m.selected = 0
		m.status = "Filter cancelled"
	case "enter":
		m.filterMode = false
		m.selected = 0
		if m.filterQuery == "" {
			m.status = "Filter cleared"
		} else {
			m.status = fmt.Sprintf("Filter applied: %q", m.filterQuery)
		}
	case "backspace":
		if len(m.filterQuery) > 0 {
			// Rune-safe backspace; see updateInputMode for the
			// reasoning.
			_, size := utf8.DecodeLastRuneInString(m.filterQuery)
			m.filterQuery = m.filterQuery[:len(m.filterQuery)-size]
		}
		m.selected = 0
	default:
		if len(msg.Runes) > 0 {
			m.filterQuery += string(msg.Runes)
			m.selected = 0
		}
	}

	return m, nil
}
