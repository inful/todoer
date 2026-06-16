package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/inful/todoer/pkg/core"
)

// cmdAdd ensures today's journal exists and appends a new unchecked todo item.
// If today's journal has not yet been created, it first runs the same transfer flow as `new`.
// When backup is true, a .bak of the source journal is preserved before the carryover update.
func cmdAdd(rootDir, templateFile, todoText string, printPath, backup bool, config *Config, logger *Logger) error {
	trimmedTodo := strings.TrimSpace(todoText)
	if trimmedTodo == "" {
		return fmt.Errorf("todo text cannot be empty")
	}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(rootDir, today)

	if _, err := os.Stat(journalPath); os.IsNotExist(err) {
		logger.Info("Today's journal does not exist yet, creating it first.")
		if err := cmdNewWithOptions(rootDir, templateFile, false, backup, config, logger); err != nil {
			return fmt.Errorf("failed to create today's journal before adding todo: %w", err)
		}
	}

	if err := appendTodoToJournal(journalPath, today, trimmedTodo, config); err != nil {
		return err
	}

	logger.Info("Added todo to today's journal: %s", journalPath)
	if printPath {
		fmt.Println(journalPath)
	}

	return nil
}

func appendTodoToJournal(journalPath, today, todoText string, config *Config) error {
	content, err := os.ReadFile(journalPath)
	if err != nil {
		return fmt.Errorf("failed to read journal file %s: %w", journalPath, err)
	}

	beforeTodos, todosSection, afterTodos, err := core.ExtractTodosSectionWithHeader(string(content), config.TodosHeader)
	if err != nil {
		return fmt.Errorf("failed to locate todos section in %s: %w", journalPath, err)
	}

	journal, err := core.ParseTodosSection(todosSection)
	if err != nil {
		return fmt.Errorf("failed to parse todos section in %s: %w", journalPath, err)
	}

	journal = core.MoveUndatedTodosToCurrentDate(journal, today)
	if journal == nil {
		journal = &core.TodoJournal{Days: []*core.DaySection{}}
	}

	todaySection := findOrCreateDaySection(journal, today)
	todaySection.Items = append(todaySection.Items, &core.TodoItem{
		Completed:   false,
		Text:        todoText,
		SubItems:    []*core.TodoItem{},
		BulletLines: []string{},
	})

	newTodos := core.JournalToString(journal)
	newContent := beforeTodos + newTodos + afterTodos

	if err := safeWriteFile(journalPath, []byte(newContent), FilePermissions); err != nil {
		return fmt.Errorf("failed to write updated journal file %s: %w", journalPath, err)
	}

	return nil
}

func findOrCreateDaySection(journal *core.TodoJournal, date string) *core.DaySection {
	for _, day := range journal.Days {
		if day != nil && day.Date == date {
			return day
		}
	}

	newDay := &core.DaySection{
		Date:  date,
		Items: []*core.TodoItem{},
	}
	journal.Days = append(journal.Days, newDay)
	return newDay
}
