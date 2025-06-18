// Package core provides shared file processing functionality for the todoer application.
package core

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// ExtractDateFromFrontmatter extracts the date from the frontmatter title
func ExtractDateFromFrontmatter(content string) (string, error) {
	// Look for the title in frontmatter
	matches := FrontmatterDateRegex.FindStringSubmatch(content)

	if len(matches) < 2 {
		// If no date found in frontmatter, use today's date
		return time.Now().Format(DateFormat), nil
	}

	return matches[1], nil
}

// ExtractTodosSection extracts the TODOS section from the file content
func ExtractTodosSection(content string) (string, string, string, error) {
	// Find the TODOS section header
	todosHeaderIndex := strings.Index(content, TodosHeader)
	if todosHeaderIndex == -1 {
		return "", "", "", fmt.Errorf("could not find '%s' section in file", TodosHeader)
	}

	// Get content before TODOS (including the header)
	headerEndIndex := todosHeaderIndex + len(TodosHeader)
	contentAfterHeader := content[headerEndIndex:]

	// Find the first blank line after the header
	firstBlankLineIndex := strings.Index(contentAfterHeader, "\n\n")
	if firstBlankLineIndex == -1 {
		return "", "", "", fmt.Errorf("invalid %s section format: expected blank line after header at position %d",
			TodosHeader, headerEndIndex)
	}

	beforeTodos := content[:headerEndIndex+firstBlankLineIndex+2] // Include the blank line

	// Find the next section header (if any)
	afterHeaderContent := content[headerEndIndex+firstBlankLineIndex+2:]
	nextSectionMatch := NextSectionRegex.FindStringIndex(afterHeaderContent)

	var todosSection string
	var afterTodos string

	if nextSectionMatch != nil {
		// There is another section after TODOS
		todosEndIndex := headerEndIndex + firstBlankLineIndex + 2 + nextSectionMatch[0]
		todosSection = content[headerEndIndex+firstBlankLineIndex+2 : todosEndIndex]
		afterTodos = content[todosEndIndex:]
	} else {
		// TODOS is the last section
		todosSection = afterHeaderContent
		afterTodos = ""
	}

	return beforeTodos, strings.TrimSpace(todosSection), afterTodos, nil
}

// ProcessTodosSection processes the TODOS section and returns the completed and uncompleted sections
func ProcessTodosSection(todosSection string, originalDate string, currentDate string) (string, string, error) {
	// Parse the TODOS section into a structured format
	journal, err := ParseTodosSection(todosSection)
	if err != nil {
		return "", "", err
	}

	// Split the journal into completed and uncompleted tasks
	completedJournal, uncompletedJournal := SplitJournal(journal)

	// Add date tags to completed tasks
	TagCompletedItems(completedJournal, originalDate)

	// Add date tags to completed subtasks in uncompleted tasks
	TagCompletedSubtasks(uncompletedJournal, originalDate)

	// Convert back to string format
	completedTodos := JournalToString(completedJournal)
	uncompletedTodos := JournalToString(uncompletedJournal)

	// If there are no completed tasks, show "Moved to [[date]]"
	if len(completedJournal.Days) == 0 {
		completedTodos = "Moved to [[" + currentDate + "]]"
	}

	return completedTodos, uncompletedTodos, nil
}

// CreateFromTemplateContent creates file content from template content using Go template syntax
func CreateFromTemplateContent(templateContent, todosContent, currentDate string) (string, error) {
	// Validate inputs
	if templateContent == "" {
		return "", fmt.Errorf("template content cannot be empty")
	}

	if err := ValidateDate(currentDate); err != nil {
		return "", fmt.Errorf("invalid current date: %w", err)
	}

	// Create template data
	data := TemplateData{
		Date:  currentDate,
		TODOS: todosContent,
	}

	// Parse and execute the Go template
	tmpl, err := template.New("journal").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Clean up extra blank lines when TODOS is empty
	output := result.String()
	if strings.TrimSpace(todosContent) == "" {
		// Replace any sequence of 3+ newlines with just 2 newlines (one blank line)
		re := regexp.MustCompile(`\n{3,}`)
		output = re.ReplaceAllString(output, "\n\n")
	}

	return output, nil
}
