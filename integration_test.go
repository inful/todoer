package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileBasedCases(t *testing.T) {
	// Use a fixed date for testing
	currentDate := "2025-06-17"

	// Get all test directories
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			testDir := filepath.Join("testdata", entry.Name())

			// Read the original file
			originalPath := filepath.Join(testDir, "original.md")
			originalContent, err := os.ReadFile(originalPath)
			if err != nil {
				t.Fatalf("Failed to read original.md: %v", err)
			}

			// Read the template file
			templatePath := filepath.Join(testDir, "template.md")

			// Process the file
			date, err := extractDateFromFrontmatter(string(originalContent))
			if err != nil {
				t.Fatalf("Failed to extract date: %v", err)
			}

			beforeTodos, todosSection, afterTodos, err := extractTodosSection(string(originalContent))
			if err != nil {
				t.Fatalf("Failed to extract TODOS section: %v", err)
			}

			// Always use the new format (no blank lines) for the integration tests
			completedTodos, uncompletedTodos, err := processTodosSection(todosSection, date, currentDate)
			if err != nil {
				t.Fatalf("Failed to process TODOS section: %v", err)
			}

			// Replace consecutive blank lines to ensure consistent formatting
			completedTodos = normalizeBlankLines(completedTodos)
			uncompletedTodos = normalizeBlankLines(uncompletedTodos)

			// Generate the completed file content (without template)
			completedFileContent := beforeTodos + completedTodos + afterTodos

			// Generate the uncompleted file content using template
			uncompletedFileContent, err := createFromTemplate(templatePath, uncompletedTodos, currentDate)
			if err != nil {
				t.Fatalf("Failed to create from template: %v", err)
			}

			// Read the expected files
			expectedPath := filepath.Join(testDir, "expected.md")
			expectedContent, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("Failed to read expected.md: %v", err)
			}

			expectedModifiedPath := filepath.Join(testDir, "expected_modified_original.md")
			expectedModifiedContent, err := os.ReadFile(expectedModifiedPath)
			if err != nil {
				t.Fatalf("Failed to read expected_modified_original.md: %v", err)
			}

			// Compare results
			if strings.TrimSpace(uncompletedFileContent) != strings.TrimSpace(string(expectedContent)) {
				t.Errorf("Expected content doesn't match. \nGot: \n%s\n\nWant: \n%s",
					uncompletedFileContent, string(expectedContent))
			}

			if strings.TrimSpace(completedFileContent) != strings.TrimSpace(string(expectedModifiedContent)) {
				t.Errorf("Expected modified content doesn't match. \nGot: \n%s\n\nWant: \n%s",
					completedFileContent, string(expectedModifiedContent))
			}
		})
	}
}

func TestTemplateIntegration(t *testing.T) {
	// Use a fixed date for testing
	currentDate := "2025-06-17"
	testDir := "testdata/template"

	// Read the original file
	originalPath := filepath.Join(testDir, "original.md")
	originalContent, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("Failed to read original.md: %v", err)
	}

	// Read the template file
	templatePath := filepath.Join(testDir, "template.md")

	// Process the file
	date, err := extractDateFromFrontmatter(string(originalContent))
	if err != nil {
		t.Fatalf("Failed to extract date: %v", err)
	}

	beforeTodos, todosSection, afterTodos, err := extractTodosSection(string(originalContent))
	if err != nil {
		t.Fatalf("Failed to extract TODOS section: %v", err)
	}

	completedTodos, uncompletedTodos, err := processTodosSection(todosSection, date, currentDate)
	if err != nil {
		t.Fatalf("Failed to process TODOS section: %v", err)
	}

	// Generate the completed file content (without template)
	completedFileContent := beforeTodos + completedTodos + afterTodos

	// Generate the uncompleted file content using template
	uncompletedFileContent, err := createFromTemplate(templatePath, uncompletedTodos, currentDate)
	if err != nil {
		t.Fatalf("Failed to create from template: %v", err)
	}

	// Read the expected files
	expectedPath := filepath.Join(testDir, "expected.md")
	expectedContent, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read expected.md: %v", err)
	}

	expectedModifiedPath := filepath.Join(testDir, "expected_modified_original.md")
	expectedModifiedContent, err := os.ReadFile(expectedModifiedPath)
	if err != nil {
		t.Fatalf("Failed to read expected_modified_original.md: %v", err)
	}

	// Compare results
	if strings.TrimSpace(uncompletedFileContent) != strings.TrimSpace(string(expectedContent)) {
		t.Errorf("Template-generated content doesn't match. \nGot: \n%s\n\nWant: \n%s",
			uncompletedFileContent, string(expectedContent))
	}

	if strings.TrimSpace(completedFileContent) != strings.TrimSpace(string(expectedModifiedContent)) {
		t.Errorf("Completed content doesn't match. \nGot: \n%s\n\nWant: \n%s",
			completedFileContent, string(expectedModifiedContent))
	}
}

// normalizeBlankLines replaces consecutive newlines with a single newline
// This is useful to ensure consistent formatting between day headers
func normalizeBlankLines(content string) string {
	// Replace any sequence of blank lines with a single newline for day headers
	normalized := strings.ReplaceAll(content, "\n\n- [[", "\n- [[")

	// Return the normalized content
	return normalized
}
