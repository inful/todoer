package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inful/todoer/pkg/core"
)

func TestProcessJournal_ValidationErrors(t *testing.T) {
	tempDir := setupTempDir(t)

	config := &Config{RootDir: tempDir}

	tests := []struct {
		name          string
		sourceFile    string
		targetFile    string
		templateDate  string
		expectError   bool
		errorContains string
	}{
		{
			name:          "same source and target",
			sourceFile:    "same.md",
			targetFile:    "same.md",
			expectError:   true,
			errorContains: "source and target files cannot be the same",
		},
		{
			name:          "invalid template date",
			sourceFile:    "source.md",
			targetFile:    "target.md",
			templateDate:  "invalid-date",
			expectError:   true,
			errorContains: "invalid template date",
		},
		{
			name:        "non-existent source file",
			sourceFile:  filepath.Join(tempDir, "nonexistent.md"),
			targetFile:  filepath.Join(tempDir, "target.md"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(ModeQuiet)
			err := processJournal(ProcessOptions{
				SourceFile:   tt.sourceFile,
				TargetFile:   tt.targetFile,
				TemplateDate: tt.templateDate,
			}, config, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("processJournal() expected error, got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("processJournal() error = %v, want to contain %v", err, tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("processJournal() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestProcessJournal_Success(t *testing.T) {
	tempDir := setupTempDir(t)

	// Create a valid source file with todos
	sourceContent := `---
date: 2024-01-01
---

# Daily Journal

## Todos

- [ ] Task 1
- [x] Completed task
- [ ] Task 2

## Notes

Some notes here.
`

	sourceFile := filepath.Join(tempDir, "source.md")
	targetFile := filepath.Join(tempDir, "target.md")
	createTestFile(t, sourceFile, sourceContent)

	config := &Config{RootDir: tempDir}

	logger := NewLogger(ModeQuiet)
	err := processJournal(ProcessOptions{
		SourceFile: sourceFile,
		TargetFile: targetFile,
	}, config, logger)
	if err != nil {
		t.Fatalf("processJournal() unexpected error: %v", err)
	}

	// Check that target file was created
	if _, err := os.Stat(targetFile); err != nil {
		t.Errorf("Target file was not created: %v", err)
	}

	// Check that backup was created
	backupFile := sourceFile + ".bak"
	if _, err := os.Stat(backupFile); err != nil {
		t.Errorf("Backup file was not created: %v", err)
	}
}

func TestProcessJournal_SkipBackupStillUpdatesSource(t *testing.T) {
	tempDir := setupTempDir(t)

	sourceContent := `---
date: 2024-01-01
---

# Daily Journal

## Todos

- [ ] Task to carry
- [x] Already done

## Notes
`

	sourceFile := filepath.Join(tempDir, "source.md")
	targetFile := filepath.Join(tempDir, "target.md")
	createTestFile(t, sourceFile, sourceContent)

	config := &Config{RootDir: tempDir}
	logger := NewLogger(ModeQuiet)

	if err := processJournal(ProcessOptions{
		SourceFile: sourceFile,
		TargetFile: targetFile,
		SkipBackup: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal() unexpected error: %v", err)
	}

	if _, err := os.Stat(sourceFile + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup file when skipBackup=true, got err=%v", err)
	}

	sourceAfter, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatalf("failed to read updated source file: %v", err)
	}

	sourceAfterStr := string(sourceAfter)
	if sourceAfterStr == sourceContent {
		t.Fatalf("expected source file to be updated when skipBackup=true")
	}

	if !strings.Contains(sourceAfterStr, "Moved to [[") {
		t.Fatalf("expected source file to include moved marker after update, got:\n%s", sourceAfterStr)
	}

	if _, err := os.Stat(targetFile); err != nil {
		t.Fatalf("expected target file to be created: %v", err)
	}
}

func TestFindClosestJournalFile(t *testing.T) {
	tempDir := setupTempDir(t)

	// Create some test journal files
	testFiles := []string{
		"2024/01/2024-01-01.md",
		"2024/01/2024-01-05.md",
		"2024/01/2024-01-10.md",
		"2024/01/other-file.txt", // Should be ignored
	}

	for _, file := range testFiles {
		createTestFile(t, filepath.Join(tempDir, file), "test content")
	}

	tests := []struct {
		name        string
		today       string
		expectFile  string
		expectError bool
	}{
		{
			name:       "find closest before date",
			today:      "2024-01-07",
			expectFile: "2024/01/2024-01-05.md",
		},
		{
			name:       "find closest when multiple exist",
			today:      "2024-01-15",
			expectFile: "2024/01/2024-01-10.md",
		},
		{
			name:        "no previous journals",
			today:       "2024-01-01",
			expectError: true,
		},
		{
			name:        "invalid date format",
			today:       "invalid-date",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findClosestJournalFile(tempDir, tt.today)

			if tt.expectError {
				if err == nil {
					t.Errorf("findClosestJournalFile() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("findClosestJournalFile() unexpected error: %v", err)
				}
				expectedPath := filepath.Join(tempDir, tt.expectFile)
				if result != expectedPath {
					t.Errorf("findClosestJournalFile() = %v, want %v", result, expectedPath)
				}
			}
		})
	}
}

func TestProcessJournal_PrintPath(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] Carryover\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	if err := processJournal(ProcessOptions{
		SourceFile:   source,
		TargetFile:   target,
		TemplateDate: today,
		PrintPath:    true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}
}

func TestProcessJournal_OnlyCompletedItems(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	// Source with only completed todos: nothing to carry over to today, but
	// the source's completed item is still date-tagged in the carryover
	// pipeline and the target is still written.
	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [x] Done\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	if err := processJournal(ProcessOptions{
		SourceFile:   source,
		TargetFile:   target,
		TemplateDate: today,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	// Source updated (date tag added to the completed item).
	afterSource, _ := os.ReadFile(source)
	if !strings.Contains(string(afterSource), "#"+yesterday) {
		t.Fatalf("expected source to be updated with date tag, got:\n%s", string(afterSource))
	}

	// Target exists; the completed item lives in the target's completed section,
	// not under today.
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected target to be created, got %v", err)
	}
	targetContent, _ := os.ReadFile(target)
	if strings.Contains(string(targetContent), "  - [ ]") {
		t.Fatalf("expected no uncompleted items in target, got:\n%s", string(targetContent))
	}
}

func TestProcessJournal_MergeIntoExistingTarget(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	// Source has one carryover item.
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	// Target already exists with a pre-existing today item.
	createTestFile(t, target, "---\ndate: "+today+"\n---\n\n# J\n\n## Todos\n\n- [["+today+"]]\n  - [ ] already in today\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	if err := processJournal(ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	content := string(after)
	if !strings.Contains(content, "carry me") {
		t.Fatalf("expected the carryover item to be merged in, got:\n%s", content)
	}
	if !strings.Contains(content, "already in today") {
		t.Fatalf("expected the pre-existing today item to be preserved, got:\n%s", content)
	}
}

func TestProcessJournal_MergeIsIdempotent(t *testing.T) {
	tempDir := setupTempDir(t)
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")
	// First run: target doesn't exist; created with the carryover.
	// Second run: target exists; should merge but not duplicate.

	logger := NewLogger(ModeQuiet)
	mergeOpts := ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
	}
	if err := processJournal(mergeOpts, config, logger); err != nil {
		t.Fatalf("first processJournal: %v", err)
	}
	if err := processJournal(mergeOpts, config, logger); err != nil {
		t.Fatalf("second processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	content := string(after)
	// The item should appear exactly once under the carryover day.
	day := "- [[" + yesterday + "]]"
	if _, afterDay, found := strings.Cut(content, day); found {
		if strings.Count(afterDay, "  - [ ] carry me") != 1 {
			t.Fatalf("expected exactly one 'carry me' after day header, got:\n%s", afterDay)
		}
	} else {
		t.Fatalf("expected carryover day %q in target, got:\n%s", day, content)
	}
}

func TestProcessJournal_AnnotateTargetFingerprintHelper(t *testing.T) {
	// Direct test of the helper: feed it a target without a fingerprint
	// and confirm both keys are added.
	target := "---\ndate: 2026-03-16\n---\n\n# J\n\n## Todos\n\n- [[2026-03-16]]\n  - [ ] x\n\n## Notes\n"
	source := []byte("source content for fingerprinting")
	out, err := annotateTargetWithFingerprint([]byte(target), source)
	if err != nil {
		t.Fatalf("annotateTargetWithFingerprint: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "fingerprint:") {
		t.Fatalf("expected fingerprint key, got:\n%s", got)
	}
	if strings.Contains(got, "todoer_source_fingerprint") {
		t.Fatalf("expected old field name to be gone, got:\n%s", got)
	}
	// Calling it again should be idempotent: the value is replaced
	// (not duplicated).
	out2, err := annotateTargetWithFingerprint(out, source)
	if err != nil {
		t.Fatalf("annotateTargetWithFingerprint (second call): %v", err)
	}
	if strings.Count(string(out2), "fingerprint:") != 1 {
		t.Fatalf("expected exactly one fingerprint key on idempotent call, got:\n%s", string(out2))
	}
}

func TestProcessJournal_AnnotateStripsLegacyFingerprintFields(t *testing.T) {
	// One-time migration: a target that still carries the pre-v0.4.0
	// spike fields (todoer_source_fingerprint / todoer_source_fingerprint_algo)
	// should have them stripped when the new fingerprint is written.
	// This prevents stale frontmatter from accumulating forever
	// after a user upgrades todoer with the spike enabled.
	target := "---\ndate: 2026-03-16\ntodoer_source_fingerprint: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\ntodoer_source_fingerprint_algo: sha256\n---\n\n# J\n\n## Notes\n"
	source := []byte("source content for fingerprinting")
	out, err := annotateTargetWithFingerprint([]byte(target), source)
	if err != nil {
		t.Fatalf("annotateTargetWithFingerprint: %v", err)
	}
	got := string(out)
	if strings.Contains(got, "todoer_source_fingerprint") {
		t.Fatalf("expected legacy todoer_source_fingerprint to be stripped, got:\n%s", got)
	}
	if !strings.Contains(got, "fingerprint:") {
		t.Fatalf("expected new fingerprint key to be present, got:\n%s", got)
	}
	// Non-fingerprint frontmatter fields must be preserved.
	if !strings.Contains(got, "date: 2026-03-16") {
		t.Fatalf("expected unrelated 'date' frontmatter key to be preserved, got:\n%s", got)
	}
}
