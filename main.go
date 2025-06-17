package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Constants for the application
const (
	FilePermissions = 0644
)

// Generator represents a TODO journal generator that can process journal files
// and generate both modified original content and new uncompleted task files
type Generator struct {
	templateContent string
	templateDate    string
	currentDate     string
}

// NewGenerator creates a new Generator instance with the specified template and dates
func NewGenerator(templateContent, templateDate string) (*Generator, error) {
	// Validate the template date format
	if err := validateDate(templateDate); err != nil {
		return nil, fmt.Errorf("invalid template date: %w", err)
	}

	// Use current date for completion tagging
	currentDate := time.Now().Format(DateFormat)

	return &Generator{
		templateContent: templateContent,
		templateDate:    templateDate,
		currentDate:     currentDate,
	}, nil
}

// NewGeneratorFromFile creates a new Generator by reading the template from a file
func NewGeneratorFromFile(templateFile, templateDate string) (*Generator, error) {
	templateBytes, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file '%s': %w", templateFile, err)
	}

	return NewGenerator(string(templateBytes), templateDate)
}

// ProcessResult holds the results of processing a journal
type ProcessResult struct {
	ModifiedOriginal io.Reader
	NewFile          io.Reader
}

// Process processes the original journal content and returns readers for both outputs
func (g *Generator) Process(originalContent string) (*ProcessResult, error) {
	// Extract the date from frontmatter
	date, err := extractDateFromFrontmatter(originalContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract date from frontmatter: %w", err)
	}

	// Extract TODOS section
	beforeTodos, todosSection, afterTodos, err := extractTodosSection(originalContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract TODOS section: %w", err)
	}

	// Process the TODOS section
	completedTodos, uncompletedTodos, err := processTodosSection(todosSection, date, g.currentDate)
	if err != nil {
		return nil, fmt.Errorf("failed to process TODOS section: %w", err)
	}

	// Create the completed file content
	completedFileContent := beforeTodos + completedTodos + afterTodos

	// Create the uncompleted file content using the template
	uncompletedFileContent, err := g.createFromTemplate(uncompletedTodos)
	if err != nil {
		return nil, fmt.Errorf("failed to create content from template: %w", err)
	}

	return &ProcessResult{
		ModifiedOriginal: strings.NewReader(completedFileContent),
		NewFile:          strings.NewReader(uncompletedFileContent),
	}, nil
}

// ProcessFile processes a journal file and returns readers for both outputs
func (g *Generator) ProcessFile(filename string) (*ProcessResult, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %w", filename, err)
	}

	return g.Process(string(content))
}

// createFromTemplate creates file content from the generator's template
func (g *Generator) createFromTemplate(todosContent string) (string, error) {
	return createFromTemplateContent(g.templateContent, todosContent, g.templateDate)
}

func main() {
	// Special case for running examples
	if len(os.Args) == 2 && os.Args[1] == "--examples" {
		RunExamples()
		return
	}

	if len(os.Args) < 4 || len(os.Args) > 5 {
		fmt.Println("Usage: todoer <source_file> <target_file> <template_file> [template_date]")
		fmt.Println("       todoer --examples  (run library usage examples)")
		fmt.Println("  source_file:    Input journal file")
		fmt.Println("  target_file:    Output file for uncompleted tasks")
		fmt.Println("  template_file:  Template for creating the target file")
		fmt.Println("  template_date:  Optional date for template rendering (YYYY-MM-DD)")
		fmt.Println("                  If not provided, current date will be used")
		os.Exit(1)
	}

	sourceFile := os.Args[1]
	targetFile := os.Args[2]
	templateFile := os.Args[3]

	var templateDate string
	if len(os.Args) == 5 {
		templateDate = os.Args[4]
		// Validate the template date format
		if err := validateDate(templateDate); err != nil {
			fmt.Printf("Error: invalid template date format '%s': %v\n", templateDate, err)
			os.Exit(1)
		}
	} else {
		// Use current date if not provided
		templateDate = time.Now().Format(DateFormat)
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

	// Get today's date for tagging completed items
	currentDate := time.Now().Format(DateFormat)

	// Process the TODOS section
	completedTodos, uncompletedTodos, err := processTodosSection(todosSection, date, currentDate)
	if err != nil {
		fmt.Printf("Error processing TODOS section: %v\n", err)
		os.Exit(1)
	}

	// Create the completed file content
	completedFileContent := beforeTodos + completedTodos + afterTodos

	// Create the uncompleted file content using the template with specified date
	uncompletedFileContent, err := createFromTemplate(templateFile, uncompletedTodos, templateDate)
	if err != nil {
		fmt.Printf("Error creating file from template %s: %v\n", templateFile, err)
		os.Exit(1)
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
	fmt.Printf("Created from template: %s (using date: %s)\n", templateFile, templateDate)
}
