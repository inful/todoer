// Package generator provides a library interface for processing TODO journal files.
package generator

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"todoer/pkg/core"
)

// Generator represents a TODO journal generator that can process journal files
// and generate both modified original content and new uncompleted task files
type Generator struct {
	templateContent string
	templateDate    string
	currentDate     string
	previousDate    string // Date from the previous journal being processed (empty if none)
}

// NewGenerator creates a new Generator instance with the specified template content and template date.
// Returns an error if the template date is invalid.
func NewGenerator(templateContent, templateDate string) (*Generator, error) {
	return NewGeneratorWithPrevious(templateContent, templateDate, "")
}

// NewGeneratorWithPrevious creates a new Generator instance with template content, template date, and previous journal date.
// Returns an error if the template date is invalid.
func NewGeneratorWithPrevious(templateContent, templateDate, previousDate string) (*Generator, error) {
	// Validate the template date format
	if err := core.ValidateDate(templateDate); err != nil {
		return nil, fmt.Errorf("invalid template date: %w", err)
	}

	// Use current date for completion tagging
	currentDate := time.Now().Format(core.DateFormat)

	return &Generator{
		templateContent: templateContent,
		templateDate:    templateDate,
		currentDate:     currentDate,
		previousDate:    previousDate,
	}, nil
}

// NewGeneratorFromFile creates a new Generator by reading the template from a file.
// Returns an error if the file cannot be read or the template date is invalid.
func NewGeneratorFromFile(templateFile, templateDate string) (*Generator, error) {
	return NewGeneratorFromFileWithPrevious(templateFile, templateDate, "")
}

// NewGeneratorFromFileWithPrevious creates a new Generator by reading the template from a file and including previous journal date.
// Returns an error if the file cannot be read or the template date is invalid.
func NewGeneratorFromFileWithPrevious(templateFile, templateDate, previousDate string) (*Generator, error) {
	templateBytes, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file '%s': %w", templateFile, err)
	}

	return NewGeneratorWithPrevious(string(templateBytes), templateDate, previousDate)
}

// ProcessResult holds the results of processing a journal
type ProcessResult struct {
	ModifiedOriginal io.Reader
	NewFile          io.Reader
}

// Process processes the original journal content and returns a ProcessResult containing readers for both the modified original and the new file.
// Returns an error if parsing or processing fails.
func (g *Generator) Process(originalContent string) (*ProcessResult, error) {
	// Extract the date from frontmatter
	date, err := core.ExtractDateFromFrontmatter(originalContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract date from frontmatter: %w", err)
	}

	// Extract TODOS section
	beforeTodos, todosSection, afterTodos, err := core.ExtractTodosSection(originalContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract TODOS section: %w", err)
	}

	// Process the TODOS section
	completedTodos, uncompletedTodos, err := core.ProcessTodosSection(todosSection, date, g.templateDate)
	if err != nil {
		return nil, fmt.Errorf("failed to process TODOS section: %w", err)
	}

	// Create the completed file content
	completedFileContent := beforeTodos + completedTodos + afterTodos

	// Create the uncompleted file content using the template
	uncompletedFileContent, err := g.createFromTemplateWithDate(uncompletedTodos, g.templateDate)
	if err != nil {
		return nil, fmt.Errorf("failed to create content from template: %w", err)
	}

	return &ProcessResult{
		ModifiedOriginal: strings.NewReader(completedFileContent),
		NewFile:          strings.NewReader(uncompletedFileContent),
	}, nil
}

// ProcessFile processes a journal file and returns a ProcessResult containing readers for both the modified original and the new file.
// Returns an error if the file cannot be read or processing fails.
func (g *Generator) ProcessFile(filename string) (*ProcessResult, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %w", filename, err)
	}

	return g.Process(string(content))
}

// createFromTemplateWithDate creates file content from the generator's template using a specific date
func (g *Generator) createFromTemplateWithDate(todosContent string, dateToUse string) (string, error) {
	return core.CreateFromTemplateContent(g.templateContent, todosContent, dateToUse, g.previousDate)
}

// ExtractDateFromFrontmatter extracts the date from the frontmatter title of the given content.
// Returns the extracted date or an error if extraction fails.
func ExtractDateFromFrontmatter(content string) (string, error) {
	return core.ExtractDateFromFrontmatter(content)
}

// ExtractTodosSection extracts the TODOS section from the file content.
// Returns the content before the section, the section itself, the content after, and an error if extraction fails.
func ExtractTodosSection(content string) (string, string, string, error) {
	return core.ExtractTodosSection(content)
}

// CreateFromTemplateContent creates file content from template content using Go template syntax.
// Returns the generated content or an error if template processing fails.
func CreateFromTemplateContent(templateContent, todosContent, currentDate string) (string, error) {
	return core.CreateFromTemplateContent(templateContent, todosContent, currentDate, "")
}

// IsCompleted checks if a todo item is completed.
// Returns true if the item is completed, false otherwise.
func IsCompleted(item *core.TodoItem) bool {
	return core.IsCompleted(item)
}
