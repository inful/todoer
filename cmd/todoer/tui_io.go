package main

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/inful/todoer/pkg/core"
)

func (m *tuiModel) reloadFromDisk() error {
	content, err := os.ReadFile(m.journalPath)
	if err != nil {
		return fmt.Errorf("failed to read journal file %s: %w", m.journalPath, err)
	}

	beforeTodos, todosSection, afterTodos, err := core.ExtractTodosSectionWithHeader(string(content), m.config.TodosHeader)
	if err != nil {
		return fmt.Errorf("failed to locate todos section in %s: %w", m.journalPath, err)
	}

	journal, err := core.ParseTodosSection(todosSection)
	if err != nil {
		return fmt.Errorf("failed to parse todos section in %s: %w", m.journalPath, err)
	}

	m.beforeTodos = beforeTodos
	m.afterTodos = afterTodos
	m.journal = journal
	m.todayDay = core.FindDaySection(journal, m.today)
	m.displayDay = m.pickInitialDisplayDay()
	m.refreshItems()
	if m.selected >= len(m.items) {
		m.selected = max(0, len(m.items)-1)
	}
	m.dirty = false
	m.externalChanged = false
	m.fileHash = hashBytes(content)

	return nil
}

func (m *tuiModel) saveToDisk() error {
	if !m.dirty {
		return nil
	}

	currentContent, err := os.ReadFile(m.journalPath)
	if err != nil {
		return fmt.Errorf("failed to read journal %s for conflict check: %w", m.journalPath, err)
	}

	currentHash := hashBytes(currentContent)
	if currentHash != m.fileHash {
		m.externalChanged = true
		return fmt.Errorf("file changed externally, reload before saving")
	}

	newTodos := core.JournalToString(m.journal)
	newContent := []byte(m.beforeTodos + newTodos + m.afterTodos)

	if err := safeWriteFile(m.journalPath, newContent, FilePermissions); err != nil {
		return fmt.Errorf("failed to write updated journal file %s: %w", m.journalPath, err)
	}

	m.fileHash = hashBytes(newContent)
	m.dirty = false
	m.externalChanged = false

	return nil
}

func (m *tuiModel) checkExternalChanges() {
	content, err := os.ReadFile(m.journalPath)
	if err != nil {
		return
	}

	newHash := hashBytes(content)
	if newHash == m.fileHash {
		return
	}

	if m.dirty {
		m.externalChanged = true
		m.status = "External change detected. Save blocked; press r to reload."
		return
	}

	if err := m.reloadFromDisk(); err == nil {
		m.status = "External change detected and reloaded"
	}
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}
