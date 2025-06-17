package main

import (
	"fmt"
	"os"
	"time"
)

// Constants for the application
const (
	FilePermissions = 0644
)

func main() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Println("Usage: todoer <source_file> <target_file> [template_file]")
		fmt.Println("  source_file:   Input journal file")
		fmt.Println("  target_file:   Output file for uncompleted tasks")
		fmt.Println("  template_file: Optional template for creating the target file")
		os.Exit(1)
	}

	sourceFile := os.Args[1]
	targetFile := os.Args[2]
	var templateFile string
	if len(os.Args) == 4 {
		templateFile = os.Args[3]
	}

	// Validate that source and target files are different
	if sourceFile == targetFile {
		fmt.Printf("Error: source and target files cannot be the same\n")
		os.Exit(1)
	}

	// Read the source file
	content, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", sourceFile, err)
		os.Exit(1)
	}

	// Extract the date from frontmatter
	date, err := extractDateFromFrontmatter(string(content))
	if err != nil {
		fmt.Printf("Error extracting date from %s: %v\n", sourceFile, err)
		os.Exit(1)
	}

	// Extract TODOS section
	beforeTodos, todosSection, afterTodos, err := extractTodosSection(string(content))
	if err != nil {
		fmt.Printf("Error extracting TODOS section from %s: %v\n", sourceFile, err)
		os.Exit(1)
	}

	// Get today's date for tagging
	currentDate := time.Now().Format(DateFormat)

	// Process the TODOS section
	completedTodos, uncompletedTodos, err := processTodosSection(todosSection, date, currentDate)
	if err != nil {
		fmt.Printf("Error processing TODOS section: %v\n", err)
		os.Exit(1)
	}

	// Create the completed file content
	completedFileContent := beforeTodos + completedTodos + afterTodos

	// Create the uncompleted file content (with template if provided)
	var uncompletedFileContent string
	if templateFile != "" {
		uncompletedFileContent, err = createFromTemplate(templateFile, uncompletedTodos, currentDate)
		if err != nil {
			fmt.Printf("Error creating file from template %s: %v\n", templateFile, err)
			os.Exit(1)
		}
	} else {
		uncompletedFileContent = beforeTodos + uncompletedTodos + afterTodos
	}

	// Write the outputs to files
	err = os.WriteFile(sourceFile, []byte(completedFileContent), FilePermissions)
	if err != nil {
		fmt.Printf("Error writing to source file %s: %v\n", sourceFile, err)
		os.Exit(1)
	}

	err = os.WriteFile(targetFile, []byte(uncompletedFileContent), FilePermissions)
	if err != nil {
		fmt.Printf("Error writing to target file %s: %v\n", targetFile, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully processed journal.\n")
	fmt.Printf("Completed tasks kept in: %s\n", sourceFile)
	fmt.Printf("Uncompleted tasks moved to: %s\n", targetFile)
	if templateFile != "" {
		fmt.Printf("Created from template: %s\n", templateFile)
	}
}
