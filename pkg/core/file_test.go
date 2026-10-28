package core

import (
	"strings"
	"testing"
)

// Test ExtractDateFromFrontmatter function
func TestExtractDateFromFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		expectEmpty bool // Whether we expect no date at all
	}{
		{
			name:        "empty content returns no date",
			content:     "",
			expectError: false,
			expectEmpty: true,
		},
		{
			name: "valid frontmatter with date should extract date",
			content: `---
title: 2025-06-19
author: Test
---
Content here`,
			expectError: false,
			expectEmpty: false,
		},
		{
			name: "frontmatter without a date under the key returns empty",
			content: `---
title: Some Title
author: Test
---
Content here`,
			expectError: false,
			expectEmpty: true,
		},
		{
			name: "unparseable date under the key returns empty",
			content: `---
title: 25-06-19
author: Test
---
Content here`,
			expectError: false,
			expectEmpty: true,
		},
		{
			name:        "no frontmatter returns empty",
			content:     "Just some content without frontmatter",
			expectError: false,
			expectEmpty: true,
		},
	}

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

			if tt.expectEmpty {
				if r != "" {
					t.Errorf("Expected an empty date, got %s", r)
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
	if MovedToTemplate != "- [x] Moved to [[%s]]" {
		t.Errorf("MovedToTemplate = %q, expected %q", MovedToTemplate, "- [x] Moved to [[%s]]")
	}
}

// TestExtractDateFromFrontmatter_OtherKeyIsTried pins that a journal whose
// frontmatter uses the other common key is still dated. The built-in
// template writes both "title" and "date"; a hand-written journal usually
// has only one.
func TestExtractDateFromFrontmatter_OtherKeyIsTried(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "configured key missing, date key present",
			content: "---\ndate: 2025-06-19\n---\nBody",
			want:    "2025-06-19",
		},
		{
			name:    "configured key missing, title key present",
			content: "---\ntitle: 2025-06-19\n---\nBody",
			want:    "2025-06-19",
		},
		{
			name:    "no date under any key",
			content: "---\nauthor: Someone\n---\nBody",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractDateFromFrontmatter(tt.content, "title")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractDateFromFrontmatter_OnlyTheFrontmatterBlock pins that a date in
// the body is never taken for the file's date. The old regex matched body
// lines such as "created_date: 2019-01-01".
func TestExtractDateFromFrontmatter_OnlyTheFrontmatterBlock(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "body line ending in _date is ignored",
			content: "---\nauthor: Me\n---\n\ncreated_date: 2019-01-01\n\n---\n\n## Todos\n\n- [ ] x\n",
			want:    "",
		},
		{
			name:    "body line with a date key is ignored",
			content: "---\ntitle: Some Title\n---\n\n## Notes\n\ndate: 2019-01-01\n\n---\n\nend\n",
			want:    "",
		},
		{
			name:    "the frontmatter date wins over a body date",
			content: "---\ndate: 2025-06-20\n---\n\ncreated_date: 2019-01-01\n",
			want:    "2025-06-20",
		},
		{
			name:    "no frontmatter at all",
			content: "# Title\n\ndate: 2019-01-01\n",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractDateFromFrontmatter(tt.content, "date")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
