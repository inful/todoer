package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegration(t *testing.T) {
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

			// Read the input file
			inputPath := filepath.Join(testDir, "input.md")
			inputContent, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("Failed to read input.md: %v", err)
			}

			// Use the shared template file
			templatePath := filepath.Join("testdata", "shared_template.md")

			// Process the file
			date, err := extractDateFromFrontmatter(string(inputContent))
			if err != nil {
				t.Fatalf("Failed to extract date: %v", err)
			}

			beforeTodos, todosSection, afterTodos, err := extractTodosSection(string(inputContent))
			if err != nil {
				t.Fatalf("Failed to extract TODOS section: %v", err)
			}

			// Process the TODOS section
			completedTodos, uncompletedTodos, err := processTodosSection(todosSection, date, currentDate)
			if err != nil {
				t.Fatalf("Failed to process TODOS section: %v", err)
			}

			// Replace consecutive blank lines to ensure consistent formatting
			completedTodos = normalizeBlankLines(completedTodos)
			uncompletedTodos = normalizeBlankLines(uncompletedTodos)

			// Generate the completed file content (modified input)
			completedFileContent := beforeTodos + completedTodos + afterTodos

			// Generate the uncompleted file content using template
			uncompletedFileContent, err := createFromTemplate(templatePath, uncompletedTodos, currentDate)
			if err != nil {
				t.Fatalf("Failed to create from template: %v", err)
			}

			// Read the expected files
			expectedOutputPath := filepath.Join(testDir, "expected_output.md")
			expectedOutputContent, err := os.ReadFile(expectedOutputPath)
			if err != nil {
				t.Fatalf("Failed to read expected_output.md: %v", err)
			}

			expectedInputAfterPath := filepath.Join(testDir, "expected_input_after.md")
			expectedInputAfterContent, err := os.ReadFile(expectedInputAfterPath)
			if err != nil {
				t.Fatalf("Failed to read expected_input_after.md: %v", err)
			}

			// Compare results
			if strings.TrimSpace(uncompletedFileContent) != strings.TrimSpace(string(expectedOutputContent)) {
				t.Errorf("Expected output content doesn't match. \nGot: \n%s\n\nWant: \n%s",
					uncompletedFileContent, string(expectedOutputContent))
			}

			if strings.TrimSpace(completedFileContent) != strings.TrimSpace(string(expectedInputAfterContent)) {
				t.Errorf("Expected input after processing doesn't match. \nGot: \n%s\n\nWant: \n%s",
					completedFileContent, string(expectedInputAfterContent))
			}
		})
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
