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

// ExtractDateFromFrontmatter extracts the date from the frontmatter title.
// It looks for a date pattern in the frontmatter using FrontmatterDateRegex.
// If no date is found, it returns today's date as a fallback.
func ExtractDateFromFrontmatter(content string) (string, error) {
	if content == "" {
		return time.Now().Format(DateFormat), nil
	}

	// Look for the title in frontmatter
	matches := FrontmatterDateRegex.FindStringSubmatch(content)

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
// It returns three parts: content before TODOS, the TODOS section content, and content after TODOS.
// The function expects a specific format with a blank line after the TODOS header.
func ExtractTodosSection(content string) (string, string, string, error) {
	if content == "" {
		return "", "", "", fmt.Errorf("content cannot be empty")
	}

	// Find the TODOS section header
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
		// There is another section after TODOS
		todosEndIndex := beforeTodosEnd + nextSectionMatch[0]
		todosSection = content[beforeTodosEnd:todosEndIndex]
		afterTodos = content[todosEndIndex:]
	} else {
		// TODOS is the last section
		todosSection = afterHeaderContent
		afterTodos = ""
	}

	return beforeTodos, strings.TrimSpace(todosSection), afterTodos, nil
}

// ProcessTodosSection processes the TODOS section and returns the completed and uncompleted sections.
// It parses the todos, splits them into completed/uncompleted, adds date tags, and converts back to strings.
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

	// Parse the TODOS section into a structured format
	journal, err := ParseTodosSection(todosSection)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse todos section: %w", err)
	}

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

// CreateFromTemplateContent creates file content from template content using Go template syntax.
// It validates inputs, executes the template with the provided data, and cleans up formatting.
// The template receives TemplateData with Date and TODOS fields.
func CreateFromTemplateContent(templateContent, todosContent, currentDate string) (string, error) {
	// Validate inputs
	if err := validateTemplateInputs(templateContent, currentDate); err != nil {
		return "", err
	}

	// Create template data
	data := TemplateData{
		Date:  currentDate,
		TODOS: todosContent,
	}

	// Parse and execute the Go template
	output, err := executeTemplate(templateContent, data)
	if err != nil {
		return "", err
	}

	// Clean up extra blank lines when TODOS is empty
	if strings.TrimSpace(todosContent) == "" {
		output = cleanExcessiveBlankLines(output)
	}

	return output, nil
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
	tmpl, err := template.New("journal").Parse(templateContent)
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
