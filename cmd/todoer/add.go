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
// The previous version of the journal is kept in ROOT/.backups before it is
// rewritten.
func cmdAdd(rootDir, templateFile, todoText string, printPath bool, config *Config, logger *Logger) error {
	trimmedTodo := strings.TrimSpace(todoText)
	if trimmedTodo == "" {
		return fmt.Errorf("todo text cannot be empty")
	}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(rootDir, today)

	if _, err := os.Stat(journalPath); os.IsNotExist(err) {
		logger.Info("Today's journal does not exist yet, creating it first.")
		if err := cmdNewWithOptions(rootDir, templateFile, false, config, logger); err != nil {
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
		if strings.TrimSpace(string(content)) != "" {
			return fmt.Errorf("failed to locate todos section in %s: %w (add a '%s' section to the file, or set todos_header in the config)",
				journalPath, err, config.TodosHeader)
		}
		// An empty file has no section yet: start one.
		beforeTodos, todosSection, afterTodos = config.TodosHeader+"\n\n", "", "\n"
	}

	journal, err := core.ParseTodosSection(todosSection)
	if err != nil {
		return fmt.Errorf("failed to parse todos section in %s: %w", journalPath, err)
	}

	journal = core.MoveUndatedTodosToCurrentDate(journal, today)
	if journal == nil {
		journal = &core.TodoJournal{Days: []*core.DaySection{}}
	}

	todaySection := core.FindOrCreateDaySection(journal, today)
	todaySection.Items = append(todaySection.Items, &core.TodoItem{
		Completed:   false,
		Text:        todoText,
		SubItems:    []*core.TodoItem{},
		BulletLines: []core.BulletLine{},
	})

	// Keep the day sections in date order, the way the TUI writes them.
	core.SortJournalDays(journal)

	newTodos := core.JournalToString(journal)
	newContent := core.MatchLineEndings(beforeTodos+newTodos+afterTodos, string(content))

	// The whole file is rewritten from the model, so keep the previous
	// version: a bug here would otherwise cost the user the day.
	if err := backupFile(config.RootDir, journalPath, content, config.backups()); err != nil {
		return fmt.Errorf("failed to back up journal file %s: %w", journalPath, err)
	}

	if err := safeWriteFile(journalPath, []byte(newContent), FilePermissions); err != nil {
		return fmt.Errorf("failed to write updated journal file %s: %w", journalPath, err)
	}

	return nil
}
