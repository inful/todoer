package main

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/inful/todoer/pkg/core"
)

// reloadFromDisk re-reads the journal into the model.
//
// A parse failure is not fatal: the journal has a line the model cannot
// represent, so the TUI opens read-only with the reason in the status line
// rather than refusing to start. Writing would rewrite the todos section
// from the model and drop that line, so every write is refused until the
// user reloads a file that parses.
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
		m.readOnly = true
		m.status = fmt.Sprintf("Read-only: %v", err)
		// Remember the hash anyway: otherwise the periodic check sees a
		// "change" every tick and re-parses the file forever.
		m.fileHash = hashBytes(content)
		return nil
	}

	m.readOnly = false
	m.beforeTodos = beforeTodos
	m.afterTodos = afterTodos
	m.journal = journal
	// Sort the day sections so the carryover fallback (which walks
	// slices.Backward) is independent of the on-disk file order.
	core.SortJournalDays(journal)
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
	if m.readOnly {
		return fmt.Errorf("journal has a line todoer cannot parse; not writing. Fix the file and press r")
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
	newContent := []byte(core.MatchLineEndings(m.beforeTodos+newTodos+m.afterTodos, string(currentContent)))

	// Keep the pre-save bytes next to the file: the TUI rewrites the
	// whole journal on every save, so this is the only undo.
	if err := safeWriteFile(m.journalPath+".bak", currentContent, FilePermissions); err != nil {
		return fmt.Errorf("failed to back up journal file %s: %w", m.journalPath, err)
	}

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
		if m.readOnly {
			// reloadFromDisk set the reason; do not overwrite it.
			return
		}
		m.status = "External change detected and reloaded"
	}
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}
