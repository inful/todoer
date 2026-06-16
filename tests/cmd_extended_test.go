package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testConfigFile(t *testing.T, binaryPath string) {
	tempDir := t.TempDir()

	// Create a config file in the correct location
	configDir := filepath.Join(tempDir, "todoer")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configContent := `root_dir = "` + tempDir + `"
template_file = "` + filepath.Join(tempDir, "custom_template.md") + `"

[custom_variables]
author = "Test Author"
project = "Test Project"`

	configFile := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create custom template
	templateContent := `---
title: {{.Date}}
author: {{.Custom.author}}
project: {{.Custom.project}}
---

# {{.Custom.project}} Journal - {{.DateLong}}

## Todos
{{.TODOS}}

## Notes
Author: {{.Custom.author}}`

	templateFile := filepath.Join(tempDir, "custom_template.md")
	if err := os.WriteFile(templateFile, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	// Create source file
	sourceFile := filepath.Join(tempDir, "source.md")
	sourceContent := `---
title: 2025-06-19
---

# Daily Journal

## Todos

- [[2025-06-19]]
  - [ ] Test config functionality

## Notes`

	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0o644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	targetFile := filepath.Join(tempDir, "target.md")

	// Set config file environment variable
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		if oldConfigHome != "" {
			t.Setenv("XDG_CONFIG_HOME", oldConfigHome)
		} else {
			tUnsetenv(t, "XDG_CONFIG_HOME")
		}
	}()

	// Run with config file
	cmd := exec.Command(binaryPath, "process", sourceFile, targetFile)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Process with config failed: %v\nOutput: %s", err, output)
	}

	// Read and verify target content contains custom variables
	targetContent, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	targetStr := string(targetContent)
	if !strings.Contains(targetStr, "Test Author") {
		t.Error("Target should contain custom author variable")
	}
	if !strings.Contains(targetStr, "Test Project") {
		t.Error("Target should contain custom project variable")
	}
}

func testTemplateFeatures(t *testing.T, binaryPath string) {
	tempDir := t.TempDir()

	// Create a template with advanced features
	templateContent := `---
title: {{.Date}}
---

# Daily Journal - {{formatDate .Date "Monday, January 02, 2006"}}

**Today:** {{weekday .Date}} {{if isWeekend .Date}}🏖️{{else}}💼{{end}}

## Randomized Ideas
{{shuffle "Idea one\nIdea two\nIdea three\nIdea four"}}

## Statistics
{{if .TotalTodos}}
- Total todos: {{.TotalTodos}}
- Completed: {{.CompletedTodos}}
{{else}}
- No todos to track
{{end}}

## Todos
{{.TODOS}}

## Math Test
2 + 3 = {{add 2 3}}
10 - 4 = {{sub 10 4}}

## String Operations
{{upper "hello world"}}
{{title "test string"}}`

	templateFile := filepath.Join(tempDir, "advanced_template.md")
	if err := os.WriteFile(templateFile, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	// Create source file with some todos
	sourceFile := filepath.Join(tempDir, "source.md")
	sourceContent := `---
title: 2025-06-19
---

# Daily Journal

## Todos

- [[2025-06-19]]
  - [ ] Test template features
  - [ ] Verify functionality
  - [x] Complete this task

## Notes`

	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0o644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	targetFile := filepath.Join(tempDir, "target.md")

	// Run with advanced template
	cmd := exec.Command(binaryPath, "--template-file", templateFile, "process", sourceFile, targetFile, "--template-date", "2025-06-19")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Process with advanced template failed: %v\nOutput: %s", err, output)
	}

	// Read and verify target content
	targetContent, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	targetStr := string(targetContent)

	// Check date formatting
	if !strings.Contains(targetStr, "2025") {
		t.Error("Should contain formatted date")
	}

	// Check weekday function
	expectedWeekday := time.Date(2025, 6, 19, 0, 0, 0, 0, time.UTC).Weekday().String()
	if !strings.Contains(targetStr, expectedWeekday) {
		t.Errorf("Should contain weekday: %s", expectedWeekday)
	}

	// Check shuffle function (verify different ideas are present)
	ideaCount := 0
	ideas := []string{"Idea one", "Idea two", "Idea three", "Idea four"}
	for _, idea := range ideas {
		if strings.Contains(targetStr, idea) {
			ideaCount++
		}
	}
	if ideaCount != 4 {
		t.Errorf("Should contain all 4 shuffled ideas, found %d", ideaCount)
	}

	// Check todo statistics
	if !strings.Contains(targetStr, "Total todos: 2") {
		t.Error("Should show correct total todos count")
	}
	if !strings.Contains(targetStr, "Completed: 1") {
		t.Error("Should show correct completed todos count")
	}

	// Check arithmetic functions
	if !strings.Contains(targetStr, "2 + 3 = 5") {
		t.Error("Should contain correct addition result")
	}
	if !strings.Contains(targetStr, "10 - 4 = 6") {
		t.Error("Should contain correct subtraction result")
	}

	// Check string operations
	if !strings.Contains(targetStr, "HELLO WORLD") {
		t.Error("Should contain uppercase string")
	}
	if !strings.Contains(targetStr, "Test String") {
		t.Error("Should contain title case string")
	}

	t.Logf("Advanced template test completed successfully")
}

func testEnvironmentVariables(t *testing.T, binaryPath string) {
	tempDir := t.TempDir()

	// Create test template
	templateContent := `---
title: {{.Date}}
---

# Custom Template Journal

## Todos
{{.TODOS}}

## Notes from ENV`

	templateFile := filepath.Join(tempDir, "env_template.md")
	if err := os.WriteFile(templateFile, []byte(templateContent), 0o644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	// Create source file
	sourceFile := filepath.Join(tempDir, "source.md")
	sourceContent := `---
title: 2025-06-19
---

# Daily Journal

## Todos

- [[2025-06-19]]
  - [ ] Test environment variables

## Notes`

	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0o644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	targetFile := filepath.Join(tempDir, "target.md")

	// Test with environment variables
	cmd := exec.Command(binaryPath, "process", sourceFile, targetFile)
	cmd.Env = append(os.Environ(),
		"TODOER_TEMPLATE_FILE="+templateFile,
		"TODOER_ROOT_DIR="+tempDir,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Process with env vars failed: %v\nOutput: %s", err, output)
	}

	// Verify the custom template was used
	targetContent, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	if !strings.Contains(string(targetContent), "Custom Template Journal") {
		t.Error("Should have used custom template from environment variable")
	}
	if !strings.Contains(string(targetContent), "Notes from ENV") {
		t.Error("Should contain custom template content")
	}
}

func testConcurrency(t *testing.T, binaryPath string) {
	tempDir := t.TempDir()

	// Create multiple source files
	sourceFiles := make([]string, 5)
	targetFiles := make([]string, 5)

	for i := range 5 {
		sourceFile := filepath.Join(tempDir, fmt.Sprintf("source%d.md", i))
		targetFile := filepath.Join(tempDir, fmt.Sprintf("target%d.md", i))

		sourceContent := fmt.Sprintf(`---
title: 2025-06-19
---

# Daily Journal %d

## Todos

- [[2025-06-19]]
  - [ ] Concurrent test %d

## Notes`, i, i)

		if err := os.WriteFile(sourceFile, []byte(sourceContent), 0o644); err != nil {
			t.Fatalf("Failed to create source file %d: %v", i, err)
		}

		sourceFiles[i] = sourceFile
		targetFiles[i] = targetFile
	}

	// Run processes concurrently
	type result struct {
		index int
		err   error
	}

	results := make(chan result, 5)

	for i := range 5 {
		go func(index int) {
			cmd := exec.Command(binaryPath, "process", sourceFiles[index], targetFiles[index])
			err := cmd.Run()
			results <- result{index: index, err: err}
		}(i)
	}

	// Collect results
	for range 5 {
		res := <-results
		if res.err != nil {
			t.Errorf("Concurrent process %d failed: %v", res.index, res.err)
		}

		// Verify target file was created
		if _, err := os.Stat(targetFiles[res.index]); os.IsNotExist(err) {
			t.Errorf("Target file %d was not created", res.index)
		}
	}
}

func testLargeFile(t *testing.T, binaryPath string) {
	tempDir := t.TempDir()

	// Create a large source file
	sourceFile := filepath.Join(tempDir, "large_source.md")
	var sourceContent strings.Builder

	sourceContent.WriteString(`---
title: 2025-06-19
---

# Large Daily Journal

## Todos

`)

	// Add many todo items
	for i := range 1000 {
		completed := "[ ]"
		if i%3 == 0 {
			completed = "[x]"
		}
		fmt.Fprintf(&sourceContent, "- [[2025-06-19]]\n  - %s Large todo item %d\n", completed, i)
	}

	sourceContent.WriteString("\n## Notes\n\nLarge journal with many todos.\n")

	if err := os.WriteFile(sourceFile, []byte(sourceContent.String()), 0o644); err != nil {
		t.Fatalf("Failed to create large source file: %v", err)
	}

	targetFile := filepath.Join(tempDir, "large_target.md")

	// Process the large file
	start := time.Now()
	cmd := exec.Command(binaryPath, "process", sourceFile, targetFile)
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Large file processing failed: %v\nOutput: %s", err, output)
	}

	// Verify reasonable performance (should complete within 10 seconds)
	if duration > 10*time.Second {
		t.Errorf("Large file processing took too long: %v", duration)
	}

	// Verify target file was created and has content
	targetContent, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read large target file: %v", err)
	}

	if len(targetContent) == 0 {
		t.Error("Target file should not be empty")
	}

	// Should contain uncompleted todos
	targetStr := string(targetContent)
	uncompletedCount := strings.Count(targetStr, "[ ] Large todo item")
	if uncompletedCount == 0 {
		t.Error("Should contain uncompleted todos")
	}

	t.Logf("Large file processing completed in %v with %d uncompleted todos", duration, uncompletedCount)
}

func testEdgeCases(t *testing.T, binaryPath string) {
	tempDir := t.TempDir()

	t.Run("EmptySourceFile", func(t *testing.T) {
		sourceFile := filepath.Join(tempDir, "empty.md")
		if err := os.WriteFile(sourceFile, []byte(""), 0o644); err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		targetFile := filepath.Join(tempDir, "empty_target.md")

		cmd := exec.Command(binaryPath, "process", sourceFile, targetFile)
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for empty source file")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "ERROR") {
			t.Errorf("Expected error message, got: %s", outputStr)
		}
	})

	t.Run("NoTodosSection", func(t *testing.T) {
		sourceFile := filepath.Join(tempDir, "no_todos.md")
		content := `---
title: 2025-06-19
---

# Daily Journal

## Notes

Just notes, no todos section.`

		if err := os.WriteFile(sourceFile, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create no todos file: %v", err)
		}

		targetFile := filepath.Join(tempDir, "no_todos_target.md")

		cmd := exec.Command(binaryPath, "process", sourceFile, targetFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Processing file without todos should not fail: %v\nOutput: %s", err, output)
		}

		// Target should be created with empty todos
		targetContent, err := os.ReadFile(targetFile)
		if err != nil {
			t.Fatalf("Failed to read target file: %v", err)
		}

		if len(targetContent) == 0 {
			t.Error("Target file should not be empty")
		}
	})

	t.Run("UnicodeContent", func(t *testing.T) {
		sourceFile := filepath.Join(tempDir, "unicode.md")
		content := `---
title: 2025-06-19
---

# 日記 (Daily Journal) 📝

## Todos

- [[2025-06-19]]
  - [ ] Unicode test: café, naïve, 中文, 🎉
  - [x] Completed: résumé

## Notes

Testing Unicode: ñáéíóú, العربية, 日本語`

		if err := os.WriteFile(sourceFile, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create unicode file: %v", err)
		}

		targetFile := filepath.Join(tempDir, "unicode_target.md")

		cmd := exec.Command(binaryPath, "process", sourceFile, targetFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Unicode processing failed: %v\nOutput: %s", err, output)
		}

		// Verify unicode is preserved
		targetContent, err := os.ReadFile(targetFile)
		if err != nil {
			t.Fatalf("Failed to read unicode target file: %v", err)
		}

		targetStr := string(targetContent)
		if !strings.Contains(targetStr, "café") {
			t.Error("Should preserve Unicode characters")
		}
		if !strings.Contains(targetStr, "🎉") {
			t.Error("Should preserve emoji characters")
		}
		if !strings.Contains(targetStr, "中文") {
			t.Error("Should preserve CJK characters")
		}
	})
}
