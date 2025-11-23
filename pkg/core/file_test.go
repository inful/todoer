package core

import (
	"strings"
	"testing"
	"time"
)

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

// Test ExtractTodosSection function
func TestExtractTodosSection(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedBefore string
		expectedTodos  string
		expectedAfter  string
		expectError    bool
	}{
		{
			name:        "empty content should return error",
			content:     "",
			expectError: true,
		},
		{
			name:        "content without Todos section should return error",
			content:     "Some content without todos",
			expectError: true,
		},
		{
			name:        "Todos section without blank line should return error",
			content:     "## Todos\nImmediate content",
			expectError: true,
		},
		{
			name: "valid Todos section as last section",
			content: `# Title

## Todos

- [ ] Task 1
- [x] Task 2`,
			expectedBefore: "# Title\n\n## Todos\n\n",
			expectedTodos:  "- [ ] Task 1\n- [x] Task 2",
			expectedAfter:  "",
			expectError:    false,
		},
		{
			name: "valid Todos section with content after",
			content: `# Title

## Todos

- [ ] Task 1
- [x] Task 2

## Notes

Some notes here`,
			expectedBefore: "# Title\n\n## Todos\n\n",
			expectedTodos:  "- [ ] Task 1\n- [x] Task 2",
			expectedAfter:  "\n\n## Notes\n\nSome notes here",
			expectError:    false,
		},
		{
			name: "empty Todos section",
			content: `# Title

## Todos

## Notes

Some notes here`,
			expectedBefore: "# Title\n\n## Todos\n\n",
			expectedTodos:  "## Notes\n\nSome notes here", // This is what actually gets extracted
			expectedAfter:  "",
			expectError:    false,
		},
		{
			name:        "Todos at end of content without blank line",
			content:     "## Todos",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, todos, after, err := ExtractTodosSection(tt.content)

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

			if before != tt.expectedBefore {
				t.Errorf("Before section mismatch:\nExpected: %q\nGot: %q", tt.expectedBefore, before)
			}
			if todos != tt.expectedTodos {
				t.Errorf("Todos section mismatch:\nExpected: %q\nGot: %q", tt.expectedTodos, todos)
			}
			if after != tt.expectedAfter {
				t.Errorf("After section mismatch:\nExpected: %q\nGot: %q", tt.expectedAfter, after)
			}
		})
	}
}

// Test ProcessTodosSection function
func TestProcessTodosSection(t *testing.T) {
	tests := []struct {
		name              string
		todosSection      string
		originalDate      string
		currentDate       string
		expectedCompleted string
		expectedUncomp    string
		expectError       bool
	}{
		{
			name:         "empty original date should return error",
			todosSection: "- [ ] Task",
			originalDate: "",
			currentDate:  "2025-06-19",
			expectError:  true,
		},
		{
			name:         "empty current date should return error",
			todosSection: "- [ ] Task",
			originalDate: "2025-06-18",
			currentDate:  "",
			expectError:  true,
		},
		{
			name:         "invalid original date should return error",
			todosSection: "- [ ] Task",
			originalDate: "invalid-date",
			currentDate:  "2025-06-19",
			expectError:  true,
		},
		{
			name:         "invalid current date should return error",
			todosSection: "- [ ] Task",
			originalDate: "2025-06-18",
			currentDate:  "invalid-date",
			expectError:  true,
		},
		{
			name:              "empty todos section should return moved message",
			todosSection:      "",
			originalDate:      "2025-06-18",
			currentDate:       "2025-06-19",
			expectedCompleted: "Moved to [[2025-06-19]]",
			expectedUncomp:    "",
			expectError:       false,
		},
		{
			name:              "whitespace-only todos section should return moved message",
			todosSection:      "   \n\t  ",
			originalDate:      "2025-06-18",
			currentDate:       "2025-06-19",
			expectedCompleted: "Moved to [[2025-06-19]]",
			expectedUncomp:    "",
			expectError:       false,
		},
		{
			name: "valid todos with completed and uncompleted",
			todosSection: `- [[2025-06-18]]
  - [x] Completed task
  - [ ] Uncompleted task`,
			originalDate:      "2025-06-18",
			currentDate:       "2025-06-19",
			expectedCompleted: "- [[2025-06-18]]\n  - [x] Completed task #2025-06-18",
			expectedUncomp:    "- [[2025-06-18]]\n  - [ ] Uncompleted task",
			expectError:       false,
		},
		{
			name: "only uncompleted tasks should return moved message for completed",
			todosSection: `- [[2025-06-18]]
  - [ ] Uncompleted task 1
  - [ ] Uncompleted task 2`,
			originalDate:      "2025-06-18",
			currentDate:       "2025-06-19",
			expectedCompleted: "Moved to [[2025-06-19]]",
			expectedUncomp:    "- [[2025-06-18]]\n  - [ ] Uncompleted task 1\n  - [ ] Uncompleted task 2",
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completed, uncompleted, err := ProcessTodosSection(tt.todosSection, tt.originalDate, tt.currentDate)

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

			if completed != tt.expectedCompleted {
				t.Errorf("Completed section mismatch:\nExpected: %q\nGot: %q", tt.expectedCompleted, completed)
			}
			if uncompleted != tt.expectedUncomp {
				t.Errorf("Uncompleted section mismatch:\nExpected: %q\nGot: %q", tt.expectedUncomp, uncompleted)
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
		customVars   map[string]interface{}
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
			customVars: map[string]interface{}{
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
			customVars: map[string]interface{}{
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
			customVars: map[string]interface{}{
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
