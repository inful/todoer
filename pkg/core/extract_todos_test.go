package core

import (
	"strings"
	"testing"
)

// TestExtractTodosSectionWithHeader pins the contract of the
// three-string return (beforeTodos, todosSection, afterTodos):
//
//   - beforeTodos includes the header line and the blank line that
//     follows it.
//   - todosSection is the content between the header's blank line
//     and the next section (or end of file), with surrounding
//     whitespace stripped via strings.TrimSpace.
//   - afterTodos starts at the next section header (or is empty
//     when Todos is the last section).
//
// The previous implementation used \n\n##  as the next-section
// regex, which silently dropped trailing sections when the next
// header immediately followed the mandatory blank line. The
// current implementation uses (?m)^\n##  and also handles the
// case where afterHeaderContent begins with a section header
// without a preceding newline.
func TestExtractTodosSectionWithHeader(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		header           string
		wantBeforeTodos  string
		wantTodosSection string
		wantAfterTodos   string
		wantErr          bool
	}{
		{
			name: "section with content, blank line, then next section",
			content: `---
title: 2026-01-01
---

# Title

` + TodosHeader + `

- [[2025-12-31]]
  - [ ] task

## Other Section

More content here.`,
			header:           TodosHeader,
			wantBeforeTodos:  TodosHeader + "\n\n",
			wantTodosSection: "- [[2025-12-31]]\n  - [ ] task",
			// When a blank line separates the todos content from the
			// next section, that blank line is included in
			// afterTodos so the round-trip output keeps the visual
			// separator.
			wantAfterTodos: "\n\n## Other Section\n\nMore content here.",
		},
		{
			name: "no content after Todos",
			content: `---
title: 2026-01-01
---

# Title

` + TodosHeader + `

## Other Section

More content here.`,
			header:           TodosHeader,
			wantBeforeTodos:  TodosHeader + "\n\n",
			wantTodosSection: "",
			// When the next section immediately follows the
			// mandatory blank line after the header, afterTodos
			// starts at the section header (no leading blank
			// line). The blank line was already consumed as the
			// header separator.
			wantAfterTodos: "## Other Section\n\nMore content here.",
		},
		{
			name: "Todos is the last section",
			content: `---
title: 2026-01-01
---

# Title

` + TodosHeader + `

- [[2025-12-31]]
  - [ ] task`,
			header:           TodosHeader,
			wantBeforeTodos:  TodosHeader + "\n\n",
			wantTodosSection: "- [[2025-12-31]]\n  - [ ] task",
			wantAfterTodos:   "",
		},
		{
			name: "no TODOS section at all",
			content: `---
title: 2026-01-01
---

# Title

Some content but no Todos header.`,
			header:  TodosHeader,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBefore, gotTodos, gotAfter, err := ExtractTodosSectionWithHeader(tt.content, tt.header)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// beforeTodos ends with the header line and the blank
			// line separator that follows it. The exact prefix
			// before that depends on the file content (frontmatter,
			// other sections, etc.), so we only assert the suffix.
			if !strings.HasSuffix(gotBefore, tt.wantBeforeTodos) {
				t.Errorf("beforeTodos should end with %q\ngot:  %q", tt.wantBeforeTodos, gotBefore)
			}
			if gotTodos != tt.wantTodosSection {
				t.Errorf("todosSection mismatch:\nwant: %q\ngot:  %q", tt.wantTodosSection, gotTodos)
			}
			if gotAfter != tt.wantAfterTodos {
				t.Errorf("afterTodos mismatch:\nwant: %q\ngot:  %q", tt.wantAfterTodos, gotAfter)
			}
		})
	}
}
