package core

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestExtractFrontmatterMetadata(t *testing.T) {
	t.Run("extracts metadata from existing frontmatter", func(t *testing.T) {
		content := `---
title: 2026-03-17
todoer_carryover_to: 2026-03-18
todoer_carryover_updated_at: 2026-03-17T10:30:00Z
---
Body`

		metadata, hasFrontmatter, err := ExtractFrontmatterMetadata(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !hasFrontmatter {
			t.Fatalf("expected hasFrontmatter=true")
		}

		expected := map[string]string{
			"title":                       "2026-03-17",
			"todoer_carryover_to":         "2026-03-18",
			"todoer_carryover_updated_at": "2026-03-17T10:30:00Z",
		}
		if !reflect.DeepEqual(metadata, expected) {
			t.Fatalf("metadata mismatch\nexpected=%v\nactual=%v", expected, metadata)
		}
	})

	t.Run("returns no frontmatter for plain markdown", func(t *testing.T) {
		metadata, hasFrontmatter, err := ExtractFrontmatterMetadata("# Title\n\nBody")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasFrontmatter {
			t.Fatalf("expected hasFrontmatter=false")
		}
		if len(metadata) != 0 {
			t.Fatalf("expected empty metadata, got=%v", metadata)
		}
	})

	t.Run("returns error for malformed frontmatter", func(t *testing.T) {
		_, hasFrontmatter, err := ExtractFrontmatterMetadata("---\ntitle: bad\nBody")
		if err == nil {
			t.Fatalf("expected malformed frontmatter error")
		}
		if !hasFrontmatter {
			t.Fatalf("expected hasFrontmatter=true for malformed frontmatter")
		}
	})
}

func TestUpsertFrontmatterMetadata(t *testing.T) {
	t.Run("creates frontmatter when document has none", func(t *testing.T) {
		content := "# Daily Journal\n\n## Todos\n"
		updated, err := UpsertFrontmatterMetadata(content, map[string]string{
			"todoer_carryover_to":         "2026-03-18",
			"todoer_carryover_updated_at": "2026-03-17T11:00:00Z",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.HasPrefix(updated, "---\n") {
			t.Fatalf("expected frontmatter block at top, got:\n%s", updated)
		}
		if !strings.Contains(updated, "todoer_carryover_to: 2026-03-18") {
			t.Fatalf("expected carryover_to metadata, got:\n%s", updated)
		}
		if !strings.Contains(updated, "# Daily Journal") {
			t.Fatalf("expected original body preserved, got:\n%s", updated)
		}
	})

	t.Run("updates keys and preserves unrelated metadata", func(t *testing.T) {
		content := `---
title: 2026-03-17
weather: sunny
todoer_carryover_to: 2026-03-17
---
Body`

		updated, err := UpsertFrontmatterMetadata(content, map[string]string{
			"todoer_carryover_to":         "2026-03-18",
			"todoer_carryover_updated_at": "2026-03-17T11:30:00Z",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(updated, "title: 2026-03-17") || !strings.Contains(updated, "weather: sunny") {
			t.Fatalf("expected unrelated metadata preserved, got:\n%s", updated)
		}
		if strings.Count(updated, "todoer_carryover_to:") != 1 {
			t.Fatalf("expected single carryover_to key after upsert, got:\n%s", updated)
		}
		if !strings.Contains(updated, "todoer_carryover_to: 2026-03-18") {
			t.Fatalf("expected updated carryover_to value, got:\n%s", updated)
		}
		if !strings.Contains(updated, "todoer_carryover_updated_at: 2026-03-17T11:30:00Z") {
			t.Fatalf("expected new updated_at key, got:\n%s", updated)
		}
	})

	t.Run("upsert is idempotent", func(t *testing.T) {
		content := `---
title: 2026-03-17
---
Body`

		updates := map[string]string{
			"todoer_carryover_to":         "2026-03-18",
			"todoer_carryover_updated_at": "2026-03-17T12:00:00Z",
		}

		first, err := UpsertFrontmatterMetadata(content, updates)
		if err != nil {
			t.Fatalf("unexpected error on first upsert: %v", err)
		}
		second, err := UpsertFrontmatterMetadata(first, updates)
		if err != nil {
			t.Fatalf("unexpected error on second upsert: %v", err)
		}

		if first != second {
			t.Fatalf("expected idempotent upsert\nfirst:\n%s\nsecond:\n%s", first, second)
		}
	})

	t.Run("malformed frontmatter falls back to prepending new block", func(t *testing.T) {
		content := "---\ntitle: broken\n# Body"
		updated, err := UpsertFrontmatterMetadata(content, map[string]string{
			"todoer_carryover_to": "2026-03-18",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Count(updated, "---") < 2 {
			t.Fatalf("expected new valid frontmatter delimiters, got:\n%s", updated)
		}
		if !strings.Contains(updated, "todoer_carryover_to: 2026-03-18") {
			t.Fatalf("expected prepended metadata key, got:\n%s", updated)
		}
		if !strings.Contains(updated, "title: broken") {
			t.Fatalf("expected original malformed content retained, got:\n%s", updated)
		}
	})

	t.Run("CRLF line endings are preserved on roundtrip", func(t *testing.T) {
		content := "---\r\ntitle: 2026-03-17\r\n---\r\n# Body\r\n\r\nSome CRLF body text.\r\n"

		updated, err := UpsertFrontmatterMetadata(content, map[string]string{
			"todoer_carryover_to": "2026-03-18",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Every line break in the result should be CRLF. Specifically,
		// the new key we just inserted and the body separator should be
		// CRLF rather than LF.
		if !strings.Contains(updated, "\r\n") {
			t.Fatalf("expected CRLF to be preserved, got:\n%q", updated)
		}
		if strings.Contains(updated, "title: 2026-03-17\ntodoer_carryover_to") {
			t.Fatalf("expected new key line break to be CRLF, got:\n%q", updated)
		}
		if !strings.Contains(updated, "todoer_carryover_to: 2026-03-18\r\n") {
			t.Fatalf("expected CRLF after new key, got:\n%q", updated)
		}
		if !strings.Contains(updated, "---\r\n# Body\r\n") {
			t.Fatalf("expected CRLF around body, got:\n%q", updated)
		}
		if !strings.Contains(updated, "Some CRLF body text.\r\n") {
			t.Fatalf("expected CRLF at end of body, got:\n%q", updated)
		}

		// And the metadata should still be readable.
		metadata, hasFrontmatter, err := ExtractFrontmatterMetadata(updated)
		if err != nil {
			t.Fatalf("extract failed: %v", err)
		}
		if !hasFrontmatter {
			t.Fatalf("expected frontmatter to be detectable after upsert")
		}
		if metadata["todoer_carryover_to"] != "2026-03-18" {
			t.Fatalf("expected new key in metadata, got %v", metadata)
		}
		if metadata["title"] != "2026-03-17" {
			t.Fatalf("expected title preserved, got %v", metadata)
		}
	})
}

// Test ExtractDateFromFrontmatter function
func TestExtractDateFromFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		expectToday bool // Whether we expect today's date as fallback
	}{
		{
			name:        "empty content should return today's date",
			content:     "",
			expectError: false,
			expectToday: true,
		},
		{
			name: "valid frontmatter with date should extract date",
			content: `---
title: 2025-06-19
author: Test
---
Content here`,
			expectError: false,
			expectToday: false,
		},
		{
			name: "frontmatter without date should return today's date",
			content: `---
title: Some Title
author: Test
---
Content here`,
			expectError: false,
			expectToday: true,
		},
		{
			name: "invalid date format in frontmatter should return today's date",
			content: `---
title: 25-06-19
author: Test
---
Content here`,
			expectError: false,
			expectToday: true, // Actually returns today's date as fallback since regex doesn't match
		},
		{
			name:        "no frontmatter should return today's date",
			content:     "Just some content without frontmatter",
			expectError: false,
			expectToday: true,
		},
	}

	today := time.Now().Format(DateFormat)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ExtractDateFromFrontmatter(tt.content, "title")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectToday {
				if r != today {
					t.Errorf("Expected today's date %s, got %s", today, r)
				}
			} else {
				// For the valid frontmatter test case
				if r != "2025-06-19" {
					t.Errorf("Expected 2025-06-19, got %s", r)
				}
			}
		})
	}
}

// Test CreateFromTemplate basic behavior
func TestCreateFromTemplate(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		todosContent string
		currentDate  string
		expected     string
		expectError  bool
	}{
		{
			name:         "empty template should return error",
			template:     "",
			todosContent: "Some todos",
			currentDate:  "2025-06-19",
			expectError:  true,
		},
		{
			name:         "invalid date should return error",
			template:     "Date: {{.Date}}",
			todosContent: "Some todos",
			currentDate:  "invalid-date",
			expectError:  true,
		},
		{
			name:         "simple template should work",
			template:     "Date: {{.Date}}\nTodos: {{.TODOS}}",
			todosContent: "- [ ] Task 1",
			currentDate:  "2025-06-19",
			expected:     "Date: 2025-06-19\nTodos: - [ ] Task 1",
			expectError:  false,
		},
		{
			name:         "template with empty todos should clean blank lines",
			template:     "Date: {{.Date}}\n\n\nTodos:\n{{.TODOS}}\n\n\nEnd",
			todosContent: "",
			currentDate:  "2025-06-19",
			expected:     "Date: 2025-06-19\n\nTodos:\n\nEnd", // 3+ newlines get reduced to 2
			expectError:  false,
		},
		{
			name:         "template with non-empty todos should not clean lines",
			template:     "Date: {{.Date}}\n\n\nTodos:\n{{.TODOS}}\n\n\nEnd",
			todosContent: "- [ ] Task",
			currentDate:  "2025-06-19",
			expected:     "Date: 2025-06-19\n\n\nTodos:\n- [ ] Task\n\n\nEnd",
			expectError:  false,
		},
		{
			name:         "invalid template syntax should return error",
			template:     "Date: {{.Date}\nTodos: {{.TODOS}}",
			todosContent: "Some todos",
			currentDate:  "2025-06-19",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CreateFromTemplate(TemplateOptions{
				Content:      tt.template,
				TodosContent: tt.todosContent,
				CurrentDate:  tt.currentDate,
			})

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Result mismatch:\nExpected: %q\nGot: %q", tt.expected, result)
			}
		})
	}
}

// Test enhanced date variables in CreateFromTemplate
func TestCreateFromTemplateWithDateVariables(t *testing.T) {
	tests := []struct {
		name             string
		templateContent  string
		currentDate      string
		previousDate     string
		todosContent     string
		expectedContains []string
		expectError      bool
	}{
		{
			name: "template with current date variables should format correctly",
			templateContent: `Date: {{.Date}}
Short: {{.DateShort}}
Long: {{.DateLong}}
Year: {{.Year}}
Month: {{.Month}} ({{.MonthName}})
Day: {{.Day}} ({{.DayName}})
Week: {{.WeekNumber}}`,
			currentDate:  "2025-06-20",
			previousDate: "",
			todosContent: "",
			expectedContains: []string{
				"Date: 2025-06-20",
				"Short: 06/20/25",
				"Long: June 20, 2025",
				"Year: 2025",
				"Month: 06 (June)",
				"Day: 20 (Friday)",
				"Week: 25",
			},
			expectError: false,
		},
		{
			name: "template with previous date variables should format correctly",
			templateContent: `Previous: {{.PreviousDate}}
PrevShort: {{.PreviousDateShort}}
PrevLong: {{.PreviousDateLong}}
PrevYear: {{.PreviousYear}}
PrevMonth: {{.PreviousMonth}} ({{.PreviousMonthName}})
PrevDay: {{.PreviousDay}} ({{.PreviousDayName}})
PrevWeek: {{.PreviousWeekNumber}}`,
			currentDate:  "2025-06-20",
			previousDate: "2025-06-19",
			todosContent: "",
			expectedContains: []string{
				"Previous: 2025-06-19",
				"PrevShort: 06/19/25",
				"PrevLong: June 19, 2025",
				"PrevYear: 2025",
				"PrevMonth: 06 (June)",
				"PrevDay: 19 (Thursday)",
				"PrevWeek: 25",
			},
			expectError: false,
		},
		{
			name: "template with empty previous date should handle gracefully",
			templateContent: `Previous: '{{.PreviousDate}}'
PrevShort: '{{.PreviousDateShort}}'
PrevLong: '{{.PreviousDateLong}}'`,
			currentDate:  "2025-06-20",
			previousDate: "",
			todosContent: "",
			expectedContains: []string{
				"Previous: ''",
				"PrevShort: ''",
				"PrevLong: ''",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CreateFromTemplate(TemplateOptions{
				Content:      tt.templateContent,
				TodosContent: tt.todosContent,
				CurrentDate:  tt.currentDate,
				PreviousDate: tt.previousDate,
			})

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Result does not contain expected string '%s'. Result:\n%s", expected, result)
				}
			}
		})
	}
}

// Test CreateFromTemplate with statistics via TemplateOptions
func TestCreateFromTemplateWithStats(t *testing.T) {
	tests := []struct {
		name             string
		templateContent  string
		todosContent     string
		currentDate      string
		previousDate     string
		journal          *TodoJournal
		expectedContains []string
		expectError      bool
	}{
		{
			name: "template with todo statistics should render correctly",
			templateContent: `Date: {{.Date}}
Total Todos: {{.TotalTodos}}
Completed: {{.CompletedTodos}}
Oldest: {{.OldestTodoDate}}
Days Span: {{.TodoDaysSpan}}
Dates: {{range .TodoDates}}{{.}} {{end}}`,
			todosContent: "- [ ] Task 1\n- [ ] Task 2",
			currentDate:  "2025-06-20",
			previousDate: "2025-06-19",
			journal: &TodoJournal{
				Days: []*DaySection{
					{
						Date: "2025-06-18",
						Items: []*TodoItem{
							{Completed: false, Text: "Task 1"},
							{Completed: true, Text: "Done task"},
						},
					},
					{
						Date: "2025-06-19",
						Items: []*TodoItem{
							{Completed: false, Text: "Task 2"},
						},
					},
				},
			},
			expectedContains: []string{
				"Date: 2025-06-20",
				"Total Todos: 2",
				"Completed: 1",
				"Oldest: 2025-06-18",
				"Days Span: 2",
				"Dates: 2025-06-18 2025-06-19",
			},
			expectError: false,
		},
		{
			name: "template with empty journal should handle gracefully",
			templateContent: `Todos: {{.TotalTodos}}
Completed: {{.CompletedTodos}}
Oldest: {{.OldestTodoDate}}`,
			todosContent: "",
			currentDate:  "2025-06-20",
			previousDate: "",
			journal:      &TodoJournal{},
			expectedContains: []string{
				"Todos: 0",
				"Completed: 0",
				"Oldest: ",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CreateFromTemplate(TemplateOptions{
				Content:      tt.templateContent,
				TodosContent: tt.todosContent,
				CurrentDate:  tt.currentDate,
				PreviousDate: tt.previousDate,
				Journal:      tt.journal,
			})

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Result does not contain expected string '%s'. Result:\n%s", expected, result)
				}
			}
		})
	}
}

// Test CreateFromTemplate with custom variables via TemplateOptions
func TestCreateFromTemplateWithCustom(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		todos        string
		currentDate  string
		previousDate string
		journal      *TodoJournal
		customVars   map[string]any
		expected     []string // strings that should be in the result
		expectError  bool
	}{
		{
			name: "template with custom variables should render correctly",
			template: `---
date: {{.Date}}
---

# {{.Custom.ProjectName}} - {{.DateLong}}

## Summary
Version: {{.Custom.Version}}
Debug: {{.Custom.Debug}}

## Todos

{{.TODOS}}`,
			todos:       "- [ ] Test task",
			currentDate: "2025-06-20",
			journal:     &TodoJournal{},
			customVars: map[string]any{
				"ProjectName": "MyProject",
				"Version":     "1.0.0",
				"Debug":       true,
			},
			expected: []string{
				"date: 2025-06-20",
				"# MyProject - June 20, 2025",
				"Version: 1.0.0",
				"Debug: true",
				"- [ ] Test task",
			},
			expectError: false,
		},
		{
			name:        "template with invalid custom variables should fail",
			template:    `Project: {{.Custom.ProjectName}}`,
			todos:       "",
			currentDate: "2025-06-20",
			journal:     &TodoJournal{},
			customVars: map[string]any{
				"Date": "invalid", // reserved name
			},
			expectError: true,
		},
		{
			name: "template with no custom variables should work",
			template: `Date: {{.Date}}
Todos: {{.TODOS}}`,
			todos:       "- [ ] Task",
			currentDate: "2025-06-20",
			journal:     &TodoJournal{},
			customVars:  nil,
			expected: []string{
				"Date: 2025-06-20",
				"- [ ] Task",
			},
			expectError: false,
		},
		{
			name:        "template with array custom variables should work",
			template:    `Tags: {{range .Custom.Tags}}{{.}} {{end}}`,
			todos:       "",
			currentDate: "2025-06-20",
			journal:     &TodoJournal{},
			customVars: map[string]any{
				"Tags": []string{"work", "personal", "urgent"},
			},
			expected: []string{
				"Tags: work personal urgent",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CreateFromTemplate(TemplateOptions{
				Content:      tt.template,
				TodosContent: tt.todos,
				CurrentDate:  tt.currentDate,
				PreviousDate: tt.previousDate,
				Journal:      tt.journal,
				CustomVars:   tt.customVars,
			})

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Result does not contain expected string '%s'. Result:\n%s", expected, result)
				}
			}
		})
	}
}

// Test helper functions
func TestValidateProcessInputs(t *testing.T) {
	tests := []struct {
		name         string
		originalDate string
		currentDate  string
		expectError  bool
	}{
		{
			name:         "valid dates should not return error",
			originalDate: "2025-06-18",
			currentDate:  "2025-06-19",
			expectError:  false,
		},
		{
			name:         "empty original date should return error",
			originalDate: "",
			currentDate:  "2025-06-19",
			expectError:  true,
		},
		{
			name:         "empty current date should return error",
			originalDate: "2025-06-18",
			currentDate:  "",
			expectError:  true,
		},
		{
			name:         "invalid original date should return error",
			originalDate: "invalid",
			currentDate:  "2025-06-19",
			expectError:  true,
		},
		{
			name:         "invalid current date should return error",
			originalDate: "2025-06-18",
			currentDate:  "invalid",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProcessInputs(tt.originalDate, tt.currentDate)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestValidateTemplateInputs(t *testing.T) {
	tests := []struct {
		name            string
		templateContent string
		currentDate     string
		expectError     bool
	}{
		{
			name:            "valid inputs should not return error",
			templateContent: "Date: {{.Date}}",
			currentDate:     "2025-06-19",
			expectError:     false,
		},
		{
			name:            "empty template should return error",
			templateContent: "",
			currentDate:     "2025-06-19",
			expectError:     true,
		},
		{
			name:            "invalid date should return error",
			templateContent: "Date: {{.Date}}",
			currentDate:     "invalid",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplateInputs(tt.templateContent, tt.currentDate)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestExecuteTemplate(t *testing.T) {
	tests := []struct {
		name            string
		templateContent string
		data            TemplateData
		expected        string
		expectError     bool
	}{
		{
			name:            "valid template should execute correctly",
			templateContent: "Date: {{.Date}}, Todos: {{.TODOS}}",
			data:            TemplateData{Date: "2025-06-19", TODOS: "- [ ] Task", PreviousDate: "2025-06-18"},
			expected:        "Date: 2025-06-19, Todos: - [ ] Task",
			expectError:     false,
		},
		{
			name:            "invalid template syntax should return error",
			templateContent: "Date: {{.Date}",
			data:            TemplateData{Date: "2025-06-19", TODOS: "", PreviousDate: ""},
			expectError:     true,
		},
		{
			name:            "template with undefined field should return error",
			templateContent: "Date: {{.UndefinedField}}",
			data:            TemplateData{Date: "2025-06-19", TODOS: "", PreviousDate: ""},
			expectError:     true,
		},
		{
			name:            "template with PreviousDate should execute correctly",
			templateContent: "Today: {{.Date}}, From: {{.PreviousDate}}, Todos: {{.TODOS}}",
			data:            TemplateData{Date: "2025-06-19", TODOS: "- [ ] Task", PreviousDate: "2025-06-18"},
			expected:        "Today: 2025-06-19, From: 2025-06-18, Todos: - [ ] Task",
			expectError:     false,
		},
		{
			name:            "template with empty PreviousDate should work",
			templateContent: "Today: {{.Date}}, From: {{.PreviousDate}}, Todos: {{.TODOS}}",
			data:            TemplateData{Date: "2025-06-19", TODOS: "- [ ] Task", PreviousDate: ""},
			expected:        "Today: 2025-06-19, From: , Todos: - [ ] Task",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeTemplate(tt.templateContent, tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCleanExcessiveBlankLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no excessive blank lines should remain unchanged",
			input:    "Line 1\n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "three newlines should become two",
			input:    "Line 1\n\n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "many newlines should become two",
			input:    "Line 1\n\n\n\n\n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "multiple occurrences should all be cleaned",
			input:    "Line 1\n\n\nLine 2\n\n\n\nLine 3",
			expected: "Line 1\n\nLine 2\n\nLine 3",
		},
		{
			name:     "empty string should remain empty",
			input:    "",
			expected: "",
		},
		{
			name:     "only newlines should be reduced",
			input:    "\n\n\n\n",
			expected: "\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanExcessiveBlankLines(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Test constants
func TestFileConstants(t *testing.T) {
	if BlankLineSeparator != "\n\n" {
		t.Errorf("BlankLineSeparator = %q, expected %q", BlankLineSeparator, "\n\n")
	}
	if MovedToTemplate != "Moved to [[%s]]" {
		t.Errorf("MovedToTemplate = %q, expected %q", MovedToTemplate, "Moved to [[%s]]")
	}
}
