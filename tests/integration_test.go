package main

import (
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"todoer/pkg/generator"
)

func TestIntegration(t *testing.T) {
	// Use a fixed date for testing
	currentDate := "2025-06-17"

	// Use an in-memory filesystem
	fs := afero.NewMemMapFs()

	// Copy testdata into afero fs (in a real project, you'd want a helper for this)
	copyTestdataToAferoFs(t, fs)

	// Get all test directories
	entries, err := afero.ReadDir(fs, "/testdata")
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			testDir := filepath.Join("/testdata", entry.Name())

			// Read the input file
			inputPath := filepath.Join(testDir, "input.md")
			inputContent, err := afero.ReadFile(fs, inputPath)
			if err != nil {
				t.Fatalf("Failed to read input.md: %v", err)
			}

			// Use the shared template file
			templatePath := filepath.Join("/testdata", "shared_template.md")
			templateContent, err := afero.ReadFile(fs, templatePath)
			if err != nil {
				t.Fatalf("Failed to read template file: %v", err)
			}

			// Create legacy generator
			gen, err := generator.NewGeneratorWithOptions(string(templateContent), currentDate)
			if err != nil {
				t.Fatalf("Failed to create generator: %v", err)
			}

			// Process the content using legacy processing
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
			expectedOutput, err := afero.ReadFile(fs, expectedOutputPath)
			if err != nil {
				t.Fatalf("Failed to read expected output: %v", err)
			}

			expectedInputAfterPath := filepath.Join(testDir, "expected_input_after.md")
			expectedInputAfter, err := afero.ReadFile(fs, expectedInputAfterPath)
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

// Helper to copy testdata from disk to afero fs for in-memory testing
func copyTestdataToAferoFs(t *testing.T, fsys afero.Fs) {
	diskFs := afero.NewOsFs()
	err := afero.Walk(diskFs, "testdata", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath := "/" + path // ensure leading slash for afero memfs
		if info != nil && info.IsDir() {
			return fsys.MkdirAll(relPath, 0755)
		}
		data, err := afero.ReadFile(diskFs, path)
		if err != nil {
			return err
		}
		return afero.WriteFile(fsys, relPath, data, 0644)
	})
	if err != nil {
		t.Fatalf("Failed to copy testdata to afero fs: %v", err)
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
