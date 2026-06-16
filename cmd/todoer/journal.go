package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/inful/todoer/pkg/core"
	"github.com/inful/todoer/pkg/generator"
)

// getGenerator builds a Generator from CLI/config, resolving template and previous date.
func getGenerator(templateFile, templateDate, sourceFile string, config *Config) (*generator.Generator, string, error) {
	if templateDate == "" {
		templateDate = time.Now().Format(core.DateFormat)
	}

	previousDate := ""
	if sourceFile != "" {
		if content, readErr := os.ReadFile(sourceFile); readErr == nil {
			if extractedDate, extractErr := core.ExtractDateFromFrontmatter(string(content), config.FrontmatterDateKey); extractErr == nil {
				previousDate = extractedDate
			}
		}
	}

	tmplContent, tmplName, err := resolveTemplate(templateFile)
	if err != nil {
		return nil, "", fmt.Errorf("error resolving template: %w", err)
	}

	gen, err := generator.NewGeneratorWithOptions(tmplContent, templateDate,
		generator.WithPreviousDate(previousDate),
		generator.WithCustomVariables(config.Custom),
		generator.WithFrontmatterDateKey(config.FrontmatterDateKey),
		generator.WithTodosHeader(config.TodosHeader),
	)
	if err != nil {
		return nil, "", fmt.Errorf("error creating generator from template: %w", err)
	}

	return gen, tmplName, nil
}

// processJournal processes a journal file, writing the target and optionally
// updating the source with a backup. When merge is true and the target
// file already exists, the source's uncompleted items are merged into the
// target's existing todos via core.MergeCarryover instead of overwriting
// the target. When merge is false, the target is overwritten in full.
func processJournal(sourceFile, targetFile, templateFile, templateDate string, skipBackup, printPath, merge bool, config *Config, logger *Logger) error {
	logger.Debug("Processing journal: source=%s, target=%s, template=%s, date=%s", sourceFile, targetFile, templateFile, templateDate)

	if err := validateProcessArgs(sourceFile, targetFile, templateDate); err != nil {
		return err
	}

	gen, templateSource, err := getGenerator(templateFile, templateDate, sourceFile, config)
	if err != nil {
		return err
	}

	logger.Debug("Using template source: %s", templateSource)

	result, err := gen.ProcessFile(sourceFile)
	if err != nil {
		return fmt.Errorf("error processing file %s: %w", sourceFile, err)
	}

	modifiedContentBytes, err := io.ReadAll(result.ModifiedOriginal)
	if err != nil {
		return fmt.Errorf("error reading modified content: %w", err)
	}

	newContentBytes, err := io.ReadAll(result.NewFile)
	if err != nil {
		return fmt.Errorf("error reading new file content: %w", err)
	}

	// If the target exists and the caller asked for a merge, splice
	// the source's uncompleted items into the target's existing todos
	// instead of overwriting. The merge preserves the target's body
	// (frontmatter, sections after Todos) and uses the source's
	// items for the carryover.
	if merge {
		if existing, err := os.ReadFile(targetFile); err == nil {
			merged, mergeErr := mergeIntoExistingTarget(existing, newContentBytes, config)
			if mergeErr != nil {
				return fmt.Errorf("error merging into existing target %s: %w", targetFile, mergeErr)
			}
			newContentBytes = merged
			logger.Info("Merged carryover into existing target: %s", targetFile)
		}
	}

	logger.Debug("Writing %d bytes to target file: %s", len(newContentBytes), targetFile)
	if err := safeWriteFile(targetFile, newContentBytes, FilePermissions); err != nil {
		return fmt.Errorf("error writing to target file %s: %w", targetFile, err)
	}

	logger.Info("Successfully processed %s -> %s (template: %s)", sourceFile, targetFile, templateSource)

	if printPath {
		fmt.Println(targetFile)
	}

	if len(modifiedContentBytes) > 0 {
		if !skipBackup {
			backupFile := sourceFile + ".bak"
			originalContentBytes, err := os.ReadFile(sourceFile)
			if err != nil {
				return fmt.Errorf("error reading original file for backup: %w", err)
			}
			if err := safeWriteFile(backupFile, originalContentBytes, FilePermissions); err != nil {
				return fmt.Errorf("error creating backup file %s: %w", backupFile, err)
			}

			logger.Info("Backup of original file created: %s", backupFile)
		}

		if err := safeWriteFile(sourceFile, modifiedContentBytes, FilePermissions); err != nil {
			return fmt.Errorf("error updating source file %s: %w", sourceFile, err)
		}

		if skipBackup {
			logger.Info("Source file updated without backup: %s", sourceFile)
		}
	} else {
		logger.Info("No modifications found in the original file.")
	}

	return nil
}

// mergeIntoExistingTarget merges the uncompleted todos from newContent
// into the existing target file's todos section. The target's body
// (everything before and after the todos section) is preserved verbatim;
// only the todos are replaced with the merge of the two journals.
func mergeIntoExistingTarget(existingTarget, newContent []byte, config *Config) ([]byte, error) {
	beforeTodos, existingTodosSection, afterTodos, err := core.ExtractTodosSectionWithHeader(string(existingTarget), config.TodosHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to extract existing todos: %w", err)
	}
	existingJournal, err := core.ParseTodosSection(existingTodosSection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse existing todos: %w", err)
	}

	_, newTodosSection, _, err := core.ExtractTodosSectionWithHeader(string(newContent), config.TodosHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to extract new todos: %w", err)
	}
	newJournal, err := core.ParseTodosSection(newTodosSection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse new todos: %w", err)
	}

	merged := core.MergeCarryover(newJournal, existingJournal)
	newTodos := core.JournalToString(merged)
	return []byte(beforeTodos + newTodos + afterTodos), nil
}

// findClosestJournalFile returns the most recent journal before the given date.
func findClosestJournalFile(rootDir, today string) (string, error) {
	var closestFile string
	var minDiff time.Duration = -1

	todayTime, err := time.Parse(core.DateFormat, today)
	if err != nil {
		return "", fmt.Errorf("invalid today date: %w", err)
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		if len(base) != len("2006-01-02.md") || filepath.Ext(base) != ".md" {
			return nil
		}

		dateStr := strings.TrimSuffix(base, ".md")
		fileTime, err := time.Parse(core.DateFormat, dateStr)
		if err != nil {
			return nil
		}

		if fileTime.Before(todayTime) {
			diff := todayTime.Sub(fileTime)
			if minDiff == -1 || diff < minDiff {
				minDiff = diff
				closestFile = path
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if closestFile == "" {
		return "", fmt.Errorf("no previous journal found in %s", rootDir)
	}

	return closestFile, nil
}

// cmdNew creates today's journal using the closest previous journal or a blank template.
// It does not preserve a .bak of the source journal by default; callers that want a
// backup should call cmdNewWithOptions with preserveSourceBackup=true or invoke
// the `new --backup` flag.
func cmdNew(rootDir, templateFile string, printPath bool, config *Config, logger *Logger) error {
	return cmdNewWithOptions(rootDir, templateFile, printPath, false, config, logger)
}

// cmdNewWithOptions creates today's journal and optionally preserves a backup of the source journal.
func cmdNewWithOptions(rootDir, templateFile string, printPath, preserveSourceBackup bool, config *Config, logger *Logger) error {
	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(rootDir, today)

	if _, err := os.Stat(journalPath); err == nil {
		if printPath {
			fmt.Println(journalPath)
		} else {
			logger.Info("Journal for today already exists: %s", journalPath)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(journalPath), 0o755); err != nil {
		return err
	}

	closest, err := findClosestJournalFile(rootDir, today)
	skipBackup := !preserveSourceBackup
	if err != nil {
		logger.Info("No previous journal found, creating a new one from template.")

		tmpFile, err := os.CreateTemp(filepath.Dir(journalPath), "empty-journal-*.md")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "warning: failed to remove temp file %s: %v\n", tmpFile.Name(), err)
			}
		}()

		if _, err := tmpFile.WriteString(core.TodosHeader + "\n\n"); err != nil {
			return fmt.Errorf("failed to write to temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf("failed to close temp file: %w", err)
		}

		closest = tmpFile.Name()
		skipBackup = true
	}

	logger.Info("Using '%s' as source to create new journal for today.", closest)

	// merge=true: the daily flow must be idempotent and merge-into-existing
	// per ADR-0001. If today's journal already exists (e.g. created by
	// hand or by an earlier run), the source's uncompleted items are
	// merged in instead of overwriting the target.
	if err := processJournal(closest, journalPath, templateFile, today, skipBackup, printPath, true, config, logger); err != nil {
		return err
	}

	return nil
}

// buildJournalPath constructs a YYYY/MM/YYYY-MM-DD.md path under rootDir.
func buildJournalPath(rootDir, date string) string {
	t, err := time.Parse(core.DateFormat, date)
	if err != nil {
		t = time.Now()
	}
	year := t.Format("2006")
	month := t.Format("01")
	return filepath.Join(rootDir, year, month, date+".md")
}
