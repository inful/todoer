package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBasicCLIFunctionality tests basic CLI functionality with a simpler approach
func TestBasicCLIFunctionality(t *testing.T) {
	// Build the binary for testing
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "todoer")

	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/todoer")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build todoer binary: %v", err)
	}

	// Create a temporary config directory to avoid loading user config
	testConfigDir := filepath.Join(tempDir, "config")

	t.Run("Help", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--help")
		cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+testConfigDir)
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Help command failed: %v", err)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "todoer") {
			t.Error("Help output should contain 'todoer'")
		}
		if !strings.Contains(outputStr, "process") {
			t.Error("Help output should mention 'process' command")
		}
		if !strings.Contains(outputStr, "new") {
			t.Error("Help output should mention 'new' command")
		}
	})

	t.Run("BasicProcess", func(t *testing.T) {
		// Create a simple test source file
		sourceFile := filepath.Join(tempDir, "test_source.md")
		sourceContent := `---
title: 2025-06-20
---

# Daily Journal

## Todos

- [[2025-06-19]]
  - [x] Complete task A #2025-06-20
  - [ ] Incomplete task B

## Notes
Some notes here.`

		if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
			t.Fatalf("Failed to write source file: %v", err)
		}

		targetFile := filepath.Join(tempDir, "test_target.md")
		cmd := exec.Command(binaryPath, "process", sourceFile, targetFile)
		cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+testConfigDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Process command failed: %v", err)
		}

		if _, err := os.Stat(targetFile); os.IsNotExist(err) {
			t.Fatalf("Target file was not created")
		}

		targetContent, err := os.ReadFile(targetFile)
		if err != nil {
			t.Fatalf("Failed to read target file: %v", err)
		}

		if !strings.Contains(string(targetContent), "Incomplete task B") {
			t.Error("Target file should contain uncompleted task")
		}
		if strings.Contains(string(targetContent), "Complete task A") {
			t.Error("Target file should not contain completed task")
		}
	})
}
