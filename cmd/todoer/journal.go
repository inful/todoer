package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"todoer/pkg/core"
	"todoer/pkg/generator"
)

// getGenerator builds a Generator from CLI/config, resolving template and previous date.
func getGenerator(templateFile, templateDate, sourceFile string, config *Config) (*generator.Generator, string, error) {
	if templateDate == "" {
		templateDate = time.Now().Format(core.DateFormat)
	}

	previousDate := ""
	if sourceFile != "" {
		if content, readErr := os.ReadFile(sourceFile); readErr == nil {
			if extractedDate, extractErr := generator.ExtractDateFromFrontmatter(string(content), config.FrontmatterDateKey); extractErr == nil {
				previousDate = extractedDate
			}
		}
	}

	tmplSource := resolveTemplate(templateFile)
	if tmplSource.err != nil {
		return nil, "", fmt.Errorf("error resolving template: %w", tmplSource.err)
	}

	gen, err := generator.NewGeneratorWithOptions(tmplSource.content, templateDate,
		generator.WithPreviousDate(previousDate),
		generator.WithCustomVariables(config.Custom),
		generator.WithFrontmatterDateKey(config.FrontmatterDateKey),
		generator.WithTodosHeader(config.TodosHeader),
	)
	if err != nil {
		return nil, "", fmt.Errorf("error creating generator from template: %w", err)
	}

	return gen, tmplSource.name, nil
}

// processJournal processes a journal file, writing the target and optionally updating source with backup.
func processJournal(sourceFile, targetFile, templateFile, templateDate string, skipBackup, printPath bool, config *Config, logger *Logger) error {
	logger.Debug("Processing journal: source=%s, target=%s, template=%s, date=%s", sourceFile, targetFile, templateFile, templateDate)

	quiet := printPath

	if err := validateProcessArgs(sourceFile, targetFile, templateDate); err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	gen, templateSource, err := getGenerator(templateFile, templateDate, sourceFile, config)
	if err != nil {
		return err
	}

	logger.Debug("Using template source: %s", templateSource)

	result, err := gen.ProcessFile(sourceFile)
	if err != nil {
		return fmt.Errorf("error processing file %s: %v", sourceFile, err)
	}

	modifiedContentBytes, err := io.ReadAll(result.ModifiedOriginal)
	if err != nil {
		return fmt.Errorf("error reading modified content: %v", err)
	}

	newContentBytes, err := io.ReadAll(result.NewFile)
	if err != nil {
		return fmt.Errorf("error reading new file content: %v", err)
	}

	logger.Debug("Writing %d bytes to target file: %s", len(newContentBytes), targetFile)
	if err := safeWriteFile(targetFile, newContentBytes, FilePermissions); err != nil {
		return fmt.Errorf("error writing to target file %s: %v", targetFile, err)
	}

	logger.Info("Successfully processed %s -> %s (template: %s)", sourceFile, targetFile, templateSource)

	if printPath {
		fmt.Println(targetFile)
	}

	if len(modifiedContentBytes) > 0 && !skipBackup {
		backupFile := sourceFile + ".bak"
		originalContentBytes, err := os.ReadFile(sourceFile)
		if err != nil {
			return fmt.Errorf("error reading original file for backup: %v", err)
		}
		if err := safeWriteFile(backupFile, originalContentBytes, FilePermissions); err != nil {
			return fmt.Errorf("error creating backup file %s: %v", backupFile, err)
		}

		if err := safeWriteFile(sourceFile, modifiedContentBytes, FilePermissions); err != nil {
			return fmt.Errorf("error updating source file %s: %v", sourceFile, err)
		}

		if !quiet {
			fmt.Printf("Backup of original file created: %s\n", backupFile)
		}
	} else if !quiet {
		fmt.Printf("No modifications found in the original file, backup not created.\n")
	}

	return nil
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
		fileTime, err := time.Parse("2006-01-02", dateStr)
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
func cmdNew(rootDir, templateFile string, printPath bool, config *Config, logger *Logger) error {
	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(rootDir, today)

	if _, err := os.Stat(journalPath); err == nil {
		if printPath {
			fmt.Println(journalPath)
		} else {
			fmt.Printf("Journal for today already exists: %s\n", journalPath)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(journalPath), 0o755); err != nil {
		return err
	}

	closest, err := findClosestJournalFile(rootDir, today)
	skipBackup := false
	if err != nil {
		if !printPath {
			fmt.Println("No previous journal found, creating a new one from template.")
		}

		tmpFile, err := os.CreateTemp("", "empty-journal-*.md")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(core.TodosHeader + "\n\n"); err != nil {
			return fmt.Errorf("failed to write to temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf("failed to close temp file: %w", err)
		}

		closest = tmpFile.Name()
		skipBackup = true
	}

	if !printPath {
		fmt.Printf("Using '%s' as source to create new journal for today.\n", closest)
	}

	if err := processJournal(closest, journalPath, templateFile, today, skipBackup, printPath, config, logger); err != nil {
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
