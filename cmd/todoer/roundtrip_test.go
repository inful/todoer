package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/inful/todoer/pkg/core"
)

// TestRewriteKeepsEveryTodoLine is the guard for the write path used by
// `add` and the TUI: both rebuild the whole todos section from the parsed
// model, so a line the parser does not understand is deleted from the
// user's file. Each case below must survive one `add`.
func TestRewriteKeepsEveryTodoLine(t *testing.T) {
	tests := []struct {
		name    string
		content string
		must    []string
	}{
		{
			name: "capital X checkbox",
			content: "## Todos\n\n" +
				"- [X] capital X done\n" +
				"- [ ] plain task\n",
			must: []string{"- [x] capital X done"},
		},
		{
			name: "cancelled checkbox",
			content: "## Todos\n\n" +
				"- [-] cancelled task\n" +
				"- [ ] plain task\n",
			must: []string{"- [x] cancelled task"},
		},
		{
			name: "mixed with a day section",
			content: "## Todos\n\n" +
				"- [[2025-06-20]]\n" +
				"  - [X] capital X under a day\n" +
				"  - [-] cancelled under a day\n",
			must: []string{"- [x] capital X under a day", "- [x] cancelled under a day"},
		},
		{
			name: "sub-list and bullets",
			content: "## Todos\n\n" +
				"- [[2025-06-20]]\n" +
				"  - [ ] parent\n" +
				"    - [ ] child\n" +
				"      - detail note\n",
			must: []string{"- [ ] parent", "- [ ] child", "- detail note"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			today := time.Now().Format(core.DateFormat)
			journalPath := buildJournalPath(tempDir, today)

			content := "---\ntitle: " + today + "\n---\n\n# J\n\n" + tt.content + "\n## Notes\n\nkeep me\n"
			createTestFile(t, journalPath, content)

			config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
			logger := NewLogger(ModeQuiet)
			if err := cmdAdd(tempDir, "", "new todo", false, false, config, logger); err != nil {
				t.Fatalf("cmdAdd: %v", err)
			}

			after, err := os.ReadFile(journalPath)
			if err != nil {
				t.Fatalf("read journal: %v", err)
			}
			got := string(after)
			for _, want := range tt.must {
				if !strings.Contains(got, want) {
					t.Errorf("line %q was lost by add; file is now:\n%s", want, got)
				}
			}
			if !strings.Contains(got, "keep me") {
				t.Errorf("content after the todos section was lost:\n%s", got)
			}
			if !strings.Contains(got, "- [ ] new todo") {
				t.Errorf("the new todo is missing:\n%s", got)
			}
		})
	}
}

// TestRewriteRejectsUnparseableLines pins the other half of the contract:
// when a line cannot be represented in the model, the command must fail
// loudly instead of writing the file without it.
func TestRewriteRejectsUnparseableLines(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(tempDir, today)

	content := "---\ntitle: " + today + "\n---\n\n# J\n\n## Todos\n\n" +
		"- [[" + today + "]]\n  - [ ] task\n\n" +
		"A loose prose line inside the todos section\n\n## Notes\n"
	createTestFile(t, journalPath, content)

	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
	logger := NewLogger(ModeQuiet)
	err := cmdAdd(tempDir, "", "new todo", false, false, config, logger)
	if err == nil {
		t.Fatal("expected an error for an unparseable line, got success")
	}
	if !strings.Contains(err.Error(), "unparseable line") {
		t.Fatalf("expected an unparseable-line error, got: %v", err)
	}

	after, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("read journal: %v", readErr)
	}
	if string(after) != content {
		t.Fatalf("file must be untouched after a failed add, got:\n%s", string(after))
	}
}

// TestProcessIsRepeatable covers the "Moved to" placeholder: a journal
// whose day had nothing completed must still process on a later run.
func TestProcessIsRepeatable(t *testing.T) {
	tempDir := setupTempDir(t)
	today := time.Now().Format(core.DateFormat)
	yesterday := time.Now().AddDate(0, 0, -1).Format(core.DateFormat)

	source := buildJournalPath(tempDir, yesterday)
	target := buildJournalPath(tempDir, today)
	createTestFile(t, source, "---\ndate: "+yesterday+"\n---\n\n# J\n\n## Todos\n\n- [["+yesterday+"]]\n  - [ ] carry me\n\n## Notes\n")

	config := &Config{RootDir: tempDir, TodosHeader: core.TodosHeader}
	logger := NewLogger(ModeQuiet)
	opts := ProcessOptions{SourceFile: source, TargetFile: target, TemplateDate: today}

	if err := processJournal(opts, config, logger); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// Second run: the source now holds the "Moved to" placeholder.
	if err := processJournal(opts, config, logger); err != nil {
		t.Fatalf("second run on the already-processed source: %v", err)
	}

	after, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if !strings.Contains(string(after), "Moved to [[") {
		t.Fatalf("expected the moved marker to survive, got:\n%s", string(after))
	}
}

// TestExtractPicksTheTodosHeaderNotAPrefix pins that a heading whose name
// merely starts with the todos header is not treated as the todos section.
func TestExtractPicksTheTodosHeaderNotAPrefix(t *testing.T) {
	content := "---\ntitle: 2025-06-20\n---\n\n" +
		"## Todos Archive\n\n- [[2025-06-19]]\n  - [ ] archived task\n\n" +
		"## Todos\n\n- [[2025-06-20]]\n  - [ ] real task\n\n## Notes\n"

	_, section, _, err := core.ExtractTodosSectionWithHeader(content, core.TodosHeader)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !strings.Contains(section, "real task") {
		t.Fatalf("expected the real todos section, got: %q", section)
	}
	if strings.Contains(section, "archived task") {
		t.Fatalf("archive section was treated as the todos section: %q", section)
	}
}
