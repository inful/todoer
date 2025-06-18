package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"todoer/pkg/generator"
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

			// Create generator
			gen, err := generator.NewGeneratorFromFile(templatePath, currentDate)
			if err != nil {
				t.Fatalf("Failed to create generator: %v", err)
			}

			// Process the content
			result, err := gen.Process(string(inputContent))
			if err != nil {
				t.Fatalf("Failed to process content: %v", err)
			}

			// Read results
			modifiedOriginalBytes, err := io.ReadAll(result.ModifiedOriginal)
			if err != nil {
				t.Fatalf("Failed to read modified original: %v", err)
			}

			newFileBytes, err := io.ReadAll(result.NewFile)
			if err != nil {
				t.Fatalf("Failed to read new file: %v", err)
			}

			// Replace consecutive blank lines to ensure consistent formatting
			completedTodos := normalizeBlankLines(string(modifiedOriginalBytes))
			uncompletedTodos := normalizeBlankLines(string(newFileBytes))

			// Read expected outputs
			expectedOutputPath := filepath.Join(testDir, "expected_output.md")
			expectedOutput, err := os.ReadFile(expectedOutputPath)
			if err != nil {
				t.Fatalf("Failed to read expected output: %v", err)
			}

			expectedInputAfterPath := filepath.Join(testDir, "expected_input_after.md")
			expectedInputAfter, err := os.ReadFile(expectedInputAfterPath)
			if err != nil {
				t.Fatalf("Failed to read expected input after: %v", err)
			}

			expectedCompletedTodos := normalizeBlankLines(string(expectedInputAfter))
			expectedUncompletedTodos := normalizeBlankLines(string(expectedOutput))

			// Compare results
			if completedTodos != expectedCompletedTodos {
				t.Errorf("Completed todos do not match expected.\nExpected:\n%s\nActual:\n%s",
					expectedCompletedTodos, completedTodos)
			}

			if uncompletedTodos != expectedUncompletedTodos {
				t.Errorf("Uncompleted todos do not match expected.\nExpected:\n%s\nActual:\n%s",
					expectedUncompletedTodos, uncompletedTodos)
			}
		})
	}
}

// normalizeBlankLines replaces consecutive blank lines with a single blank line
func normalizeBlankLines(content string) string {
	// Split into lines
	lines := strings.Split(content, "\n")
	var result []string
	prevWasBlank := false

	for _, line := range lines {
		isBlank := strings.TrimSpace(line) == ""

		if isBlank {
			if !prevWasBlank {
				result = append(result, line)
			}
			prevWasBlank = true
		} else {
			result = append(result, line)
			prevWasBlank = false
		}
	}

	return strings.Join(result, "\n")
}
