// Package core provides shared file processing functionality for the todoer application.
package core

import (
	"fmt"
	"maps"
	"regexp"
	"strings"
	"text/template"
)

// Constants for file processing
const (
	// BlankLineSeparator is the sequence that separates sections
	BlankLineSeparator = "\n\n"
	// MovedToTemplate is the placeholder written into a source journal that
	// has nothing left completed. It is shaped like a completed todo so the
	// file stays parseable: a bare note line here would make every later run
	// on this file fail with "unparseable line".
	MovedToTemplate = "- [x] Moved to [[%s]]"
)

// Pre-compiled regex for better performance
var (
	excessiveBlankLinesRegex = regexp.MustCompile(`\n{3,}`)
	// dateShape matches a date value in a frontmatter line.
	dateShape = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// ExtractDateFromFrontmatter extracts the date from the frontmatter using a configurable key.
// Only the frontmatter block is searched, so a body line such as
// "created_date: 2019-01-01" is never mistaken for the file's date.
//
// If the key is not in the frontmatter, the other common key ("date", then
// "title") is tried before giving up: the built-in template writes both, but
// a hand-written journal usually has only one.
//
// If no date is found it returns an empty string: callers that need a value
// decide what to use. Falling back to today here made {{.PreviousDate}} and
// the "Moved to" marker silently claim a date the file never had.
func ExtractDateFromFrontmatter(content string, dateKey string) (string, error) {
	if content == "" {
		return "", nil
	}

	lines := frontmatterLines(content)
	for _, key := range dateKeyOrder(dateKey) {
		for _, line := range lines {
			k, value, ok := frontmatterValue(line)
			if !ok || k != key {
				continue
			}
			if !dateShape.MatchString(value) {
				continue
			}
			if err := ValidateDate(value); err != nil {
				return "", fmt.Errorf("invalid date in frontmatter: %w", err)
			}
			return value, nil
		}
	}

	return "", nil
}

// dateKeyOrder returns the keys to try: the configured one first, then the
// other common ones, without repeats.
func dateKeyOrder(dateKey string) []string {
	keys := []string{dateKey}
	for _, alt := range []string{"date", "title"} {
		if alt != dateKey {
			keys = append(keys, alt)
		}
	}
	return keys
}

// frontmatterLines returns the lines of the leading frontmatter block, or
// nil when the document has none.
func frontmatterLines(content string) []string {
	if content == "" {
		return nil
	}
	sep := "\n"
	if strings.Contains(content, "\r\n") {
		sep = "\r\n"
	}
	lines := strings.Split(content, sep)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return lines[1:i]
		}
	}
	return nil
}

// frontmatterValue splits a "key: value" line of a frontmatter block.
func frontmatterValue(line string) (string, string, bool) {
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

// lineSeparator returns the dominant line ending of content. Counting both
// endings (instead of testing for any CRLF) keeps a file with a stray
// Windows line ending working: it is written back the way it came in.
func lineSeparator(content string) string {
	crlf := strings.Count(content, "\r\n")
	lf := strings.Count(content, "\n") - crlf
	if crlf > lf {
		return "\r\n"
	}
	return "\n"
}

// MatchLineEndings rewrites content to use the same line ending as
// reference. Callers splice a section built from the model or from a
// template (both always use "\n") together with slices of a file that may
// be CRLF; without this the written file mixes both endings. Reference may
// be the file the content came from.
func MatchLineEndings(content, reference string) string {
	sep := lineSeparator(reference)
	if sep == "\n" {
		return content
	}
	const crlf = "\r\n"
	// Protect existing CRLFs, convert the bare LFs, then restore.
	guarded := strings.ReplaceAll(content, crlf, "\x00")
	guarded = strings.ReplaceAll(guarded, "\n", crlf)
	return strings.ReplaceAll(guarded, "\x00", crlf)
}

// todosHeaderPattern builds the pattern that locates the todos section
// header as a whole markdown heading. Anchoring at the line start and
// requiring the line to end after the header keeps a different section
// whose name merely starts with the same word (e.g. "## Todos Archive")
// or the words appearing inside a note from being treated as the todos
// section: rewriting the wrong section would move the user's items out
// of it.
func todosHeaderPattern(todosHeader string) string {
	return `(?m)^` + regexp.QuoteMeta(todosHeader) + `[ \t]*\r?$`
}

// ExtractTodosSectionWithHeader extracts the TODOS section using a configurable header.
// It returns content before the section, the section body, and content after.
// The function expects a specific format with a blank line after the Todos header.
func ExtractTodosSectionWithHeader(content string, todosHeader string) (string, string, string, error) {
	if content == "" {
		return "", "", "", fmt.Errorf("content cannot be empty")
	}
	if todosHeader == "" {
		return "", "", "", fmt.Errorf("todos header cannot be empty")
	}

	// Find the Todos section header as its own heading line
	headerRange := regexp.MustCompile(todosHeaderPattern(todosHeader)).FindStringIndex(content)
	if headerRange == nil {
		return "", "", "", fmt.Errorf("could not find '%s' section in file", todosHeader)
	}

	// Calculate the end of the header. The pattern allows a trailing \r
	// (so the anchor matches before the \n of a CRLF line); step back over
	// it so the section search starts at the same place an LF file would.
	headerEndIndex := headerRange[1]
	if headerEndIndex > headerRange[0] && content[headerEndIndex-1] == '\r' {
		headerEndIndex--
	}
	if headerEndIndex >= len(content) {
		return "", "", "", fmt.Errorf("incomplete %s section: no content after header", todosHeader)
	}

	contentAfterHeader := content[headerEndIndex:]

	// Find the first blank line after the header. Both endings are looked
	// for and the earlier one wins: a file with a stray CRLF among LF
	// lines must keep working, so the whole file's style must not decide
	// this.
	lfIndex := strings.Index(contentAfterHeader, "\n\n")
	crlfIndex := strings.Index(contentAfterHeader, "\r\n\r\n")
	var blankLine string
	var blankLineIndex int
	switch {
	case lfIndex >= 0 && (crlfIndex < 0 || lfIndex <= crlfIndex):
		blankLine, blankLineIndex = "\n\n", lfIndex
	case crlfIndex >= 0:
		blankLine, blankLineIndex = "\r\n\r\n", crlfIndex
	default:
		return "", "", "", fmt.Errorf("invalid %s section format: expected blank line after header", todosHeader)
	}

	// Calculate section boundaries
	beforeTodosEnd := headerEndIndex + blankLineIndex + len(blankLine)
	beforeTodos := content[:beforeTodosEnd]

	// Find the next section header (if any)
	afterHeaderContent := content[beforeTodosEnd:]

	// An empty todos section: the next heading follows the blank line
	// directly, so everything from here on is a trailing section. This
	// has to be checked before searching for the next header — otherwise
	// the search would find the heading *after* that one and report its
	// name as todos content.
	//
	// The blank line goes back in front of the trailing content: it is
	// what separates the todos section from the section after it, and
	// without it the last todo ends up glued to the next heading.
	if strings.HasPrefix(strings.TrimLeft(afterHeaderContent, " \t"), "## ") {
		return beforeTodos, "", blankLine + afterHeaderContent, nil
	}

	nextSectionMatch := NextSectionRegex.FindStringIndex(afterHeaderContent)

	var todosSection string
	var afterTodos string

	if nextSectionMatch != nil {
		// There is another section after Todos. nextSectionMatch[0]
		// points at the newline that ends the previous line; if the
		// character before that newline is also a newline, the
		// blank-line separator is part of the previous content, and
		// we include it in the todos section so the trailing blank
		// line is preserved on round-trip.
		todosEndIndex := beforeTodosEnd + nextSectionMatch[0]
		if todosEndIndex > 0 && content[todosEndIndex-1] == '\n' {
			todosEndIndex--
		}
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
		TotalTodos:     todoStats.TotalTodos,
		CompletedTodos: todoStats.CompletedTodos,
		// Kept for existing templates; same value as TotalTodos.
		UncompletedTodos:         todoStats.TotalTodos,
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
