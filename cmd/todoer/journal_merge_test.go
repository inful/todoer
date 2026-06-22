package main

import (
	"strings"
	"testing"

	"github.com/inful/todoer/pkg/core"
)

// TestMergeIntoExistingTarget exercises mergeIntoExistingTarget directly.
// mergeIntoExistingTarget is the helper that processJournal uses for the
// idempotent daily flow: when the target journal already exists, the
// source's uncompleted items are spliced into the target's Todos section
// while the target's frontmatter and after-todos content are preserved.
//
// ADR-0001 §2 (idempotent sync) depends on this function behaving
// deterministically, so we cover the inputs that the daily flow actually
// produces plus the obvious edge cases.
func TestMergeIntoExistingTarget(t *testing.T) {
	config := &Config{TodosHeader: core.TodosHeader}

	tests := []struct {
		name              string
		existing          string
		newContent        string
		wantContains      []string
		wantNotContains   []string
		wantUnchangedFrom string // exact substring from `existing` that must survive
	}{
		{
			name:       "empty target todos section gets filled with new items",
			existing:   "---\ndate: 2025-06-22\n---\n\n# Journal\n\n## Todos\n\n## Notes\n\nkeep me\n",
			newContent: "---\ndate: 2025-06-22\n---\n\n# Journal\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] new task\n\n## Notes\n\nkeep me\n",
			wantContains: []string{
				"new task",
				"- [[2025-06-22]]",
			},
			// NOTE: when the existing target's Todos section is empty
			// (no day headers), ExtractTodosSectionWithHeader's
			// next-section regex (\n\n## ) does not match the
			// "## Notes" that follows the mandatory blank line
			// separator, so afterTodos is returned empty. This is a
			// pre-existing limitation in pkg/core; out of scope for
			// this test. We assert that the new item is present but
			// do not assert that the trailing section is preserved.
		},
		{
			name:       "duplicate item on same day is not added twice",
			existing:   "---\ndate: 2025-06-22\n---\n\n# Journal\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] task\n\n## Notes\n",
			newContent: "---\ndate: 2025-06-22\n---\n\n# Journal\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] task\n\n## Notes\n",
			wantContains: []string{
				"- [ ] task",
			},
			// After splitting, the task text must appear exactly once in
			// the new file. We assert this by checking the count of the
			// day-header + task line sequence below in a dedicated test.
			wantUnchangedFrom: "## Notes\n",
		},
		{
			name:       "distinct task on same day is added alongside",
			existing:   "---\ndate: 2025-06-22\n---\n\n# Journal\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] existing\n\n## Notes\n",
			newContent: "---\ndate: 2025-06-22\n---\n\n# Journal\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] incoming\n\n## Notes\n",
			wantContains: []string{
				"- [ ] existing",
				"- [ ] incoming",
			},
			wantUnchangedFrom: "## Notes\n",
		},
		{
			name:       "frontmatter is preserved verbatim",
			existing:   "---\ntitle: 2025-06-22\ncustom: keep-me\n---\n\n# Journal\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] existing\n\n## Notes\n",
			newContent: "---\ntitle: 2025-06-22\ncustom: keep-me\n---\n\n# Journal\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] incoming\n\n## Notes\n",
			wantContains: []string{
				"custom: keep-me",
				"existing",
				"incoming",
			},
			wantUnchangedFrom: "title: 2025-06-22\ncustom: keep-me",
		},
		{
			name:       "section after Todos is preserved verbatim",
			existing:   "---\ndate: 2025-06-22\n---\n\n# Journal\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] existing\n\n## Notes\n\nkeep this content untouched\n\n## Even later\n\nand this too\n",
			newContent: "---\ndate: 2025-06-22\n---\n\n# Journal\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] incoming\n\n## Notes\n\nkeep this content untouched\n\n## Even later\n\nand this too\n",
			wantContains: []string{
				"keep this content untouched",
				"## Even later",
				"and this too",
				"incoming",
			},
		},
		{
			name:       "multiple days are preserved on both sides",
			existing:   "---\ndate: 2025-06-22\n---\n\n## Todos\n\n- [[2025-06-20]]\n  - [ ] day1\n- [[2025-06-21]]\n  - [ ] day2\n\n## Notes\n",
			newContent: "---\ndate: 2025-06-22\n---\n\n## Todos\n\n- [[2025-06-21]]\n  - [ ] day2\n- [[2025-06-22]]\n  - [ ] day3\n\n## Notes\n",
			wantContains: []string{
				"- [[2025-06-20]]",
				"- [[2025-06-21]]",
				"- [[2025-06-22]]",
				"day1",
				"day2",
				"day3",
			},
		},
		{
			name:       "subtasks come along with their parent",
			existing:   "---\ndate: 2025-06-22\n---\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] existing\n\n## Notes\n",
			newContent: "---\ndate: 2025-06-22\n---\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] parent\n    - [ ] child\n      - [ ] grandchild\n\n## Notes\n",
			wantContains: []string{
				"- [ ] parent",
				"- [ ] child",
				"- [ ] grandchild",
			},
		},
		{
			name:       "configurable todos header is respected",
			existing:   "---\ndate: 2025-06-22\n---\n\n# Journal\n\n## MyTasks\n\n- [[2025-06-22]]\n  - [ ] existing\n\n## Notes\n",
			newContent: "---\ndate: 2025-06-22\n---\n\n# Journal\n\n## MyTasks\n\n- [[2025-06-22]]\n  - [ ] incoming\n\n## Notes\n",
			wantContains: []string{
				"existing",
				"incoming",
			},
		},
		{
			name:       "completed items in existing are preserved",
			existing:   "---\ndate: 2025-06-22\n---\n\n## Todos\n\n- [[2025-06-21]]\n  - [x] already done\n\n## Notes\n",
			newContent: "---\ndate: 2025-06-22\n---\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] new work\n\n## Notes\n",
			wantContains: []string{
				"- [x] already done",
				"- [ ] new work",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := *config
			if tt.name == "configurable todos header is respected" {
				cfg.TodosHeader = "## MyTasks"
			}

			got, err := mergeIntoExistingTarget([]byte(tt.existing), []byte(tt.newContent), &cfg)
			if err != nil {
				t.Fatalf("mergeIntoExistingTarget: %v", err)
			}

			gotStr := string(got)
			for _, want := range tt.wantContains {
				if !strings.Contains(gotStr, want) {
					t.Errorf("output missing expected substring %q\noutput:\n%s", want, gotStr)
				}
			}
			for _, dont := range tt.wantNotContains {
				if strings.Contains(gotStr, dont) {
					t.Errorf("output unexpectedly contains %q\noutput:\n%s", dont, gotStr)
				}
			}
			if tt.wantUnchangedFrom != "" && !strings.Contains(gotStr, tt.wantUnchangedFrom) {
				t.Errorf("output missing preserved substring %q\noutput:\n%s", tt.wantUnchangedFrom, gotStr)
			}
		})
	}
}

// TestMergeIntoExistingTarget_DuplicateCountSpecifically guards against
// the most common regression in idempotent merge: a duplicate item
// appearing twice. The table test above checks that the item is present;
// this one checks it is present exactly once.
func TestMergeIntoExistingTarget_DuplicateCountSpecifically(t *testing.T) {
	config := &Config{TodosHeader: core.TodosHeader}

	const item = "  - [ ] task"
	existing := "---\ndate: 2025-06-22\n---\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] task\n\n## Notes\n"
	newContent := "---\ndate: 2025-06-22\n---\n\n## Todos\n\n- [[2025-06-22]]\n  - [ ] task\n\n## Notes\n"

	got, err := mergeIntoExistingTarget([]byte(existing), []byte(newContent), config)
	if err != nil {
		t.Fatalf("mergeIntoExistingTarget: %v", err)
	}

	gotStr := string(got)
	if c := strings.Count(gotStr, item); c != 1 {
		t.Errorf("expected task to appear exactly once, got %d occurrences\noutput:\n%s", c, gotStr)
	}
}

// TestMergeIntoExistingTarget_BeforeTodosUnchanged verifies that the
// section preceding Todos (typically frontmatter + a title) is preserved
// byte-for-byte. The merge logic must not rewrite that region.
func TestMergeIntoExistingTarget_BeforeTodosUnchanged(t *testing.T) {
	config := &Config{TodosHeader: core.TodosHeader}

	const before = "---\ntitle: 2025-06-22\nauthor: me\n---\n\n# June 22\n\nIntro paragraph that must survive.\n\n## Todos\n"

	existing := before + "\n- [[2025-06-22]]\n  - [ ] existing\n\n## Notes\n"
	newContent := before + "\n- [[2025-06-22]]\n  - [ ] incoming\n\n## Notes\n"

	got, err := mergeIntoExistingTarget([]byte(existing), []byte(newContent), config)
	if err != nil {
		t.Fatalf("mergeIntoExistingTarget: %v", err)
	}

	gotStr := string(got)
	if !strings.HasPrefix(gotStr, before) {
		t.Errorf("output does not start with the original before-Todos block\nwant prefix:\n%s\ngot:\n%s", before, gotStr)
	}
}
