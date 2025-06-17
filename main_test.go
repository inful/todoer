package main

import (
	"os"
	"strings"
	"testing"
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
			got, err := extractDateFromFrontmatter(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractDateFromFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != "" && got != tt.want {
				t.Errorf("extractDateFromFrontmatter() = %v, want %v", got, tt.want)
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
			gotBeforeTodos, gotTodosSection, gotAfterTodos, err := extractTodosSection(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractTodosSection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Compare the content regardless of exact whitespace
				if !strings.Contains(gotBeforeTodos, tt.wantBeforeTodos) {
					t.Errorf("extractTodosSection() gotBeforeTodos should contain %q, got %q", tt.wantBeforeTodos, gotBeforeTodos)
				}

				if strings.TrimSpace(gotTodosSection) != strings.TrimSpace(tt.wantTodosSection) {
					t.Errorf("extractTodosSection() gotTodosSection = %q, want %q", gotTodosSection, tt.wantTodosSection)
				}

				if tt.wantAfterTodos != "" && !strings.Contains(gotAfterTodos, tt.wantAfterTodos) {
					t.Errorf("extractTodosSection() gotAfterTodos should contain %q, got %q", tt.wantAfterTodos, gotAfterTodos)
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
		completedItem := &TodoItem{
			Completed: true,
			Text:      "Completed task",
			SubItems:  []*TodoItem{},
		}

		if !isCompleted(completedItem) {
			t.Errorf("isCompleted() should return true for a completed item with no subitems")
		}

		// A completed item with completed subitems
		completedItemWithSubitems := &TodoItem{
			Completed: true,
			Text:      "Completed task",
			SubItems: []*TodoItem{
				{
					Completed: true,
					Text:      "Completed subtask",
					SubItems:  []*TodoItem{},
				},
			},
		}

		if !isCompleted(completedItemWithSubitems) {
			t.Errorf("isCompleted() should return true for a completed item with completed subitems")
		}

		// A completed item with uncompleted subitems
		mixedItem := &TodoItem{
			Completed: true,
			Text:      "Completed task",
			SubItems: []*TodoItem{
				{
					Completed: false,
					Text:      "Uncompleted subtask",
					SubItems:  []*TodoItem{},
				},
			},
		}

		if isCompleted(mixedItem) {
			t.Errorf("isCompleted() should return false for a completed item with uncompleted subitems")
		}

		// An uncompleted item
		uncompletedItem := &TodoItem{
			Completed: false,
			Text:      "Uncompleted task",
			SubItems:  []*TodoItem{},
		}

		if isCompleted(uncompletedItem) {
			t.Errorf("isCompleted() should return false for an uncompleted item")
		}
	})

	// Test hasDateTag
	t.Run("hasDateTag", func(t *testing.T) {
		if !hasDateTag("Task with tag #2025-05-13") {
			t.Errorf("hasDateTag() should return true for text with a date tag")
		}

		if hasDateTag("Task without tag") {
			t.Errorf("hasDateTag() should return false for text without a date tag")
		}

		if hasDateTag("Task with invalid tag #20250513") {
			t.Errorf("hasDateTag() should return false for text with an invalid date tag format")
		}
	})
}

// TestCreateFromTemplate tests the createFromTemplate function
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
			// Create a temporary template file
			tmpFile, err := os.CreateTemp("", "template*.md")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.template)
			if err != nil {
				t.Fatalf("Failed to write template: %v", err)
			}
			tmpFile.Close()

			// Test the function
			result, err := createFromTemplate(tmpFile.Name(), tt.todosContent, tt.currentDate)
			if err != nil {
				t.Fatalf("createFromTemplate failed: %v", err)
			}

			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("Expected:\n%s\n\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

// normalizeWhitespace removes extra whitespace for comparison
