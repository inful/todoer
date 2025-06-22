package main

import (
	"io"
	"strings"
	"testing"

	"todoer/pkg/generator"

	"github.com/spf13/afero"
)

func TestGeneratorLibraryInterface(t *testing.T) {
	// Test content
	originalContent := `---
title: "2025-01-15"
---

# Daily Journal - January 15, 2025

## Todos

- [[2025-01-14]]
  - [ ] Task 1
  - [x] Completed task
  - [ ] Task 2
    - [ ] Subtask

## Notes

Some notes here.`

	templateContent := `# New Journal - {{.Date}}

## Todos

{{.TODOS}}

## Notes

End of journal.`

	templateDate := "2025-12-25"

	// Create generator
	gen, err := generator.NewGeneratorWithOptions(templateContent, templateDate, generator.WithTodosHeader("## Todos"))
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Process the content
	result, err := gen.Process(originalContent)
	if err != nil {
		t.Fatalf("Failed to process content: %v", err)
	}

	// Read the modified original content
	modifiedOriginalBytes, err := io.ReadAll(result.ModifiedOriginal)
	if err != nil {
		t.Fatalf("Failed to read modified original: %v", err)
	}
	modifiedOriginal := string(modifiedOriginalBytes)

	// Read the new file content
	newFileBytes, err := io.ReadAll(result.NewFile)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}
	newFile := string(newFileBytes)

	// Verify that completed tasks are marked in the modified original
	if !strings.Contains(modifiedOriginal, "[x] Completed task") {
		t.Errorf("Modified original should contain the completed task. Got: %s", modifiedOriginal)
	}

	// Verify that the new file contains uncompleted tasks
	if !strings.Contains(newFile, "[ ] Task 1") {
		t.Errorf("New file should contain uncompleted Task 1. Got: %s", newFile)
	}
	if !strings.Contains(newFile, "[ ] Task 2") {
		t.Errorf("New file should contain uncompleted Task 2. Got: %s", newFile)
	}
	if !strings.Contains(newFile, "[ ] Subtask") {
		t.Errorf("New file should contain uncompleted Subtask. Got: %s", newFile)
	}

	// Verify that the template date is used in the new file
	if !strings.Contains(newFile, "# New Journal - 2025-12-25") {
		t.Errorf("New file should contain the template date. Got: %s", newFile)
	}

	// Verify that completed tasks are NOT in the new file
	if strings.Contains(newFile, "[x] Completed task") {
		t.Error("New file should not contain completed tasks")
	}
}

func TestGeneratorFromFile(t *testing.T) {
	// Use in-memory filesystem
	fs := afero.NewMemMapFs()

	// Create a template file in-memory
	templateContent := `# Generated from file - {{.Date}}

## Todos

{{.TODOS}}`
	tempTemplateFile := "/test_lib_template.md"
	err := writeStringToFile(fs, tempTemplateFile, templateContent)
	if err != nil {
		t.Fatalf("Failed to create temp template file: %v", err)
	}

	templateDate := "2025-03-15"

	// Read template content from in-memory fs and create generator
	templateContentFromFile, err := afero.ReadFile(fs, tempTemplateFile)
	if err != nil {
		t.Fatalf("Failed to read template file: %v", err)
	}

	gen, err := generator.NewGenerator(string(templateContentFromFile), templateDate)
	if err != nil {
		t.Fatalf("Failed to create generator from file: %v", err)
	}

	// Test with simple content
	originalContent := `---
title: "2025-01-01"
---

# Test

## Todos

- [[2024-12-31]]
  - [ ] Simple task`

	result, err := gen.Process(originalContent)
	if err != nil {
		t.Fatalf("Failed to process with file-based generator: %v", err)
	}

	newFileBytes, err := io.ReadAll(result.NewFile)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}
	newFile := string(newFileBytes)

	// Verify template date is used
	if !strings.Contains(newFile, "# Generated from file - 2025-03-15") {
		t.Errorf("Expected template date in output. Got: %s", newFile)
	}
}

func TestGeneratorWithCustomTodosHeader(t *testing.T) {
	customHeader := "## Tasks"
	originalContent := `---
title: "2025-01-15"
---

# Daily Journal - January 15, 2025

## Tasks

- [[2025-01-14]]
  - [ ] Custom header task
  - [x] Completed custom header task

## Notes

Some notes here.`

	templateContent := `# New Journal - {{.Date}}

## Tasks

{{.TODOS}}

## Notes

End of journal.`

	templateDate := "2025-12-25"

	gen, err := generator.NewGeneratorWithOptions(templateContent, templateDate, generator.WithTodosHeader(customHeader))
	if err != nil {
		t.Fatalf("Failed to create generator with custom header: %v", err)
	}

	result, err := gen.Process(originalContent)
	if err != nil {
		t.Fatalf("Failed to process content with custom header: %v", err)
	}

	newFileBytes, err := io.ReadAll(result.NewFile)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}
	newFile := string(newFileBytes)

	if !strings.Contains(newFile, customHeader) {
		t.Errorf("New file should contain the custom todos header '%s'. Got: %s", customHeader, newFile)
	}
	if !strings.Contains(newFile, "Custom header task") {
		t.Errorf("New file should contain the uncompleted custom header task. Got: %s", newFile)
	}
	if strings.Contains(newFile, "Completed custom header task") {
		t.Error("New file should not contain completed custom header task")
	}
}

// Helper function to write string to file using afero
func writeStringToFile(fs afero.Fs, filename, content string) error {
	return afero.WriteFile(fs, filename, []byte(content), 0644)
}
