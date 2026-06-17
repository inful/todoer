// Package core provides shared file processing functionality for the todoer application.
package core

import (
	"fmt"
	"maps"
	"regexp"
	"sort"
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

// ExtractFrontmatterMetadata reads simple "key: value" pairs from the document frontmatter.
// It returns metadata, whether a frontmatter block exists, and an error for malformed frontmatter.
func ExtractFrontmatterMetadata(content string) (map[string]string, bool, error) {
	fmLines, _, _, hasFrontmatter, malformed := splitFrontmatter(content)
	if malformed {
		return map[string]string{}, true, fmt.Errorf("malformed frontmatter: missing closing delimiter")
	}
	if !hasFrontmatter {
		return map[string]string{}, false, nil
	}

	metadata := make(map[string]string)
	for _, line := range fmLines {
		key, value, ok := parseFrontmatterLine(line)
		if ok {
			metadata[key] = value
		}
	}

	return metadata, true, nil
}

// UpsertFrontmatterMetadata updates or inserts simple metadata in a document frontmatter block.
// If the document has malformed frontmatter, a new metadata block is prepended as a safe fallback.
// The document's original line ending style (LF or CRLF) is preserved.
func UpsertFrontmatterMetadata(content string, updates map[string]string) (string, error) {
	if len(updates) == 0 {
		return content, nil
	}

	fmLines, body, sep, hasFrontmatter, malformed := splitFrontmatter(content)
	if malformed || !hasFrontmatter {
		return prependFrontmatterBlock(content, updates, sep), nil
	}

	lineByKey := make(map[string]int)
	for i, line := range fmLines {
		key, _, ok := parseFrontmatterLine(line)
		if ok {
			lineByKey[key] = i
		}
	}

	keys := sortedMetadataKeys(updates)
	for _, key := range keys {
		line := key + ": " + updates[key]
		if idx, exists := lineByKey[key]; exists {
			fmLines[idx] = line
		} else {
			fmLines = append(fmLines, line)
		}
	}

	return joinFrontmatterAndBody(fmLines, body, sep), nil
}

// DeleteFrontmatterMetadata removes the listed keys from a document's
// frontmatter block. Other keys (and their order) and the body's line
// ending style are preserved. Missing keys are no-ops. A document
// without a frontmatter block, or with a malformed block, is returned
// unchanged. An empty keys list is a no-op.
func DeleteFrontmatterMetadata(content string, keys []string) (string, error) {
	if len(keys) == 0 {
		return content, nil
	}
	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}

	fmLines, body, sep, hasFM, malformed := splitFrontmatter(content)
	if malformed || !hasFM {
		return content, nil
	}

	filtered := fmLines[:0:0]
	for _, line := range fmLines {
		key, _, ok := parseFrontmatterLine(line)
		if ok && keySet[key] {
			continue
		}
		filtered = append(filtered, line)
	}

	return joinFrontmatterAndBody(filtered, body, sep), nil
}

// splitFrontmatter splits a document into its frontmatter lines and the
// body, preserving the line ending style. The returned separator is the
// one detected in the input (LF or CRLF); it is used by the re-emit
// helpers so a roundtrip on a CRLF file does not silently rewrite line
// endings.
func splitFrontmatter(content string) ([]string, string, string, bool, bool) {
	if content == "" {
		return nil, "", "\n", false, false
	}

	sep := "\n"
	if strings.Contains(content, "\r\n") {
		sep = "\r\n"
	}

	lines := strings.Split(content, sep)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, content, sep, false, false
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return lines[1:i], strings.Join(lines[i+1:], sep), sep, true, false
		}
	}

	return nil, content, sep, true, true
}

func parseFrontmatterLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}

	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", false
	}

	return key, value, true
}

func prependFrontmatterBlock(content string, updates map[string]string, sep string) string {
	keys := sortedMetadataKeys(updates)
	var builder strings.Builder
	builder.WriteString("---")
	builder.WriteString(sep)
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(updates[key])
		builder.WriteString(sep)
	}
	builder.WriteString("---")
	if content != "" {
		builder.WriteString(sep)
		builder.WriteString(content)
	}
	return builder.String()
}

func joinFrontmatterAndBody(fmLines []string, body string, sep string) string {
	var builder strings.Builder
	builder.WriteString("---")
	builder.WriteString(sep)
	for _, line := range fmLines {
		builder.WriteString(line)
		builder.WriteString(sep)
	}
	builder.WriteString("---")
	if body != "" {
		builder.WriteString(sep)
		builder.WriteString(body)
	}
	return builder.String()
}

func sortedMetadataKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
	PreviousDate string         // Previous journal date (optional)
	Journal      *TodoJournal   // Journal for statistics calculation (optional)
	CustomVars   map[string]any // Custom template variables (optional)
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

// CreateTemplateFunctions returns a map of custom template functions for enhanced template functionality.
// These functions provide date arithmetic, string manipulation, and utility operations for templates.
// The functions are organized into separate categories for maintainability.
func CreateTemplateFunctions() template.FuncMap {
	result := make(template.FuncMap)

	// Merge date functions
	maps.Copy(result, createDateFunctions())

	// Merge string functions
	maps.Copy(result, createStringFunctions())

	// Merge utility functions
	maps.Copy(result, createUtilityFunctions())

	return result
}
