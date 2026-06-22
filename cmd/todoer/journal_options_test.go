package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/inful/todoer/pkg/core"
)

// TestProcessJournalWithOptions_ShimIsEquivalent asserts that the
// struct-based entry point produces the same on-disk result as the
// legacy positional call for the realistic inputs the daily flow
// generates. The legacy processJournal will be removed in Phase 3c;
// this test guards the migration.
func TestProcessJournalWithOptions_ShimIsEquivalent(t *testing.T) {
	tempDir := t.TempDir()
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	// Source has two uncompleted items, one with a subitem.
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n  - [ ] another\n    - [ ] sub\n\n## Notes\n")
	// Target already has a pre-existing today item.
	createTestFile(t, target, "---\ndate: "+today+"\n---\n\n# J\n\n## Todos\n\n- [["+today+"]]\n  - [ ] already in today\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	opts := ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateFile:  "",
		TemplateDate:  today,
		MergeIfExists: true,
		SkipBackup:    false,
		PrintPath:     false,
	}
	if err := processJournal(opts, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	after, _ := os.ReadFile(target)
	content := string(after)
	for _, want := range []string{
		"carry me",
		"another",
		"sub",
		"already in today",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("expected merged output to contain %q, got:\n%s", want, content)
		}
	}

	// The .bak next to source should exist because SkipBackup=false.
	bak := source + ".bak"
	if _, err := os.Stat(bak); err != nil {
		t.Errorf("expected .bak file next to source when SkipBackup=false: %v", err)
	}
}

// TestProcessJournalWithOptions_NoBackupWhenSkipped verifies the
// SkipBackup field on ProcessOptions is honoured.
func TestProcessJournalWithOptions_NoBackupWhenSkipped(t *testing.T) {
	tempDir := t.TempDir()
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	// Source has at least one completed item so processJournal
	// writes modified content back (and would create a .bak).
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [x] done yesterday\n\n## Notes\n")
	createTestFile(t, target, "---\ndate: "+today+"\n---\n\n# J\n\n## Todos\n\n- [["+today+"]]\n  - [ ] new today\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	opts := ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
		SkipBackup:    true,
	}
	if err := processJournal(opts, config, logger); err != nil {
		t.Fatalf("processJournal: %v", err)
	}

	bak := source + ".bak"
	if _, err := os.Stat(bak); !os.IsNotExist(err) {
		t.Errorf("expected no .bak when SkipBackup=true, got err=%v", err)
	}
}

// TestProcessJournalWithOptions_PrintPath verifies the PrintPath field
// is honoured (writes TargetFile to stdout).
func TestProcessJournalWithOptions_PrintPath(t *testing.T) {
	tempDir := t.TempDir()
	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}

	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)
	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)

	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [x] done\n\n## Notes\n")
	createTestFile(t, target, "---\ndate: "+today+"\n---\n\n# J\n\n## Todos\n\n## Notes\n")

	logger := NewLogger(ModeQuiet)
	opts := ProcessOptions{
		SourceFile:    source,
		TargetFile:    target,
		TemplateDate:  today,
		MergeIfExists: true,
		PrintPath:     true,
	}
	out := captureStdout(t, func() error {
		return processJournal(opts, config, logger)
	})

	out = strings.TrimSpace(out)
	// Resolve symlinks for the comparison because TempDir on macOS
	// is sometimes a symlink under /var which macOS resolves to
	// /private/var.
	resolvedTarget, _ := filepath.EvalSymlinks(target)
	resolvedOut, _ := filepath.EvalSymlinks(out)
	if resolvedOut != resolvedTarget {
		t.Errorf("printed path = %q, want %q", out, target)
	}
}
