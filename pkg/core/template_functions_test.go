package core

import (
	"testing"
	"text/template"
	"strings"
)

func TestTemplateFunctions(t *testing.T) {
	// Create a template with all available functions
	funcMap := createTemplateFunctions()
	
	// Test date arithmetic functions
	t.Run("Date Arithmetic Functions", func(t *testing.T) {
		tests := []struct {
			name     string
			template string
			expected string
		}{
			{
				name:     "addDays positive",
				template: `{{addDays "2025-01-15" 5}}`,
				expected: "2025-01-20",
			},
			{
				name:     "addDays negative", 
				template: `{{addDays "2025-01-15" -3}}`,
				expected: "2025-01-12",
			},
			{
				name:     "subDays",
				template: `{{subDays "2025-01-15" 10}}`,
				expected: "2025-01-05",
			},
			{
				name:     "addWeeks",
				template: `{{addWeeks "2025-01-15" 2}}`,
				expected: "2025-01-29",
			},
			{
				name:     "addMonths",
				template: `{{addMonths "2025-01-15" 3}}`,
				expected: "2025-04-15",
			},
			{
				name:     "formatDate",
				template: `{{formatDate "2025-01-15" "January 02, 2006"}}`,
				expected: "January 15, 2025",
			},
			{
				name:     "weekday",
				template: `{{weekday "2025-01-15"}}`,
				expected: "Wednesday",
			},
			{
				name:     "isWeekend Saturday",
				template: `{{isWeekend "2025-01-18"}}`,
				expected: "true",
			},
			{
				name:     "isWeekend Wednesday",
				template: `{{isWeekend "2025-01-15"}}`,
				expected: "false",
			},
			{
				name:     "daysDiff",
				template: `{{daysDiff "2025-01-15" "2025-01-20"}}`,
				expected: "5",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tmpl, err := template.New("test").Funcs(funcMap).Parse(tt.template)
				if err != nil {
					t.Fatalf("Failed to parse template: %v", err)
				}

				var result strings.Builder
				err = tmpl.Execute(&result, nil)
				if err != nil {
					t.Fatalf("Failed to execute template: %v", err)
				}

				if result.String() != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result.String())
				}
			})
		}
	})

	// Test string manipulation functions
	t.Run("String Manipulation Functions", func(t *testing.T) {
		tests := []struct {
			name     string
			template string
			expected string
		}{
			{
				name:     "upper",
				template: `{{upper "hello world"}}`,
				expected: "HELLO WORLD",
			},
			{
				name:     "lower",
				template: `{{lower "HELLO WORLD"}}`,
				expected: "hello world",
			},
			{
				name:     "title",
				template: `{{title "hello world test"}}`,
				expected: "Hello World Test",
			},
			{
				name:     "trim",
				template: `{{trim "  hello world  "}}`,
				expected: "hello world",
			},
			{
				name:     "replace",
				template: `{{replace "world" "universe" "hello world"}}`,
				expected: "hello universe",
			},
			{
				name:     "contains true",
				template: `{{contains "hello world" "world"}}`,
				expected: "true",
			},
			{
				name:     "contains false",
				template: `{{contains "hello world" "universe"}}`,
				expected: "false",
			},
			{
				name:     "hasPrefix",
				template: `{{hasPrefix "hello world" "hello"}}`,
				expected: "true",
			},
			{
				name:     "hasSuffix",
				template: `{{hasSuffix "hello world" "world"}}`,
				expected: "true",
			},
			{
				name:     "join",
				template: `{{join ", " (split " " "hello world test")}}`,
				expected: "hello, world, test",
			},
			{
				name:     "repeat",
				template: `{{repeat "abc" 3}}`,
				expected: "abcabcabc",
			},
			{
				name:     "len",
				template: `{{len "hello"}}`,
				expected: "5",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tmpl, err := template.New("test").Funcs(funcMap).Parse(tt.template)
				if err != nil {
					t.Fatalf("Failed to parse template: %v", err)
				}

				var result strings.Builder
				err = tmpl.Execute(&result, nil)
				if err != nil {
					t.Fatalf("Failed to execute template: %v", err)
				}

				if result.String() != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result.String())
				}
			})
		}
	})

	// Test utility functions
	t.Run("Utility Functions", func(t *testing.T) {
		tests := []struct {
			name     string
			template string
			data     interface{}
			expected string
		}{
			{
				name:     "default with empty value",
				template: `{{default "fallback" .EmptyVal}}`,
				data:     map[string]interface{}{"EmptyVal": ""},
				expected: "fallback",
			},
			{
				name:     "default with non-empty value",
				template: `{{default "fallback" .Value}}`,
				data:     map[string]interface{}{"Value": "actual"},
				expected: "actual",
			},
			{
				name:     "empty with empty string",
				template: `{{empty ""}}`,
				expected: "true",
			},
			{
				name:     "empty with non-empty string",
				template: `{{empty "hello"}}`,
				expected: "false",
			},
			{
				name:     "notEmpty with empty string",
				template: `{{notEmpty ""}}`,
				expected: "false",
			},
			{
				name:     "notEmpty with non-empty string",
				template: `{{notEmpty "hello"}}`,
				expected: "true",
			},
			{
				name:     "seq",
				template: `{{range seq 1 3}}{{.}} {{end}}`,
				expected: "1 2 3 ",
			},
			{
				name:     "dict",
				template: `{{$d := dict "name" "John" "age" 30}}{{$d.name}} is {{$d.age}}`,
				expected: "John is 30",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tmpl, err := template.New("test").Funcs(funcMap).Parse(tt.template)
				if err != nil {
					t.Fatalf("Failed to parse template: %v", err)
				}

				var result strings.Builder
				err = tmpl.Execute(&result, tt.data)
				if err != nil {
					t.Fatalf("Failed to execute template: %v", err)
				}

				if result.String() != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result.String())
				}
			})
		}
	})

	// Test error handling for invalid dates
	t.Run("Error Handling", func(t *testing.T) {
		tests := []struct {
			name     string
			template string
			expected string
		}{
			{
				name:     "addDays with invalid date",
				template: `{{addDays "invalid-date" 5}}`,
				expected: "invalid-date",
			},
			{
				name:     "weekday with invalid date",
				template: `{{weekday "invalid-date"}}`,
				expected: "",
			},
			{
				name:     "isWeekend with invalid date",
				template: `{{isWeekend "invalid-date"}}`,
				expected: "false",
			},
			{
				name:     "daysDiff with invalid dates",
				template: `{{daysDiff "invalid" "also-invalid"}}`,
				expected: "0",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tmpl, err := template.New("test").Funcs(funcMap).Parse(tt.template)
				if err != nil {
					t.Fatalf("Failed to parse template: %v", err)
				}

				var result strings.Builder
				err = tmpl.Execute(&result, nil)
				if err != nil {
					t.Fatalf("Failed to execute template: %v", err)
				}

				if result.String() != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result.String())
				}
			})
		}
	})
}

func TestTemplateIntegration(t *testing.T) {
	// Test a comprehensive template that uses multiple functions
	templateContent := `---
title: {{.Date}}
---

# Daily Journal - {{formatDate .Date "Monday, January 02, 2006"}}

{{if .PreviousDate}}
Previous journal: {{.PreviousDate}} ({{daysDiff .PreviousDate .Date}} days ago)
{{end}}

## Dates This Week
{{$startOfWeek := subDays .Date (len (weekday .Date))}}
{{range seq 0 6}}
{{$currentDay := addDays $startOfWeek .}}
- {{formatDate $currentDay "Mon 01/02"}}: {{if isWeekend $currentDay}}Weekend{{else}}Weekday{{end}}
{{end}}

## Todo Statistics
{{if .TotalTodos}}
Total todos: {{.TotalTodos}}
{{if .TodoDaysSpan}}Spanning {{.TodoDaysSpan}} days{{end}}
{{else}}
No todos to carry over
{{end}}

## Todos
{{.TODOS}}

## Notes
{{$today := weekday .Date}}
Today is {{$today}}. {{if isWeekend .Date}}Enjoy your weekend!{{else}}Have a productive day!{{end}}
`

	data := TemplateData{
		Date:         "2025-01-15",
		PreviousDate: "2025-01-10",
		TODOS:        "- [ ] Sample todo",
		TotalTodos:   1,
		TodoDaysSpan: 5,
	}

	funcMap := createTemplateFunctions()
	tmpl, err := template.New("test").Funcs(funcMap).Parse(templateContent)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	output := result.String()

	// Check that key elements are present
	expectedParts := []string{
		"title: 2025-01-15",
		"Daily Journal - Wednesday, January 15, 2025",
		"Previous journal: 2025-01-10 (5 days ago)",
		"Total todos: 1",
		"Spanning 5 days",
		"Today is Wednesday",
		"Have a productive day!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", part, output)
		}
	}
}
