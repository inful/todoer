package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tUnsetenv unsets an environment variable for the duration of the test,
// mirroring the t.Setenv naming convention.
func tUnsetenv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("cannot unset environment variable %s: %v", key, err)
	}
}

// TestCLICommands tests the main CLI commands by running the actual binary
func TestCLICommands(t *testing.T) {
	// Build the binary for testing
	binaryPath := filepath.Join(t.TempDir(), "todoer")
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/todoer")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build todoer binary: %v", err)
	}

	t.Run("ProcessCommand", func(t *testing.T) {
		testProcessCommand(t, binaryPath)
	})

	t.Run("NewCommand", func(t *testing.T) {
		testNewCommand(t, binaryPath)
	})

	t.Run("AddCommand", func(t *testing.T) {
		testAddCommand(t, binaryPath)
	})

	t.Run("HelpCommand", func(t *testing.T) {
		testHelpCommand(t, binaryPath)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, binaryPath)
	})

	t.Run("ConfigFile", func(t *testing.T) {
		testConfigFile(t, binaryPath)
	})

	t.Run("TemplateFeatures", func(t *testing.T) {
		testTemplateFeatures(t, binaryPath)
	})

	t.Run("EnvironmentVariables", func(t *testing.T) {
		testEnvironmentVariables(t, binaryPath)
	})

	t.Run("Concurrency", func(t *testing.T) {
		testConcurrency(t, binaryPath)
	})

	t.Run("LargeFile", func(t *testing.T) {
		testLargeFile(t, binaryPath)
	})

	t.Run("EdgeCases", func(t *testing.T) {
		testEdgeCases(t, binaryPath)
	})
}

func testProcessCommand(t *testing.T, binaryPath string) {
	tempDir := t.TempDir()

	// Create a test source file
	sourceFile := filepath.Join(tempDir, "source.md")
	sourceContent := `---
title: 2025-06-19
---

# Daily Journal

## Todos

- [[2025-06-17]]
  - [x] Review code changes #2025-06-19
  - [ ] Update documentation
  - [ ] Write unit tests
- [[2025-06-18]]
  - [ ] Plan sprint meeting
  - [x] Send weekly report #2025-06-19

## Notes

Some notes here.`

	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0o644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	targetFile := filepath.Join(tempDir, "target.md")

	// Run the process command
	cmd := exec.Command(binaryPath, "process", sourceFile, targetFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Process command failed: %v\nOutput: %s", err, output)
	}

	// Check that target file was created
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Fatalf("Target file was not created")
	}

	// Read and verify target file content
	targetContent, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	targetStr := string(targetContent)

	// Should contain uncompleted todos
	if !strings.Contains(targetStr, "Update documentation") {
		t.Error("Target file should contain uncompleted todo 'Update documentation'")
	}
	if !strings.Contains(targetStr, "Write unit tests") {
		t.Error("Target file should contain uncompleted todo 'Write unit tests'")
	}
	if !strings.Contains(targetStr, "Plan sprint meeting") {
		t.Error("Target file should contain uncompleted todo 'Plan sprint meeting'")
	}

	// Should not contain completed todos without date tags
	if strings.Contains(targetStr, "Review code changes") && !strings.Contains(targetStr, "#2025-06-19") {
		t.Error("Completed todos should be tagged with completion date")
	}

	// Check backup file was created
	backupFile := sourceFile + ".bak"
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Error("Backup file should have been created")
	}

	// Verify success message
	outputStr := string(output)
	if !strings.Contains(outputStr, "Successfully processed") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}
}

func testNewCommand(t *testing.T, binaryPath string) {
	tempDir := t.TempDir()

	// Create a previous journal file
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	year := yesterday.Format("2006")
	month := yesterday.Format("01")

	journalDir := filepath.Join(tempDir, year, month)
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("Failed to create journal directory: %v", err)
	}

	yesterdayFile := filepath.Join(journalDir, yesterday.Format("2006-01-02")+".md")
	yesterdayContent := `---
title: ` + yesterday.Format("2006-01-02") + `
---

# Daily Journal

## Todos

- [[` + yesterday.Format("2006-01-02") + `]]
  - [ ] Carry this todo forward
  - [x] Completed yesterday

## Notes

Previous day notes.`

	if err := os.WriteFile(yesterdayFile, []byte(yesterdayContent), 0o644); err != nil {
		t.Fatalf("Failed to create yesterday's journal: %v", err)
	}

	// Run the new command
	cmd := exec.Command(binaryPath, "--root-dir", tempDir, "new")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("New command failed: %v\nOutput: %s", err, output)
	}

	// Check that today's journal was created
	todayYear := today.Format("2006")
	todayMonth := today.Format("01")
	todayFile := filepath.Join(tempDir, todayYear, todayMonth, today.Format("2006-01-02")+".md")

	if _, err := os.Stat(todayFile); os.IsNotExist(err) {
		t.Fatalf("Today's journal file was not created: %s", todayFile)
	}

	// Read and verify today's journal content
	todayContent, err := os.ReadFile(todayFile)
	if err != nil {
		t.Fatalf("Failed to read today's journal: %v", err)
	}

	todayStr := string(todayContent)

	// Should contain title with today's date
	if !strings.Contains(todayStr, today.Format("2006-01-02")) {
		t.Error("Today's journal should contain today's date in title")
	}

	// Should contain carried forward todo
	if !strings.Contains(todayStr, "Carry this todo forward") {
		t.Error("Today's journal should contain uncompleted todo from yesterday")
	}

	// Should not contain completed todo
	if strings.Contains(todayStr, "Completed yesterday") && !strings.Contains(todayStr, "#"+yesterday.Format("2006-01-02")) {
		t.Error("Completed todos should be tagged with date or not included")
	}

	// Verify success message
	outputStr := string(output)
	if !strings.Contains(outputStr, "Successfully processed") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}
}

func testHelpCommand(t *testing.T, binaryPath string) {
	// Test main help
	cmd := exec.Command(binaryPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "todoer") {
		t.Error("Help should contain application name")
	}
	if !strings.Contains(outputStr, "process") {
		t.Error("Help should contain process command")
	}
	if !strings.Contains(outputStr, "new") {
		t.Error("Help should contain new command")
	}
	if !strings.Contains(outputStr, "add") {
		t.Error("Help should contain add command")
	}

	// Test subcommand help
	cmd = exec.Command(binaryPath, "process", "--help")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Process help command failed: %v", err)
	}

	outputStr = string(output)
	if !strings.Contains(outputStr, "source-file") {
		t.Error("Process help should contain source-file argument")
	}
	if !strings.Contains(outputStr, "target-file") {
		t.Error("Process help should contain target-file argument")
	}
}

func testAddCommand(t *testing.T, binaryPath string) {
	tempDir := t.TempDir()

	// Create yesterday's journal so add can trigger transfer when today's file is missing.
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	year := yesterday.Format("2006")
	month := yesterday.Format("01")

	journalDir := filepath.Join(tempDir, year, month)
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("Failed to create journal directory: %v", err)
	}

	yesterdayFile := filepath.Join(journalDir, yesterday.Format("2006-01-02")+".md")
	yesterdayContent := `---
title: ` + yesterday.Format("2006-01-02") + `
---

# Daily Journal

## Todos

- [[` + yesterday.Format("2006-01-02") + `]]
  - [ ] Carry forward item

## Notes
`

	if err := os.WriteFile(yesterdayFile, []byte(yesterdayContent), 0o644); err != nil {
		t.Fatalf("Failed to create yesterday's journal: %v", err)
	}

	newTodo := "Directly added from CLI"
	cmd := exec.Command(binaryPath, "--root-dir", tempDir, "add", "Directly", "added", "from", "CLI")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Add command failed: %v\nOutput: %s", err, output)
	}

	todayFile := filepath.Join(tempDir, today.Format("2006"), today.Format("01"), today.Format("2006-01-02")+".md")
	content, err := os.ReadFile(todayFile)
	if err != nil {
		t.Fatalf("Failed to read today's journal: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Carry forward item") {
		t.Error("Today's journal should contain carried-over todo before new add")
	}
	if !strings.Contains(contentStr, "- [ ] "+newTodo) {
		t.Error("Today's journal should contain the newly added todo")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Added todo") {
		t.Errorf("Expected add success message, got: %s", outputStr)
	}
}

func testErrorHandling(t *testing.T, binaryPath string) {
	// Test missing file
	cmd := exec.Command(binaryPath, "process", "nonexistent.md", "output.md")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "ERROR") {
		t.Errorf("Expected error message, got: %s", outputStr)
	}

	// Test same source and target file
	tempDir := t.TempDir()
	sameFile := filepath.Join(tempDir, "same.md")
	if err := os.WriteFile(sameFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command(binaryPath, "process", sameFile, sameFile)
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for same source and target file")
	}

	outputStr = string(output)
	if !strings.Contains(outputStr, "cannot be the same") {
		t.Errorf("Expected specific error message, got: %s", outputStr)
	}

	// Test invalid template date
	targetFile := filepath.Join(tempDir, "target.md")
	cmd = exec.Command(binaryPath, "process", sameFile, targetFile, "--template-date", "invalid-date")
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for invalid template date")
	}

	outputStr = string(output)
	if !strings.Contains(outputStr, "invalid template date") {
		t.Errorf("Expected invalid date error, got: %s", outputStr)
	}
}
