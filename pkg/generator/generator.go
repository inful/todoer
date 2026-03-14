// Package generator provides a library interface for processing TODO journal files.
package generator

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/inful/todoer/pkg/core"
)

// Generator instances are safe for concurrent use by multiple goroutines as they
// only read from their internal state and do not modify it after construction.
type Generator struct {
	templateContent    string
	templateDate       string
	previousDate       string         // Date of previous journal (empty if none)
	customVars         map[string]any // Custom template variables
	frontmatterDateKey string         // Frontmatter date key
	todosHeader        string         // TODOS section header
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

	g := &Generator{
		templateContent:    templateContent,
		templateDate:       templateDate,
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

// ProcessResult holds readers for the modified original and new file.
type ProcessResult struct {
	ModifiedOriginal io.Reader
	NewFile          io.Reader
}

// Process handles the journal content, moving completed todos into the original
// and uncompleted todos into a freshly rendered template. Returns an error if
// parsing or processing fails.
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
		// Tolerate a missing TODOS section; treat it as empty.
		// Any other error (malformed content) is surfaced to the caller.
		if !strings.Contains(err.Error(), "could not find") {
			return nil, fmt.Errorf("failed to extract TODOS section: %w", err)
		}
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

// ProcessFile reads a journal file from disk and calls Process.
// Returns an error if the file cannot be read or processing fails.
func (g *Generator) ProcessFile(filename string) (*ProcessResult, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %w", filename, err)
	}

	return g.Process(string(content))
}

// createFromTemplateWithCustom renders the template using todos, dates, journal stats, and custom variables.
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

// validateTemplate validates the template syntax to catch errors early
func (g *Generator) validateTemplate() error {
	// Try parsing the template with the same functions used during execution
	_, err := template.New("validation").Funcs(core.CreateTemplateFunctions()).Parse(g.templateContent)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}
	return nil
}

// Option is a functional option for configuring a Generator.
type Option func(*options)

// options is the internal config struct populated by Option functions.
type options struct {
	previousDate       string
	customVars         map[string]any
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
func WithCustomVariables(vars map[string]any) Option {
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
// Fields not covered by the supplied opts retain the current generator's values.
func (g *Generator) WithOptions(opts ...Option) (*Generator, error) {
	// Seed config with all current values so callers only need to supply changes.
	config := &options{
		previousDate:       g.previousDate,
		customVars:         g.customVars,
		frontmatterDateKey: g.frontmatterDateKey,
		todosHeader:        g.todosHeader,
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
