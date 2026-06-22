package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/inful/mdfp"
	"github.com/inful/todoer/pkg/core"
	"github.com/inful/todoer/pkg/generator"
)

// Fingerprint keys (ADR-0001 spike; off by default).
//
// TODOER_FINGERPRINT=1 enables the spike. When on, the daily flow
// records the source's fingerprint in the target's frontmatter, and
// on the next sync logs a 'Fingerprint mismatch' message if the
// stored value does not match the current source's fingerprint. The
// fingerprint is computed by the github.com/inful/mdfp library
// (SHA-256 of the markdown body, excluding frontmatter — see ADR-0001).
//
// TODOER_FINGERPRINT_WRITE=0 suppresses the write side of the spike
// (the frontmatter upsert) while keeping the read side (the mismatch
// check) active. This lets users detect external source changes
// without polluting their frontmatter. The default is "write enabled"
// to preserve the original behavior; the toggle is checked only when
// TODOER_FINGERPRINT is also on.
const (
	fingerprintEnabledEnv = "TODOER_FINGERPRINT"
	fingerprintWriteEnv   = "TODOER_FINGERPRINT_WRITE"
	fingerprintKeyValue   = mdfp.FingerprintField // "fingerprint"
)

// fingerprintEnabled reports whether the ADR-0001 fingerprint spike
// is turned on. The toggle is TODOER_FINGERPRINT and accepts the
// usual truthy spellings: "1", "t", "T", "true", "TRUE", "0", "f",
// "F", "false", "FALSE". Anything else (including unset) is treated
// as off.
func fingerprintEnabled() bool {
	v, err := strconv.ParseBool(os.Getenv(fingerprintEnabledEnv))
	return err == nil && v
}

// fingerprintWriteEnabled reports whether the daily flow should
// upsert the source's fingerprint into the target's frontmatter.
// It is gated on TODOER_FINGERPRINT_WRITE. The default (unset or
// unparseable) is "write enabled" to preserve the original behavior.
// The spike itself (fingerprintEnabled) is a prerequisite.
func fingerprintWriteEnabled() bool {
	v := os.Getenv(fingerprintWriteEnv)
	if v == "" {
		return true
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return parsed
}

// computeFingerprint returns the mdfp fingerprint of the given journal
// content. mdfp.CalculateFingerprintFromParts hashes a canonical
// virtual document consisting of the frontmatter (with any existing
// 'fingerprint' field stripped) plus the body. Changes to
// non-fingerprint frontmatter fields (title, date, tags, custom
// user keys) therefore DO change the fingerprint and trigger a
// 'Fingerprint mismatch' log on the next sync. The fingerprint
// field itself is stripped from the input before hashing, so
// re-running on the same body+metadata produces the same hash.
// See github.com/inful/mdfp and ADR-0001.
func computeFingerprint(content []byte) (string, error) {
	frontmatter, body, err := mdfp.ParseMarkdown(string(content))
	if err != nil {
		return "", fmt.Errorf("parse markdown for fingerprint: %w", err)
	}
	return mdfp.CalculateFingerprintFromParts(frontmatter, body), nil
}

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
	// SkipBackup, when true, suppresses the .bak copy of SourceFile
	// that processJournal would otherwise write before updating
	// the source.
	SkipBackup bool
	// PrintPath, when true, writes TargetFile to stdout on success
	// so the command can be composed in shell pipelines.
	PrintPath bool
}

// processJournalWithOptions is the struct-based entry point. It
// forwards to the legacy processJournal until Phase 3c removes the
// old signature. New code should call this instead of the legacy
// processJournal.
func processJournalWithOptions(opts ProcessOptions, config *Config, logger *Logger) error {
	return processJournal(
		opts.SourceFile,
		opts.TargetFile,
		opts.TemplateFile,
		opts.TemplateDate,
		opts.SkipBackup,
		opts.PrintPath,
		opts.MergeIfExists,
		config,
		logger,
	)
}

// processJournal processes a journal file, writing the target and optionally
// updating the source with a backup. When mergeIfExists is true and the
// target file already exists, the source's uncompleted items are merged
// into the target's existing todos via core.MergeCarryover instead of
// overwriting the target. When mergeIfExists is false, the target is
// overwritten in full (or created if missing).
//
// The source file is read exactly once. The previous date for template
// rendering is extracted from that single read, and the same bytes are
// handed to the generator. Writes are ordered source-first, then
// target, so a failed source write leaves the target untouched.
//
// Deprecated: prefer processJournalWithOptions. processJournal will be
// removed once all callers migrate in Phase 3c.
func processJournal(sourceFile, targetFile, templateFile, templateDate string, skipBackup, printPath, mergeIfExists bool, config *Config, logger *Logger) error {
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
			if fingerprintEnabled() {
				checkFingerprintMismatch(existing, modifiedContentBytes, targetFile, logger)
			}
			merged, mergeErr := mergeIntoExistingTarget(existing, newContentBytes, config)
			if mergeErr != nil {
				return fmt.Errorf("error merging into existing target %s: %w", targetFile, mergeErr)
			}
			newContentBytes = merged
			logger.Info("Merged carryover into existing target: %s", targetFile)
		}
	}

	// When the fingerprint spike is enabled, record the source's
	// fingerprint in the target's frontmatter so a future run can
	// detect external changes to the source. The fingerprint is the
	// SHA-256 of the post-processed source content (the same bytes
	// checkFingerprintMismatch compares against), so a re-run on an
	// unchanged source will match. The write side can be suppressed
	// with TODOER_FINGERPRINT_WRITE=0 to keep the read-only mismatch
	// detection without polluting the frontmatter.
	if mergeIfExists && fingerprintEnabled() && fingerprintWriteEnabled() {
		annotated, err := annotateTargetWithFingerprint(newContentBytes, modifiedContentBytes)
		if err != nil {
			return fmt.Errorf("error recording source fingerprint: %w", err)
		}
		newContentBytes = annotated
	}

	logger.Info("Successfully processed %s -> %s (template: %s)", sourceFile, targetFile, templateSource)

	if printPath {
		fmt.Println(targetFile)
	}

	// Write the source first, then the target. This ordering means a
	// failed source write leaves the target untouched, which is a
	// strictly better failure mode than the previous target-first
	// order (where a failed source write would leave the target
	// updated but the source inconsistent).
	if len(modifiedContentBytes) > 0 {
		if !skipBackup {
			backupFile := sourceFile + ".bak"
			if err := safeWriteFile(backupFile, sourceContent, FilePermissions); err != nil {
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

	logger.Debug("Writing %d bytes to target file: %s", len(newContentBytes), targetFile)
	if err := safeWriteFile(targetFile, newContentBytes, FilePermissions); err != nil {
		return fmt.Errorf("error writing to target file %s: %w", targetFile, err)
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

// checkFingerprintMismatch compares the source content's mdfp
// fingerprint to the value recorded in the existing target's
// frontmatter. If they differ, the source has changed since the last
// sync and a mismatch is logged. The function never returns an error
// or fails the sync; the fingerprint is a hint, per ADR-0001.
// Parse failures are silently treated as "no fingerprint to compare
// against" so a malformed source doesn't block the merge.
func checkFingerprintMismatch(existingTarget, sourceContent []byte, targetFile string, logger *Logger) {
	metadata, hasFM, err := core.ExtractFrontmatterMetadata(string(existingTarget))
	if err != nil || !hasFM {
		// No prior fingerprint recorded; treat as a fresh sync.
		return
	}
	stored := metadata[fingerprintKeyValue]
	if stored == "" {
		return
	}
	current, err := computeFingerprint(sourceContent)
	if err != nil {
		// Source couldn't be parsed for fingerprinting; skip the
		// mismatch check rather than failing the sync.
		return
	}
	if stored != current {
		logger.Info("Fingerprint mismatch on %s (stored=%s..., current=%s...)", targetFile, shortHash(stored), shortHash(current))
	}
}

// annotateTargetWithFingerprint upserts the source's mdfp fingerprint
// into the target's frontmatter. The key/value are added (or
// replaced) and unrelated keys are preserved. As a one-time
// migration, any pre-v0.4.0 spike fields (todoer_source_fingerprint,
// todoer_source_fingerprint_algo) are also removed so target files
// don't accumulate stale keys after upgrading. Returns the annotated
// content, or an error if the frontmatter helpers fail or the source
// cannot be parsed.
func annotateTargetWithFingerprint(targetContent, sourceContent []byte) ([]byte, error) {
	fingerprint, err := computeFingerprint(sourceContent)
	if err != nil {
		return nil, fmt.Errorf("compute source fingerprint: %w", err)
	}
	cleaned, err := core.DeleteFrontmatterMetadata(string(targetContent), []string{
		"todoer_source_fingerprint",
		"todoer_source_fingerprint_algo",
	})
	if err != nil {
		return nil, fmt.Errorf("strip legacy fingerprint fields: %w", err)
	}
	updates := map[string]string{
		fingerprintKeyValue: fingerprint,
	}
	annotated, err := core.UpsertFrontmatterMetadata(cleaned, updates)
	if err != nil {
		return nil, err
	}
	return []byte(annotated), nil
}

func shortHash(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
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

// cmdNewWithOptions creates today's journal and optionally preserves a backup
// of the source journal.
//
// Early-return contract: if today's journal already exists, the function
// returns nil without re-running the carryover merge. This is intentional —
// the TUI's read-only carryover view (when today's section is empty or
// missing) depends on it. Callers that want to re-merge on every
// invocation should delete the existing file first or use the `process`
// subcommand with merge semantics.
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

	// mergeIfExists=true: the daily flow must be idempotent and merge-into-existing
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
