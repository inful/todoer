// Package core provides shared file processing functionality for the todoer application.
package core

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// Constants for file processing
const (
	// BlankLineSeparator is the sequence that separates sections
	BlankLineSeparator = "\n\n"
	// MovedToTemplate is the template for moved todos message
	MovedToTemplate = "Moved to [[%s]]"
)

// Pre-compiled regex for better performance
var (
	excessiveBlankLinesRegex = regexp.MustCompile(`\n{3,}`)
)

// ExtractDateFromFrontmatter extracts the date from the frontmatter using a configurable key.
// If no date is found, it returns today's date as a fallback.
func ExtractDateFromFrontmatter(content string, dateKey string) (string, error) {
	if content == "" {
		return time.Now().Format(DateFormat), nil
	}

	// Use dynamic regex for the configured key
	regex := BuildFrontmatterDateRegex(dateKey)
	matches := regex.FindStringSubmatch(content)

	if len(matches) < 2 {
		// If no date found in frontmatter, use today's date
		return time.Now().Format(DateFormat), nil
	}

	// Validate the extracted date
	extractedDate := matches[1]
	if err := ValidateDate(extractedDate); err != nil {
		return "", fmt.Errorf("invalid date in frontmatter: %w", err)
	}

	return extractedDate, nil
}

// ExtractTodosSection extracts the TODOS section from the file content.
// It returns content before the section, the section body, and content after.
// The function expects a specific format with a blank line after the Todos header.
func ExtractTodosSection(content string) (string, string, string, error) {
	if content == "" {
		return "", "", "", fmt.Errorf("content cannot be empty")
	}

	// Find the Todos section header
	todosHeaderIndex := strings.Index(content, TodosHeader)
	if todosHeaderIndex == -1 {
		return "", "", "", fmt.Errorf("could not find '%s' section in file", TodosHeader)
	}

	// Calculate the end of the header
	headerEndIndex := todosHeaderIndex + len(TodosHeader)
	if headerEndIndex >= len(content) {
		return "", "", "", fmt.Errorf("incomplete %s section: no content after header", TodosHeader)
	}

	contentAfterHeader := content[headerEndIndex:]

	// Find the first blank line after the header
	blankLineIndex := strings.Index(contentAfterHeader, BlankLineSeparator)
	if blankLineIndex == -1 {
		return "", "", "", fmt.Errorf("invalid %s section format: expected blank line after header", TodosHeader)
	}

	// Calculate section boundaries
	beforeTodosEnd := headerEndIndex + blankLineIndex + len(BlankLineSeparator)
	beforeTodos := content[:beforeTodosEnd]

	// Find the next section header (if any)
	afterHeaderContent := content[beforeTodosEnd:]
	nextSectionMatch := NextSectionRegex.FindStringIndex(afterHeaderContent)

	var todosSection string
	var afterTodos string

	if nextSectionMatch != nil {
		// There is another section after Todos
		todosEndIndex := beforeTodosEnd + nextSectionMatch[0]
		todosSection = content[beforeTodosEnd:todosEndIndex]
		afterTodos = content[todosEndIndex:]
	} else {
		// Todos is the last section
		todosSection = afterHeaderContent
		afterTodos = ""
	}

	return beforeTodos, strings.TrimSpace(todosSection), afterTodos, nil
}

// ExtractTodosSectionWithHeader extracts the TODOS section using a configurable header.
// It returns content before the section, the section body, and content after.
// The function expects a specific format with a blank line after the Todos header.
func ExtractTodosSectionWithHeader(content string, todosHeader string) (string, string, string, error) {
	if content == "" {
		return "", "", "", fmt.Errorf("content cannot be empty")
	}

	// Find the Todos section header
	todosHeaderIndex := strings.Index(content, todosHeader)
	if todosHeaderIndex == -1 {
		return "", "", "", fmt.Errorf("could not find '%s' section in file", todosHeader)
	}

	// Calculate the end of the header
	headerEndIndex := todosHeaderIndex + len(todosHeader)
	if headerEndIndex >= len(content) {
		return "", "", "", fmt.Errorf("incomplete %s section: no content after header", todosHeader)
	}

	contentAfterHeader := content[headerEndIndex:]

	// Find the first blank line after the header
	blankLineIndex := strings.Index(contentAfterHeader, BlankLineSeparator)
	if blankLineIndex == -1 {
		return "", "", "", fmt.Errorf("invalid %s section format: expected blank line after header", todosHeader)
	}

	// Calculate section boundaries
	beforeTodosEnd := headerEndIndex + blankLineIndex + len(BlankLineSeparator)
	beforeTodos := content[:beforeTodosEnd]

	// Find the next section header (if any)
	afterHeaderContent := content[beforeTodosEnd:]
	nextSectionMatch := NextSectionRegex.FindStringIndex(afterHeaderContent)

	var todosSection string
	var afterTodos string

	if nextSectionMatch != nil {
		// There is another section after Todos
		todosEndIndex := beforeTodosEnd + nextSectionMatch[0]
		todosSection = content[beforeTodosEnd:todosEndIndex]
		afterTodos = content[todosEndIndex:]
	} else {
		// Todos is the last section
		todosSection = afterHeaderContent
		afterTodos = ""
	}

	return beforeTodos, strings.TrimSpace(todosSection), afterTodos, nil
}

// ProcessTodosSection processes the TODOS section and returns completed and uncompleted sections.
// It parses todos, splits them, adds date tags, and converts back to strings.
// If there are no completed tasks, it returns a "Moved to [[date]]" message for the completed section.
func ProcessTodosSection(todosSection string, originalDate string, currentDate string) (string, string, error) {
	// Validate inputs
	if err := validateProcessInputs(originalDate, currentDate); err != nil {
		return "", "", err
	}

	// Handle empty todos section
	if strings.TrimSpace(todosSection) == "" {
		return fmt.Sprintf(MovedToTemplate, currentDate), "", nil
	}

	// Parse the Todos section into a structured format
	journal, err := ParseTodosSection(todosSection)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse todos section: %w", err)
	}

	// Move undated todos to the original date (the date from the file frontmatter)
	journal = MoveUndatedTodosToCurrentDate(journal, originalDate)

	// Split the journal into completed and uncompleted tasks
	completedJournal, uncompletedJournal := SplitJournal(journal)

	// Add date tags to completed tasks
	TagCompletedItems(completedJournal, originalDate)

	// Add date tags to completed subtasks in uncompleted tasks
	TagCompletedSubitems(uncompletedJournal, originalDate)

	// Convert back to string format
	completedTodos := JournalToString(completedJournal)
	uncompletedTodos := JournalToString(uncompletedJournal)

	// If there are no completed tasks, show "Moved to [[date]]"
	if completedJournal.IsEmpty() {
		completedTodos = fmt.Sprintf(MovedToTemplate, currentDate)
	}

	return completedTodos, uncompletedTodos, nil
}

// validateProcessInputs validates the inputs for ProcessTodosSection
func validateProcessInputs(originalDate, currentDate string) error {
	if originalDate == "" {
		return fmt.Errorf("original date cannot be empty")
	}
	if currentDate == "" {
		return fmt.Errorf("current date cannot be empty")
	}
	if err := ValidateDate(originalDate); err != nil {
		return fmt.Errorf("invalid original date: %w", err)
	}
	if err := ValidateDate(currentDate); err != nil {
		return fmt.Errorf("invalid current date: %w", err)
	}
	return nil
}

// validateTemplateInputs validates inputs for CreateFromTemplateContent
func validateTemplateInputs(templateContent, currentDate string) error {
	if templateContent == "" {
		return fmt.Errorf("template content cannot be empty")
	}
	if err := ValidateDate(currentDate); err != nil {
		return fmt.Errorf("invalid current date: %w", err)
	}
	return nil
}

// executeTemplate parses and executes a Go template with the provided data
func executeTemplate(templateContent string, data TemplateData) (string, error) {
	tmpl, err := template.New("journal").Funcs(CreateTemplateFunctions()).Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return result.String(), nil
}

// cleanExcessiveBlankLines removes sequences of 3 or more newlines and replaces them with 2 newlines.
// This prevents excessive whitespace when template sections are empty.
func cleanExcessiveBlankLines(content string) string {
	return excessiveBlankLinesRegex.ReplaceAllString(content, BlankLineSeparator)
}

// TemplateOptions contains all options for template creation.
// This provides a flexible interface for template rendering with optional features.
type TemplateOptions struct {
	// Required fields
	Content      string // Template content to render
	TodosContent string // Todos content to insert
	CurrentDate  string // Current date in YYYY-MM-DD format

	// Optional fields
	PreviousDate string                 // Previous journal date (optional)
	Journal      *TodoJournal           // Journal for statistics calculation (optional)
	CustomVars   map[string]interface{} // Custom template variables (optional)
}

// CreateFromTemplate creates file content from template using the options pattern.
// This is the unified function that supports all template features: date formatting,
// todo statistics, and custom variables. Use TemplateOptions to specify what features to enable.
func CreateFromTemplate(opts TemplateOptions) (string, error) {
	// Validate inputs
	if err := validateTemplateInputs(opts.Content, opts.CurrentDate); err != nil {
		return "", err
	}

	// Validate custom variables if present
	if opts.CustomVars != nil {
		if err := ValidateCustomVariables(opts.CustomVars); err != nil {
			return "", fmt.Errorf("invalid custom variables: %w", err)
		}
	}

	// Format current date variables
	currentDateVars := FormatDateVariables(opts.CurrentDate)

	// Format previous date variables
	previousDateVars := FormatDateVariables(opts.PreviousDate)

	// Calculate todo statistics if journal provided
	var todoStats TodoStatistics
	if opts.Journal != nil {
		todoStats = CalculateTodoStatistics(opts.Journal, opts.CurrentDate)
	}

	// Create template data with all variants and statistics
	data := TemplateData{
		Date:         opts.CurrentDate,
		TODOS:        opts.TodosContent,
		PreviousDate: opts.PreviousDate,

		// Current date variants
		DateShort:  currentDateVars.Short,
		DateLong:   currentDateVars.Long,
		Year:       currentDateVars.Year,
		Month:      currentDateVars.Month,
		MonthName:  currentDateVars.MonthName,
		Day:        currentDateVars.Day,
		DayName:    currentDateVars.DayName,
		WeekNumber: currentDateVars.WeekNumber,

		// Previous date variants
		PreviousDateShort:  previousDateVars.Short,
		PreviousDateLong:   previousDateVars.Long,
		PreviousYear:       previousDateVars.Year,
		PreviousMonth:      previousDateVars.Month,
		PreviousMonthName:  previousDateVars.MonthName,
		PreviousDay:        previousDateVars.Day,
		PreviousDayName:    previousDateVars.DayName,
		PreviousWeekNumber: previousDateVars.WeekNumber,

		// Todo statistics (will be zero values if journal not provided)
		TotalTodos:               todoStats.TotalTodos,
		CompletedTodos:           todoStats.CompletedTodos,
		UncompletedTodos:         todoStats.UncompletedTodos,
		UncompletedTopLevelTodos: todoStats.UncompletedTopLevelTodos,
		TodoDates:                todoStats.TodoDates,
		OldestTodoDate:           todoStats.OldestTodoDate,
		TodoDaysSpan:             todoStats.TodoDaysSpan,
	}

	// Merge custom variables if provided
	if opts.CustomVars != nil {
		MergeCustomVariables(&data, opts.CustomVars)
	}

	// Parse and execute the Go template
	output, err := executeTemplate(opts.Content, data)
	if err != nil {
		return "", err
	}

	// Clean up extra blank lines when TODOS is empty
	if strings.TrimSpace(opts.TodosContent) == "" {
		output = cleanExcessiveBlankLines(output)
	}

	return output, nil
}

// CreateFromTemplateContentWithStats creates file content from template content using Go template syntax with todo statistics.
// Deprecated: Use CreateFromTemplate with TemplateOptions instead for better flexibility.
func CreateFromTemplateContentWithStats(templateContent, todosContent, currentDate, previousDate string, journal *TodoJournal) (string, error) {
	return CreateFromTemplate(TemplateOptions{
		Content:      templateContent,
		TodosContent: todosContent,
		CurrentDate:  currentDate,
		PreviousDate: previousDate,
		Journal:      journal,
	})
}

// ProcessTodosSectionWithStats processes the Todos section and returns completed/uncompleted sections plus parsed journal.
// Similar to ProcessTodosSection but also returns the original parsed journal for statistics calculation.
func ProcessTodosSectionWithStats(todosSection string, originalDate string, currentDate string) (string, string, *TodoJournal, error) {
	// Validate inputs
	if err := validateProcessInputs(originalDate, currentDate); err != nil {
		return "", "", nil, err
	}

	// Handle empty todos section
	if strings.TrimSpace(todosSection) == "" {
		return fmt.Sprintf(MovedToTemplate, currentDate), "", &TodoJournal{}, nil
	}

	// Parse the Todos section into a structured format
	journal, err := ParseTodosSection(todosSection)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to parse todos section: %w", err)
	}

	// Move undated todos to the original date (the date from the file frontmatter)
	journal = MoveUndatedTodosToCurrentDate(journal, originalDate)

	// Split the journal into completed and uncompleted tasks
	completedJournal, uncompletedJournal := SplitJournal(journal)

	// Add date tags to completed tasks
	TagCompletedItems(completedJournal, originalDate)

	// Add date tags to completed subtasks in uncompleted tasks
	TagCompletedSubitems(uncompletedJournal, originalDate)

	// Convert back to string format
	completedSection := JournalToString(completedJournal)
	uncompletedSection := JournalToString(uncompletedJournal)

	// If no completed tasks, provide moved message
	if strings.TrimSpace(completedSection) == "" {
		completedSection = fmt.Sprintf(MovedToTemplate, currentDate)
	}

	// Return original journal for statistics calculation
	return completedSection, uncompletedSection, journal, nil
}

// CreateFromTemplateContentWithCustom creates template output with comprehensive data including custom variables.
// Deprecated: Use CreateFromTemplate with TemplateOptions instead for better flexibility.
func CreateFromTemplateContentWithCustom(templateContent, todosContent, currentDate, previousDate string, journal *TodoJournal, customVars map[string]interface{}) (string, error) {
	return CreateFromTemplate(TemplateOptions{
		Content:      templateContent,
		TodosContent: todosContent,
		CurrentDate:  currentDate,
		PreviousDate: previousDate,
		Journal:      journal,
		CustomVars:   customVars,
	})
}

// CreateTemplateFunctions returns a map of custom template functions for enhanced template functionality.
// These functions provide date arithmetic, string manipulation, and utility operations for templates.
// The functions are organized into separate categories for maintainability.
func CreateTemplateFunctions() template.FuncMap {
	result := make(template.FuncMap)

	// Merge date functions
	for k, v := range createDateFunctions() {
		result[k] = v
	}

	// Merge string functions
	for k, v := range createStringFunctions() {
		result[k] = v
	}

	// Merge utility functions
	for k, v := range createUtilityFunctions() {
		result[k] = v
	}

	return result
}
