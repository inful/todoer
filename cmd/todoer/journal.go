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

// getGenerator builds a Generator from CLI/config, resolving template and
// previous date. The previous date is supplied by the caller (typically
// extracted from the source file's frontmatter) so that processJournal can
// read the source file exactly once.
func getGenerator(templateFile, templateDate, previousDate string, config *Config) (*generator.Generator, string, error) {
	if templateDate == "" {
		templateDate = time.Now().Format(core.DateFormat)
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

// ProcessOptions bundles the configuration for processJournal. It
// replaces the legacy 9-parameter signature with a single struct so
// call sites are self-documenting and the three boolean toggles
// (SkipBackup, PrintPath, MergeIfExists) cannot be mis-ordered at
// the call site.
type ProcessOptions struct {
	// SourceFile is the journal file to read from.
	SourceFile string
	// TargetFile is the journal file to write to.
	TargetFile string
	// TemplateFile is the user template path; empty selects the
	// embedded default template (or the config-dir template if
	// present).
	TemplateFile string
	// TemplateDate is the logical date for template rendering; empty
	// means today.
	TemplateDate string
	// MergeIfExists, when true and TargetFile already exists,
	// splices the source's uncompleted items into the target's
	// todos via core.MergeCarryover instead of overwriting.
	MergeIfExists bool
	// SkipSourceWrite suppresses the rewrite of the source file. It is
	// set when the source is a throwaway — the empty journal used when
	// there is no previous journal — so the log does not name a file
	// that is deleted right after.
	SkipSourceWrite bool
	// PrintPath, when true, writes TargetFile to stdout on success
	// so the command can be composed in shell pipelines.
	PrintPath bool
}

// processJournal processes a journal file, writing the target and
// optionally updating the source with a backup. When MergeIfExists is
// true and the target file already exists, the source's uncompleted
// items are merged into the target's existing todos via
// core.MergeCarryover instead of overwriting the target. When
// MergeIfExists is false, the target is overwritten in full (or
// created if missing).
//
// The source file is read exactly once. The previous date for template
// rendering is extracted from that single read, and the same bytes are
// handed to the generator. Writes are ordered source-first, then
// target, so a failed source write leaves the target untouched.
func processJournal(opts ProcessOptions, config *Config, logger *Logger) error {
	sourceFile := opts.SourceFile
	targetFile := opts.TargetFile
	templateFile := opts.TemplateFile
	templateDate := opts.TemplateDate
	printPath := opts.PrintPath
	mergeIfExists := opts.MergeIfExists

	logger.Debug("Processing journal: source=%s, target=%s, template=%s, date=%s", sourceFile, targetFile, templateFile, templateDate)

	if err := validateProcessArgs(sourceFile, targetFile, templateDate); err != nil {
		return err
	}

	// Read the source once. All subsequent operations work from this
	// snapshot to avoid a TOCTOU window between the date extraction
	// and the content processing.
	sourceContent, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("error reading source file %s: %w", sourceFile, err)
	}

	// Extract the previous date from the same read. Errors here are
	// non-fatal: an empty PreviousDate just leaves the .PreviousDate*
	// template variables empty.
	var previousDate string
	if extracted, extractErr := core.ExtractDateFromFrontmatter(string(sourceContent), config.FrontmatterDateKey); extractErr == nil {
		previousDate = extracted
	}

	gen, templateSource, err := getGenerator(templateFile, templateDate, previousDate, config)
	if err != nil {
		return err
	}

	logger.Debug("Using template source: %s", templateSource)

	result, err := gen.Process(string(sourceContent))
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
	if mergeIfExists {
		if existing, err := os.ReadFile(targetFile); err == nil {
			merged, mergeErr := mergeIntoExistingTarget(existing, newContentBytes, config)
			if mergeErr != nil {
				return fmt.Errorf("error merging into existing target %s: %w", targetFile, mergeErr)
			}
			newContentBytes = merged
			logger.Info("Merged carryover into existing target: %s", targetFile)
		}
	}

	if opts.SkipSourceWrite {
		logger.Info("Created %s from the template", targetFile)
	} else {
		logger.Info("Successfully processed %s -> %s (template: %s)", sourceFile, targetFile, templateSource)
	}

	if printPath {
		fmt.Println(targetFile)
	}

	// Write the target first, then the source. The unfinished tasks
	// only exist in the target until it lands on disk, so rewriting
	// the source first would delete them if this write failed — and
	// 'new' skips the backup, so there would be nothing to restore
	// from. The other order can only duplicate tasks, which the
	// carryover merge already collapses.
	//
	// 'process' replaces the target wholesale, and the target may be
	// the user's real journal, so keep a copy of what was there. The
	// merge path above keeps the target's content and needs no copy.
	if !mergeIfExists {
		existing, err := os.ReadFile(targetFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error reading target file %s: %w", targetFile, err)
		}
		if err := backupFile(config.RootDir, targetFile, existing, config.backups()); err != nil {
			return fmt.Errorf("error backing up target file %s: %w", targetFile, err)
		}
	}

	if err := safeWriteFile(targetFile, newContentBytes, FilePermissions); err != nil {
		return fmt.Errorf("error writing to target file %s: %w", targetFile, err)
	}
	logger.Debug("Wrote %d bytes to target file: %s", len(newContentBytes), targetFile)

	if len(modifiedContentBytes) > 0 && !opts.SkipSourceWrite {
		if err := backupFile(config.RootDir, sourceFile, sourceContent, config.backups()); err != nil {
			return fmt.Errorf("error backing up source file %s: %w", sourceFile, err)
		}

		if err := safeWriteFile(sourceFile, modifiedContentBytes, FilePermissions); err != nil {
			return fmt.Errorf("error updating source file %s: %w", sourceFile, err)
		}

		logger.Info("Source file updated: %s", sourceFile)
	} else if !opts.SkipSourceWrite {
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
	return []byte(core.MatchLineEndings(beforeTodos+newTodos+afterTodos, string(existingTarget))), nil
}

// findClosestJournalFile returns the most recent journal before the given date.
//
// Only files where todoer itself puts them are considered: ROOT/YYYY/MM/YYYY-MM-DD.md.
// Anything else with a date-shaped name — a note in ROOT/meetings/, a copy in
// ROOT/archive/ — is not a journal, and 'new' rewrites the file it picks, so
// taking one of those would move tasks out of a file the user never meant to
// hand over. Hidden directories are skipped for the same reason.
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
			if path != rootDir && strings.HasPrefix(filepath.Base(path), ".") {
				return filepath.SkipDir
			}
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

		// Only files where todoer itself puts journals count. Comparing
		// against the path the writer would use keeps this in step with
		// buildJournalPath instead of restating the YYYY/MM layout.
		if filepath.Dir(path) != filepath.Dir(buildJournalPath(rootDir, dateStr)) {
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

// cmdNewWithOptions creates today's journal and optionally preserves a backup
// of the source journal.
//
// Early-return contract: if today's journal already exists, the function
// returns nil without re-running the carryover merge. This is intentional —
// the TUI's read-only carryover view (when today's section is empty or
// missing) depends on it. Callers that want to re-merge on every
// invocation should delete the existing file first or use the `process`
// subcommand with merge semantics.
func cmdNewWithOptions(rootDir, templateFile string, printPath bool, config *Config, logger *Logger) error {
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

	if err := ensureDirectories(config); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(journalPath), 0o755); err != nil {
		return err
	}

	closest, err := findClosestJournalFile(rootDir, today)
	tmpJournalName := ""
	skipSourceWrite := false
	if err != nil {
		logger.Info("No previous journal found, creating a new one from template.")

		// The placeholder source lives in the system temp directory, not
		// in the journals folder: if this run is killed, the leftover
		// file is cleaned up by the OS instead of sitting in the user's
		// journal tree.
		tmpFile, err := os.CreateTemp("", "todoer-empty-journal-*.md")
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
		tmpJournalName = closest
		skipSourceWrite = true
	}

	if closest == tmpJournalName {
		logger.Info("No previous journal to carry over from; creating today's journal from the template.")
	} else {
		logger.Info("Using '%s' as source to create new journal for today.", closest)
	}

	// mergeIfExists=true: the daily flow must be idempotent and merge-into-existing
	// per ADR-0001. If today's journal already exists (e.g. created by
	// hand or by an earlier run), the source's uncompleted items are
	// merged in instead of overwriting the target.
	if err := processJournal(ProcessOptions{
		SourceFile:      closest,
		TargetFile:      journalPath,
		TemplateFile:    templateFile,
		TemplateDate:    today,
		MergeIfExists:   true,
		SkipSourceWrite: skipSourceWrite,
		PrintPath:       printPath,
	}, config, logger); err != nil {
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
