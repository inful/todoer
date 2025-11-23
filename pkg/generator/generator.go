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

// Generator instances are safe for concurrent use by multiple goroutines as they
// only read from their internal state and do not modify it after construction.
type Generator struct {
	templateContent    string
	templateDate       string
	taggingDate        string
	previousDate       string                 // Date from the previous journal being processed (empty if none)
	customVars         map[string]interface{} // Custom template variables
	frontmatterDateKey string                 // Configurable frontmatter date key
	todosHeader        string                 // Configurable TODOS section header
}

// NewGeneratorWithOptions creates a new Generator with flexible configuration options.
// This is the recommended constructor for new code as it provides the most flexibility.
// Returns an error if the template date is invalid, custom variables are invalid, or template syntax is invalid.
func NewGeneratorWithOptions(templateContent, templateDate string, opts ...Option) (*Generator, error) {
	// Set up default configuration
	config := &options{
		todosHeader: core.TodosHeader, // Default to core.TodosHeader
	}

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
	taggingDate := time.Now().Format(core.DateFormat)

	g := &Generator{
		templateContent:    templateContent,
		templateDate:       templateDate,
		taggingDate:        taggingDate,
		previousDate:       config.previousDate,
		customVars:         config.customVars,
		frontmatterDateKey: config.frontmatterDateKey,
		todosHeader:        config.todosHeader, // Always set
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
func NewGeneratorFromFileWithOptions(templateFile, templateDate string, opts ...Option) (*Generator, error) {
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
	// Empty content is invalid; require at least some frontmatter/body
	if strings.TrimSpace(originalContent) == "" {
		return nil, fmt.Errorf("original content cannot be empty")
	}
	// Extract the date from frontmatter using the configured key
	date, err := core.ExtractDateFromFrontmatter(originalContent, g.frontmatterDateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to extract date from frontmatter: %w", err)
	}

	// Extract TODOS section
	beforeTodos, todosSection, afterTodos, err := core.ExtractTodosSectionWithHeader(originalContent, g.todosHeader)
	if err != nil {
		// If no TODOS section exists, treat it as having an empty section
		beforeTodos = originalContent
		todosSection = ""
		afterTodos = ""
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
	return core.CreateFromTemplate(core.TemplateOptions{
		Content:      g.templateContent,
		TodosContent: todosContent,
		CurrentDate:  dateToUse,
		PreviousDate: g.previousDate,
		Journal:      journal,
		CustomVars:   g.customVars,
	})
}

// ExtractDateFromFrontmatter extracts the date from the frontmatter title of the given content.
// Returns the extracted date or an error if extraction fails.
// This function is provided for CLI compatibility and convenience.
func ExtractDateFromFrontmatter(content string, dateKey string) (string, error) {
	return core.ExtractDateFromFrontmatter(content, dateKey)
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

// Option represents a configuration option for Generator creation
type Option func(*options)

// options holds configuration for Generator creation
type options struct {
	previousDate       string
	customVars         map[string]interface{}
	frontmatterDateKey string
	todosHeader        string
}

// WithPreviousDate sets the previous journal date for the generator
func WithPreviousDate(date string) Option {
	return func(config *options) {
		config.previousDate = date
	}
}

// WithCustomVariables sets custom template variables for the generator
func WithCustomVariables(vars map[string]interface{}) Option {
	return func(config *options) {
		config.customVars = vars
	}
}

// WithFrontmatterDateKey sets the frontmatter date key for the generator
func WithFrontmatterDateKey(key string) Option {
	return func(config *options) {
		config.frontmatterDateKey = key
	}
}

// WithTodosHeader sets the TODOS section header for the generator
func WithTodosHeader(header string) Option {
	return func(config *options) {
		config.todosHeader = header
	}
}

// WithOptions creates a new Generator based on the current one but with modified options.
// This allows reconfiguring an existing generator without rebuilding from scratch.
func (g *Generator) WithOptions(opts ...Option) (*Generator, error) {
	// Set up configuration with current values
	config := &options{
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
		templateContent:    g.templateContent,
		templateDate:       g.templateDate,
		taggingDate:        g.taggingDate,
		previousDate:       config.previousDate,
		customVars:         config.customVars,
		frontmatterDateKey: config.frontmatterDateKey,
		todosHeader:        config.todosHeader, // Always set
	}

	// Validate template syntax (should pass since original was valid, but safety first)
	if err := newGen.validateTemplate(); err != nil {
		return nil, err
	}

	return newGen, nil
}
