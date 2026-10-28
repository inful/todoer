package main

import (
	"testing"

	"github.com/inful/todoer/pkg/generator"
)

func TestNewGeneratorWithOptions(t *testing.T) {
	templateContent := `---
title: {{.Date}}
---

## Todos
{{.TODOS}}`

	templateDate := "2024-01-15"
	previousDate := "2024-01-14"
	customVars := map[string]any{
		"projectName": "Test Project",
		"version":     "1.0.0",
	}

	// Test with all options
	gen, err := generator.NewGeneratorWithOptions(
		templateContent,
		templateDate,
		generator.WithPreviousDate(previousDate),
		generator.WithCustomVariables(customVars),
	)
	if err != nil {
		t.Fatalf("Failed to create generator with options: %v", err)
	}

	// Test that we can process content
	originalContent := `---
title: 2024-01-14
---

## Todos

- [ ] Task 1
- [x] Task 2
`

	result, err := gen.Process(originalContent)
	if err != nil {
		t.Fatalf("Failed to process content: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestTemplateValidation(t *testing.T) {
	// Test invalid template syntax
	invalidTemplate := `---
title: {{.Date}}
---

## Todos
{{.Todos{{.InvalidSyntax}}`

	templateDate := "2024-01-15"

	_, err := generator.NewGeneratorWithOptions(invalidTemplate, templateDate)
	if err == nil {
		t.Fatal("Expected error for invalid template syntax")
	}

	if err.Error() == "" {
		t.Fatal("Expected non-empty error message")
	}
}
