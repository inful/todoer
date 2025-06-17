package main

import (
	"fmt"
	"io"
	"log"
	"strings"
)

// Example demonstrates how to use the Generator library interface
func ExampleLibraryUsage() {
	// Define your journal content
	journalContent := `---
title: "2025-01-15"
---

# My Daily Journal

## TODOS

- [[2025-01-14]]
  - [ ] Buy groceries
  - [x] Finish project
  - [ ] Call dentist
    - [ ] Make appointment
    - [x] Verify insurance

## Notes

Had a productive day!`

	// Define your template
	templateContent := `# New TODO List - {{.Date}}

## Outstanding Tasks

{{.TODOS}}

## Generated

This file was generated on {{.Date}}.`

	// Create a generator with a specific template date
	generator, err := NewGenerator(templateContent, "2025-03-01")
	if err != nil {
		log.Fatalf("Failed to create generator: %v", err)
	}

	// Process the journal
	result, err := generator.Process(journalContent)
	if err != nil {
		log.Fatalf("Failed to process journal: %v", err)
	}

	// Read the modified original (with completed tasks marked)
	modifiedOriginalBytes, err := io.ReadAll(result.ModifiedOriginal)
	if err != nil {
		log.Fatalf("Failed to read modified original: %v", err)
	}

	// Read the new file (with uncompleted tasks only)
	newFileBytes, err := io.ReadAll(result.NewFile)
	if err != nil {
		log.Fatalf("Failed to read new file: %v", err)
	}

	fmt.Println("=== Modified Original ===")
	fmt.Println(string(modifiedOriginalBytes))
	fmt.Println("\n=== New TODO File ===")
	fmt.Println(string(newFileBytes))
}

// Example using generator from file
func ExampleFromFileUsage() {
	// Create a generator from a template file
	generator, err := NewGeneratorFromFile("/tmp/test_template_with_date.md", "2025-12-31")
	if err != nil {
		log.Fatalf("Failed to create generator from file: %v", err)
	}

	// Process a file directly
	result, err := generator.ProcessFile("/tmp/test_input.md")
	if err != nil {
		log.Fatalf("Failed to process file: %v", err)
	}

	// Use the results...
	newFileBytes, err := io.ReadAll(result.NewFile)
	if err != nil {
		log.Fatalf("Failed to read new file: %v", err)
	}

	fmt.Println("=== Generated from File ===")
	fmt.Println(string(newFileBytes))
}

// RunExamples runs the library usage examples
func RunExamples() {
	fmt.Println("Running library examples...")
	ExampleLibraryUsage()
	fmt.Println("\n" + strings.Repeat("=", 50))
	ExampleFromFileUsage()
}
