package main

import (
	"strings"
	"testing"

	"todoer/pkg/core"
	"todoer/pkg/generator"
)

// TestExtractDateFromFrontmatter tests the date extraction functionality
func TestExtractDateFromFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name: "Valid Frontmatter",
			content: `---
title: 2025-05-13
---

# Title

Content here
`,
			want:    "2025-05-13",
			wantErr: false,
		},
		{
			name: "No Title in Frontmatter",
			content: `---
author: Me
---

# Title

Content here
`,
			want:    "", // Will be today's date
			wantErr: false,
		},
		{
			name:    "No Frontmatter",
			content: "# Title\n\nContent here",
			want:    "", // Will be today's date
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := core.ExtractDateFromFrontmatter(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("core.ExtractDateFromFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != "" && got != tt.want {
				t.Errorf("core.ExtractDateFromFrontmatter() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractTodosSection tests the extraction of the TODOS section
func TestExtractTodosSection(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		wantBeforeTodos  string
		wantTodosSection string
		wantAfterTodos   string
		wantErr          bool
	}{
		{
			name: "Valid TODOS Section",
			content: `---
title: 2025-05-13
---

# Title

Content here

## TODOS

- [[2025-05-12]]
  - [ ] Task 1
  - [x] Task 2

## Other Section

More content
`,
			wantBeforeTodos: `## TODOS

`,
			wantTodosSection: `- [[2025-05-12]]
  - [ ] Task 1
  - [x] Task 2`,
			wantAfterTodos: `## Other Section`,
			wantErr:        false,
		},
		{
			name: "No Section After TODOS",
			content: `---
title: 2025-05-13
---

# Title

Content here

## TODOS

- [[2025-05-12]]
  - [ ] Task 1
  - [x] Task 2
`,
			wantBeforeTodos: `## TODOS

`,
			wantTodosSection: `- [[2025-05-12]]
  - [ ] Task 1
  - [x] Task 2`,
			wantAfterTodos: "",
			wantErr:        false,
		},
		{
			name:             "No TODOS Section",
			content:          "# Title\n\nContent here",
			wantBeforeTodos:  "",
			wantTodosSection: "",
			wantAfterTodos:   "",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBeforeTodos, gotTodosSection, gotAfterTodos, err := core.ExtractTodosSection(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("core.ExtractTodosSection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Compare the content regardless of exact whitespace
				if !strings.Contains(gotBeforeTodos, tt.wantBeforeTodos) {
					t.Errorf("core.ExtractTodosSection() gotBeforeTodos should contain %q, got %q", tt.wantBeforeTodos, gotBeforeTodos)
				}

				if strings.TrimSpace(gotTodosSection) != strings.TrimSpace(tt.wantTodosSection) {
					t.Errorf("core.ExtractTodosSection() gotTodosSection = %q, want %q", gotTodosSection, tt.wantTodosSection)
				}

				if tt.wantAfterTodos != "" && !strings.Contains(gotAfterTodos, tt.wantAfterTodos) {
					t.Errorf("core.ExtractTodosSection() gotAfterTodos should contain %q, got %q", tt.wantAfterTodos, gotAfterTodos)
				}
			}
		})
	}
}

// TestParsingFunctions tests the helpers for parsing tasks
func TestParsingFunctions(t *testing.T) {
	// Test isCompleted
	t.Run("isCompleted", func(t *testing.T) {
		// A completed item with no subitems
		completedItem := &core.TodoItem{
			Completed: true,
			Text:      "Completed task",
			SubItems:  []*core.TodoItem{},
		}

		if !core.IsCompleted(completedItem) {
			t.Errorf("core.IsCompleted() should return true for a completed item with no subitems")
		}

		// A completed item with completed subitems
		completedItemWithSubitems := &core.TodoItem{
			Completed: true,
			Text:      "Completed task",
			SubItems: []*core.TodoItem{
				{
					Completed: true,
					Text:      "Completed subtask",
					SubItems:  []*core.TodoItem{},
				},
			},
		}

		if !core.IsCompleted(completedItemWithSubitems) {
			t.Errorf("core.IsCompleted() should return true for a completed item with completed subitems")
		}

		// A completed item with uncompleted subitems
		mixedItem := &core.TodoItem{
			Completed: true,
			Text:      "Completed task",
			SubItems: []*core.TodoItem{
				{
					Completed: false,
					Text:      "Uncompleted subtask",
					SubItems:  []*core.TodoItem{},
				},
			},
		}

		if core.IsCompleted(mixedItem) {
			t.Errorf("core.IsCompleted() should return false for a completed item with uncompleted subitems")
		}

		// An uncompleted item
		uncompletedItem := &core.TodoItem{
			Completed: false,
			Text:      "Uncompleted task",
			SubItems:  []*core.TodoItem{},
		}

		if core.IsCompleted(uncompletedItem) {
			t.Errorf("core.IsCompleted() should return false for an uncompleted item")
		}
	})

	// Test hasDateTag
	t.Run("hasDateTag", func(t *testing.T) {
		if !core.HasDateTag("Task with tag #2025-05-13") {
			t.Errorf("core.HasDateTag() should return true for text with a date tag")
		}

		if core.HasDateTag("Task without tag") {
			t.Errorf("core.HasDateTag() should return false for text without a date tag")
		}

		if core.HasDateTag("Task with invalid tag #20250513") {
			t.Errorf("core.HasDateTag() should return false for text with an invalid date tag format")
		}
	})
}

// TestCreateFromTemplate tests the template functionality through the generator package
func TestCreateFromTemplate(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		todosContent string
		currentDate  string
		expected     string
	}{
		{
			name: "template with TODOS placeholder",
			template: `---
title: {{.Date}}
---

# Journal {{.Date}}

## TODOS

{{.TODOS}}

## Notes

Today's notes.`,
			todosContent: `- [[2024-01-15]]
  - [ ] Task 1
  - [ ] Task 2`,
			currentDate: "2024-01-16",
			expected: `---
title: 2024-01-16
---

# Journal 2024-01-16

## TODOS

- [[2024-01-15]]
  - [ ] Task 1
  - [ ] Task 2

## Notes

Today's notes.`,
		},
		{
			name: "template with TODOS placeholder in section",
			template: `---
title: {{.Date}}
---

# Journal

## TODOS

{{.TODOS}}

## Notes

Notes here.`,
			todosContent: `- [[2024-01-15]]
  - [ ] Task 1`,
			currentDate: "2024-01-16",
			expected: `---
title: 2024-01-16
---

# Journal

## TODOS

- [[2024-01-15]]
  - [ ] Task 1

## Notes

Notes here.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the function via the generator package (which uses core)
			result, err := generator.CreateFromTemplateContent(tt.template, tt.todosContent, tt.currentDate)
			if err != nil {
				t.Fatalf("generator.CreateFromTemplateContent failed: %v", err)
			}

			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("Expected:\n%s\n\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

// normalizeWhitespace removes extra whitespace for comparison
