package main

import (
	"strings"
	"testing"

	"todoer/pkg/core"
	"todoer/pkg/generator"

	"github.com/spf13/afero"
)

// TestExtractDateFromFrontmatter tests the date extraction functionality
func TestExtractDateFromFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
		key     string
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
		{
			name:    "Date key configurable (date)",
			content: `---\ndate: 2025-06-21\n---\n`,
			want:    "2025-06-21",
			wantErr: false,
			key:     "date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "title"
			if tt.key != "" {
				key = tt.key
			}
			got, err := core.ExtractDateFromFrontmatter(tt.content, key)
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

` + core.TodosHeader + `

- [[2025-05-12]]
  - [ ] Task 1
  - [x] Task 2

## Other Section

More content
`,
			wantBeforeTodos: core.TodosHeader + `

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

` + core.TodosHeader + `

- [[2025-05-12]]
  - [ ] Task 1
  - [x] Task 2
`,
			wantBeforeTodos: core.TodosHeader + `

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
			gotBeforeTodos, gotTodosSection, gotAfterTodos, err := core.ExtractTodosSectionWithHeader(tt.content, core.TodosHeader)
			if (err != nil) != tt.wantErr {
				t.Errorf("core.ExtractTodosSectionWithHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Compare the content regardless of exact whitespace
				if !strings.Contains(gotBeforeTodos, tt.wantBeforeTodos) {
					t.Errorf("core.ExtractTodosSectionWithHeader() gotBeforeTodos should contain %q, got %q", tt.wantBeforeTodos, gotBeforeTodos)
				}

				if strings.TrimSpace(gotTodosSection) != strings.TrimSpace(tt.wantTodosSection) {
					t.Errorf("core.ExtractTodosSectionWithHeader() gotTodosSection = %q, want %q", gotTodosSection, tt.wantTodosSection)
				}

				if tt.wantAfterTodos != "" && !strings.Contains(gotAfterTodos, tt.wantAfterTodos) {
					t.Errorf("core.ExtractTodosSectionWithHeader() gotAfterTodos should contain %q, got %q", tt.wantAfterTodos, gotAfterTodos)
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

## Todos

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

## Todos

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

## Todos

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

## Todos

- [[2024-01-15]]
  - [ ] Task 1

## Notes

Notes here.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the core template API directly
			result, err := core.CreateFromTemplate(core.TemplateOptions{
				Content:      tt.template,
				TodosContent: tt.todosContent,
				CurrentDate:  tt.currentDate,
			})
			if err != nil {
				t.Fatalf("generator.CreateFromTemplateContent failed: %v", err)
			}

			if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
				t.Errorf("Expected:\n%s\n\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

// TestNewGeneratorWithOptionsAPI tests the new options-based generator API
func TestNewGeneratorWithOptionsAPI(t *testing.T) {
	templateContent := `---
title: {{.Date}}
---

# Journal {{.Date}}

## Todos
{{.TODOS}}

## Notes
Today's notes.`

	t.Run("Basic Options API", func(t *testing.T) {
		gen, err := generator.NewGeneratorWithOptions(templateContent, "2024-01-15")
		if err != nil {
			t.Fatalf("NewGeneratorWithOptions failed: %v", err)
		}

		// Test processing
		sourceContent := `---
title: 2024-01-14
---

# Daily Journal

## Todos

- [ ] Test task

## Notes`

		result, err := gen.Process(sourceContent)
		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}
	})

	t.Run("Options with Previous Date and Custom Variables", func(t *testing.T) {
		customVars := map[string]interface{}{
			"author":  "Test Author",
			"project": "Test Project",
		}

		gen, err := generator.NewGeneratorWithOptions(
			templateContent,
			"2024-01-15",
			generator.WithPreviousDate("2024-01-14"),
			generator.WithCustomVariables(customVars),
		)
		if err != nil {
			t.Fatalf("NewGeneratorWithOptions with options failed: %v", err)
		}

		// Test WithOptions method for reconfiguration
		newGen, err := gen.WithOptions(
			generator.WithPreviousDate("2024-01-13"),
		)
		if err != nil {
			t.Fatalf("WithOptions failed: %v", err)
		}

		if newGen == nil {
			t.Fatal("Expected non-nil reconfigured generator")
		}
	})

	t.Run("File-based Options API", func(t *testing.T) {
		// Use afero in-memory filesystem
		fs := afero.NewMemMapFs()
		templateFile := "/template-inmem.md"
		if err := afero.WriteFile(fs, templateFile, []byte(templateContent), 0644); err != nil {
			t.Fatalf("Failed to write template: %v", err)
		}

		templateContentFromFile, err := afero.ReadFile(fs, templateFile)
		if err != nil {
			t.Fatalf("Failed to read template: %v", err)
		}

		gen, err := generator.NewGeneratorWithOptions(string(templateContentFromFile), "2024-01-15")
		if err != nil {
			t.Fatalf("NewGeneratorFromFileWithOptions failed: %v", err)
		}

		if gen == nil {
			t.Fatal("Expected non-nil generator from file")
		}
	})
}

// normalizeWhitespace removes extra whitespace for comparison
