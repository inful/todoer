// Package generator provides a library interface for processing TODO journal files.
package generator

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"
	"todoer/pkg/core"
)

// Generator represents a TODO journal generator that can process journal files
// and generate both modified original content and new uncompleted task files.
//
// Generator instances are safe for concurrent use by multiple goroutines as they
// only read from their internal state and do not modify it after construction.
type Generator struct {
	templateContent string
	templateDate    string
	currentDate     string
	previousDate    string                 // Date from the previous journal being processed (empty if none)
	customVars      map[string]interface{} // Custom template variables
}

// NewGenerator creates a new Generator instance with the specified template content and template date.
// Returns an error if the template date is invalid.
//
// Deprecated: Use NewGeneratorWithOptions for more flexibility and better error handling.
func NewGenerator(templateContent, templateDate string) (*Generator, error) {
	return NewGeneratorWithPrevious(templateContent, templateDate, "")
}

// NewGeneratorWithPrevious creates a new Generator instance with template content, template date, and previous journal date.
// Returns an error if the template date is invalid or template syntax is invalid.
func NewGeneratorWithPrevious(templateContent, templateDate, previousDate string) (*Generator, error) {
	// Validate the template date format
	if err := core.ValidateDate(templateDate); err != nil {
		return nil, fmt.Errorf("invalid template date: %w", err)
	}

	// Use current date for completion tagging
	currentDate := time.Now().Format(core.DateFormat)

	g := &Generator{
		templateContent: templateContent,
		templateDate:    templateDate,
		currentDate:     currentDate,
		previousDate:    previousDate,
	}

	// Validate template syntax
	if err := g.validateTemplate(); err != nil {
		return nil, err
	}

	return g, nil
}

// NewGeneratorWithCustom creates a new Generator instance with custom variables support.
// Returns an error if the template date is invalid or custom variables are invalid.
func NewGeneratorWithCustom(templateContent, templateDate string, customVars map[string]interface{}) (*Generator, error) {
	return NewGeneratorWithPreviousAndCustom(templateContent, templateDate, "", customVars)
}

// NewGeneratorWithPreviousAndCustom creates a new Generator instance with previous date and custom variables.
// Returns an error if the template date is invalid, custom variables are invalid, or template syntax is invalid.
func NewGeneratorWithPreviousAndCustom(templateContent, templateDate, previousDate string, customVars map[string]interface{}) (*Generator, error) {
	// Validate the template date format
	if err := core.ValidateDate(templateDate); err != nil {
		return nil, fmt.Errorf("invalid template date: %w", err)
	}

	// Validate custom variables
	if err := core.ValidateCustomVariables(customVars); err != nil {
		return nil, fmt.Errorf("invalid custom variables: %w", err)
	}

	// Use current date for completion tagging
	currentDate := time.Now().Format(core.DateFormat)

	g := &Generator{
		templateContent: templateContent,
		templateDate:    templateDate,
		currentDate:     currentDate,
		previousDate:    previousDate,
		customVars:      customVars,
	}

	// Validate template syntax
	if err := g.validateTemplate(); err != nil {
		return nil, err
	}

	return g, nil
}

// NewGeneratorFromFile creates a new Generator by reading the template from a file.
// Returns an error if the file cannot be read or the template date is invalid.
//
// Deprecated: Use NewGeneratorFromFileWithOptions for more flexibility and better error handling.
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

// NewGeneratorFromFileWithCustom creates a new Generator by reading template from file with custom variables.
// Returns an error if the file cannot be read, template date is invalid, or custom variables are invalid.
func NewGeneratorFromFileWithCustom(templateFile, templateDate string, customVars map[string]interface{}) (*Generator, error) {
	return NewGeneratorFromFileWithPreviousAndCustom(templateFile, templateDate, "", customVars)
}

// NewGeneratorFromFileWithPreviousAndCustom creates a new Generator by reading template from file with previous date and custom variables.
// Returns an error if the file cannot be read, template date is invalid, or custom variables are invalid.
func NewGeneratorFromFileWithPreviousAndCustom(templateFile, templateDate, previousDate string, customVars map[string]interface{}) (*Generator, error) {
	templateBytes, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file '%s': %w", templateFile, err)
	}

	return NewGeneratorWithPreviousAndCustom(string(templateBytes), templateDate, previousDate, customVars)
}

// NewGeneratorWithOptions creates a new Generator with flexible configuration options.
// This is the recommended constructor for new code as it provides the most flexibility.
// Returns an error if the template date is invalid, custom variables are invalid, or template syntax is invalid.
func NewGeneratorWithOptions(templateContent, templateDate string, opts ...GeneratorOption) (*Generator, error) {
	// Set up default configuration
	config := &generatorConfig{}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Validate the template date format
	if err := core.ValidateDate(templateDate); err != nil {
		return nil, fmt.Errorf("invalid template date: %w", err)
	}

	// Validate custom variables if provided
	if config.customVars != nil {
		if err := core.ValidateCustomVariables(config.customVars); err != nil {
			return nil, fmt.Errorf("invalid custom variables: %w", err)
		}
	}

	// Use current date for completion tagging
	currentDate := time.Now().Format(core.DateFormat)

	g := &Generator{
		templateContent: templateContent,
		templateDate:    templateDate,
		currentDate:     currentDate,
		previousDate:    config.previousDate,
		customVars:      config.customVars,
	}

	// Validate template syntax
	if err := g.validateTemplate(); err != nil {
		return nil, err
	}

	return g, nil
}

// NewGeneratorFromFileWithOptions creates a new Generator by reading template from file with flexible options.
// This is the recommended constructor for file-based templates.
// Returns an error if the file cannot be read, template date is invalid, or template syntax is invalid.
func NewGeneratorFromFileWithOptions(templateFile, templateDate string, opts ...GeneratorOption) (*Generator, error) {
	templateBytes, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file '%s': %w", templateFile, err)
	}

	return NewGeneratorWithOptions(string(templateBytes), templateDate, opts...)
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

	// Process the TODOS section with statistics
	completedTodos, uncompletedTodos, journal, err := core.ProcessTodosSectionWithStats(todosSection, date, g.templateDate)
	if err != nil {
		return nil, fmt.Errorf("failed to process TODOS section: %w", err)
	}

	// Create the completed file content
	completedFileContent := beforeTodos + completedTodos + afterTodos

	// Create the uncompleted file content using the template with statistics and custom variables
	uncompletedFileContent, err := g.createFromTemplateWithCustom(uncompletedTodos, g.templateDate, journal)
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

// createFromTemplateWithCustom creates file content from the generator's template with custom variables.
// This is the most comprehensive template creation method, supporting custom variables and statistics.
func (g *Generator) createFromTemplateWithCustom(todosContent string, dateToUse string, journal *core.TodoJournal) (string, error) {
	return core.CreateFromTemplateContentWithCustom(g.templateContent, todosContent, dateToUse, g.previousDate, journal, g.customVars)
}

// ExtractDateFromFrontmatter extracts the date from the frontmatter title of the given content.
// Returns the extracted date or an error if extraction fails.
// This function is provided for CLI compatibility and convenience.
func ExtractDateFromFrontmatter(content string) (string, error) {
	return core.ExtractDateFromFrontmatter(content)
}

// CreateFromTemplateContent creates file content from template content using Go template syntax.
// Returns the generated content or an error if template processing fails.
// This function is provided for testing compatibility and convenience.
func CreateFromTemplateContent(templateContent, todosContent, currentDate string) (string, error) {
	return core.CreateFromTemplateContent(templateContent, todosContent, currentDate, "")
}

// validateTemplate validates the template syntax to catch errors early
func (g *Generator) validateTemplate() error {
	// Try parsing the template with the same functions used during execution
	_, err := template.New("validation").Funcs(core.CreateTemplateFunctions()).Parse(g.templateContent)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}
	return nil
}

// GeneratorOption represents a configuration option for Generator creation
type GeneratorOption func(*generatorConfig)

// generatorConfig holds configuration for Generator creation
type generatorConfig struct {
	previousDate string
	customVars   map[string]interface{}
}

// WithPreviousDate sets the previous journal date for the generator
func WithPreviousDate(date string) GeneratorOption {
	return func(config *generatorConfig) {
		config.previousDate = date
	}
}

// WithCustomVariables sets custom template variables for the generator
func WithCustomVariables(vars map[string]interface{}) GeneratorOption {
	return func(config *generatorConfig) {
		config.customVars = vars
	}
}

// WithOptions creates a new Generator based on the current one but with modified options.
// This allows reconfiguring an existing generator without rebuilding from scratch.
func (g *Generator) WithOptions(opts ...GeneratorOption) (*Generator, error) {
	// Set up configuration with current values
	config := &generatorConfig{
		previousDate: g.previousDate,
		customVars:   g.customVars,
	}

	// Apply new options
	for _, opt := range opts {
		opt(config)
	}

	// Validate custom variables if provided
	if config.customVars != nil {
		if err := core.ValidateCustomVariables(config.customVars); err != nil {
			return nil, fmt.Errorf("invalid custom variables: %w", err)
		}
	}

	// Create new generator with updated configuration
	newGen := &Generator{
		templateContent: g.templateContent,
		templateDate:    g.templateDate,
		currentDate:     g.currentDate,
		previousDate:    config.previousDate,
		customVars:      config.customVars,
	}

	// Validate template syntax (should pass since original was valid, but safety first)
	if err := newGen.validateTemplate(); err != nil {
		return nil, err
	}

	return newGen, nil
}
