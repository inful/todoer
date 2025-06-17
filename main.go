package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Constants for the application
const (
	TodosHeader     = "## TODOS"
	FilePermissions = 0644
	DateFormat      = "2006-01-02"
)

// Compiled regex patterns for better performance
var (
	frontmatterDateRegex = regexp.MustCompile(`(?s)---.*?title:\s*(\d{4}-\d{2}-\d{2}).*?---`)
	dayHeaderRegex       = regexp.MustCompile(`- \[\[(\d{4}-\d{2}-\d{2})\]\]`)
	todoItemRegex        = regexp.MustCompile(`^(\s*)- \[([ x])\] (.+)$`)
	bulletEntryRegex     = regexp.MustCompile(`^(\s*)- (.+)$`)
	continuationRegex    = regexp.MustCompile(`^(\s+)(.+)$`)
	dateTagRegex         = regexp.MustCompile(`#\d{4}-\d{2}-\d{2}`)
	nextSectionRegex     = regexp.MustCompile(`\n\n## `)
)

// TodoItem represents a todo item with its completion status and text
type TodoItem struct {
	Completed   bool
	Text        string
	SubItems    []*TodoItem
	BulletLines []string // Non-todo bullet entries and multiline content
}

// DaySection represents a day's todo items
type DaySection struct {
	Date  string
	Items []*TodoItem
}

// TodoJournal represents the entire journal
type TodoJournal struct {
	Days []*DaySection
}

func main() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Println("Usage: todoer <source_file> <target_file> [template_file]")
		fmt.Println("  source_file:   Input journal file")
		fmt.Println("  target_file:   Output file for uncompleted tasks")
		fmt.Println("  template_file: Optional template for creating the target file")
		os.Exit(1)
	}

	sourceFile := os.Args[1]
	targetFile := os.Args[2]
	var templateFile string
	if len(os.Args) == 4 {
		templateFile = os.Args[3]
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

	// Get today's date for tagging
	currentDate := time.Now().Format(DateFormat)

	// Process the TODOS section
	completedTodos, uncompletedTodos, err := processTodosSection(todosSection, date, currentDate)
	if err != nil {
		fmt.Printf("Error processing TODOS section: %v\n", err)
		os.Exit(1)
	}

	// Create the completed file content
	completedFileContent := beforeTodos + completedTodos + afterTodos

	// Create the uncompleted file content (with template if provided)
	var uncompletedFileContent string
	if templateFile != "" {
		uncompletedFileContent, err = createFromTemplate(templateFile, uncompletedTodos, currentDate)
		if err != nil {
			fmt.Printf("Error creating file from template %s: %v\n", templateFile, err)
			os.Exit(1)
		}
	} else {
		uncompletedFileContent = beforeTodos + uncompletedTodos + afterTodos
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
	if templateFile != "" {
		fmt.Printf("Created from template: %s\n", templateFile)
	}
}

// createFromTemplate creates file content from a template, inserting the TODOS section and replacing template variables
func createFromTemplate(templateFile, todosContent, currentDate string) (string, error) {
	// Read the template file
	templateBytes, err := os.ReadFile(templateFile)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}

	templateContent := string(templateBytes)

	// Replace template variables
	result := strings.ReplaceAll(templateContent, "{{date}}", currentDate)
	result = strings.ReplaceAll(result, "{{TODOS}}", todosContent)

	// If the template doesn't contain a TODOS placeholder, try to insert it in the TODOS section
	if !strings.Contains(templateContent, "{{TODOS}}") {
		// Find the TODOS section in the template
		todosHeaderIndex := strings.Index(result, TodosHeader)
		if todosHeaderIndex != -1 {
			// Get content before TODOS (including the header)
			headerEndIndex := todosHeaderIndex + len(TodosHeader)
			contentAfterHeader := result[headerEndIndex:]

			// Look for the next section or end of file
			nextSectionMatch := nextSectionRegex.FindStringIndex(contentAfterHeader)

			var beforeTodos, afterTodos string

			if nextSectionMatch != nil {
				// There is another section after TODOS
				beforeTodos = result[:headerEndIndex]
				afterTodos = contentAfterHeader[nextSectionMatch[0]:]
			} else {
				// TODOS is the last section
				beforeTodos = result[:headerEndIndex]
				afterTodos = ""
			}

			// Insert the TODOS content with proper spacing
			if todosContent != "" {
				result = beforeTodos + "\n\n" + todosContent + afterTodos
			} else {
				result = beforeTodos + afterTodos
			}
		}
	}

	return result, nil
}

// extractDateFromFrontmatter extracts the date from the frontmatter title
func extractDateFromFrontmatter(content string) (string, error) {
	// Look for the title in frontmatter
	matches := frontmatterDateRegex.FindStringSubmatch(content)

	if len(matches) < 2 {
		// If no date found in frontmatter, use today's date
		return time.Now().Format(DateFormat), nil
	}

	return matches[1], nil
}

// extractTodosSection extracts the TODOS section from the file content
func extractTodosSection(content string) (string, string, string, error) {
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
		return "", "", "", fmt.Errorf("invalid %s section format: expected blank line after header", TodosHeader)
	}

	beforeTodos := content[:headerEndIndex+firstBlankLineIndex+2] // Include the blank line

	// Find the next section header (if any)
	afterHeaderContent := content[headerEndIndex+firstBlankLineIndex+2:]
	nextSectionMatch := nextSectionRegex.FindStringIndex(afterHeaderContent)

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

// processTodosSection processes the TODOS section and returns the completed and uncompleted sections
func processTodosSection(todosSection string, originalDate string, currentDate string) (string, string, error) {
	// Parse the TODOS section into a structured format
	journal, err := parseTodosSection(todosSection)
	if err != nil {
		return "", "", err
	}

	// Split the journal into completed and uncompleted tasks
	completedJournal, uncompletedJournal := splitJournal(journal)

	// Add date tags to completed tasks
	tagCompletedItems(completedJournal, currentDate)

	// Add date tags to completed subtasks in uncompleted tasks
	tagCompletedSubtasks(uncompletedJournal)

	// Check if the input format has blank lines after day headers
	hasBlankLinesAfterHeaders := strings.Contains(todosSection, "- [[") && strings.Contains(todosSection, "]]\n\n")

	// Convert back to string format, with the appropriate format
	completedTodos := journalToString(completedJournal, hasBlankLinesAfterHeaders)
	uncompletedTodos := journalToString(uncompletedJournal, hasBlankLinesAfterHeaders)

	return completedTodos, uncompletedTodos, nil
}

// parseTodosSection parses the TODOS section into a structured format
func parseTodosSection(content string) (*TodoJournal, error) {
	journal := &TodoJournal{
		Days: []*DaySection{},
	}

	lines := strings.Split(content, "\n")
	var currentDay *DaySection
	var currentIndentStack []int
	var currentItemStack []*TodoItem

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		// Check for day header
		if dateMatch := dayHeaderRegex.FindStringSubmatch(trimmedLine); dateMatch != nil {
			currentDay = createNewDaySection(journal, currentDay, dateMatch[1])
			currentIndentStack = []int{}
			currentItemStack = []*TodoItem{}
			continue
		}

		// Skip processing if we don't have a current day
		if currentDay == nil {
			continue
		}

		// Check for todo item first
		if todoMatch := todoItemRegex.FindStringSubmatch(line); todoMatch != nil {
			item := createTodoItem(todoMatch)
			currentIndentStack, currentItemStack = addItemToHierarchy(
				currentDay, item, len(todoMatch[1]), currentIndentStack, currentItemStack)
			continue
		}

		// Check for bullet entry (- something that's not a todo)
		if bulletMatch := bulletEntryRegex.FindStringSubmatch(line); bulletMatch != nil {
			// This is a bullet entry, add it to the last todo item if we have one
			if len(currentItemStack) > 0 {
				bulletIndent := len(bulletMatch[1])

				// Find the appropriate parent based on indentation
				targetItem := findTargetItemForBullet(currentItemStack, currentIndentStack, bulletIndent)
				if targetItem != nil {
					// Store the bullet line with its original indentation relative to the todo item
					targetItem.BulletLines = append(targetItem.BulletLines, line)
				}
			}
			continue
		}

		// Check for continuation line (indented text that's part of a bullet or todo)
		if contMatch := continuationRegex.FindStringSubmatch(line); contMatch != nil {
			// This is a continuation line, add it to the last bullet entry or todo item
			if len(currentItemStack) > 0 {
				contIndent := len(contMatch[1])

				// Find the appropriate parent for this continuation line
				targetItem := findTargetItemForBullet(currentItemStack, currentIndentStack, contIndent)
				if targetItem != nil {
					targetItem.BulletLines = append(targetItem.BulletLines, line)
				}
			}
			continue
		}
	}

	// Add the last day if it exists
	if currentDay != nil {
		journal.Days = append(journal.Days, currentDay)
	}

	return journal, nil
}

// findTargetItemForBullet finds the appropriate todo item to attach a bullet entry to
func findTargetItemForBullet(currentItemStack []*TodoItem, currentIndentStack []int, bulletIndent int) *TodoItem {
	// Find the todo item that should contain this bullet entry based on indentation
	for i := len(currentIndentStack) - 1; i >= 0; i-- {
		if bulletIndent > currentIndentStack[i] {
			return currentItemStack[i]
		}
	}
	// If no suitable parent found, attach to the last item
	if len(currentItemStack) > 0 {
		return currentItemStack[len(currentItemStack)-1]
	}
	return nil
}

// createNewDaySection creates a new day section and adds the previous one to the journal
func createNewDaySection(journal *TodoJournal, currentDay *DaySection, date string) *DaySection {
	if currentDay != nil {
		journal.Days = append(journal.Days, currentDay)
	}
	return &DaySection{
		Date:  date,
		Items: []*TodoItem{},
	}
}

// createTodoItem creates a TodoItem from regex matches
func createTodoItem(matches []string) *TodoItem {
	return &TodoItem{
		Completed:   matches[2] == "x",
		Text:        matches[3],
		SubItems:    []*TodoItem{},
		BulletLines: []string{},
	}
}

// addItemToHierarchy adds a todo item to the correct position in the hierarchy
func addItemToHierarchy(currentDay *DaySection, item *TodoItem, indentLevel int,
	currentIndentStack []int, currentItemStack []*TodoItem) ([]int, []*TodoItem) {

	if len(currentIndentStack) == 0 {
		// This is a top-level item
		currentDay.Items = append(currentDay.Items, item)
		return append(currentIndentStack, indentLevel), append(currentItemStack, item)
	}

	lastIndentLevel := currentIndentStack[len(currentIndentStack)-1]

	if indentLevel > lastIndentLevel {
		// This is a child of the previous item
		parentItem := currentItemStack[len(currentItemStack)-1]
		parentItem.SubItems = append(parentItem.SubItems, item)
		return append(currentIndentStack, indentLevel), append(currentItemStack, item)
	}

	// Pop the stack until we find the right parent level
	for len(currentIndentStack) > 0 && indentLevel <= currentIndentStack[len(currentIndentStack)-1] {
		currentIndentStack = currentIndentStack[:len(currentIndentStack)-1]
		currentItemStack = currentItemStack[:len(currentItemStack)-1]
	}

	// Add this item
	if len(currentItemStack) == 0 {
		// This is a top-level item
		currentDay.Items = append(currentDay.Items, item)
	} else {
		// This is a child of some parent
		parentItem := currentItemStack[len(currentItemStack)-1]
		parentItem.SubItems = append(parentItem.SubItems, item)
	}

	return append(currentIndentStack, indentLevel), append(currentItemStack, item)
}

// splitJournal splits the journal into completed and uncompleted tasks
func splitJournal(journal *TodoJournal) (*TodoJournal, *TodoJournal) {
	completedJournal := &TodoJournal{
		Days: []*DaySection{},
	}

	uncompletedJournal := &TodoJournal{
		Days: []*DaySection{},
	}

	for _, day := range journal.Days {
		completedDay := &DaySection{
			Date:  day.Date,
			Items: []*TodoItem{},
		}

		uncompletedDay := &DaySection{
			Date:  day.Date,
			Items: []*TodoItem{},
		}

		hasCompletedItems := false
		hasUncompletedItems := false

		for _, item := range day.Items {
			if isCompleted(item) {
				hasCompletedItems = true
				// Create a deep copy of the item for the completed journal
				completedDay.Items = append(completedDay.Items, deepCopyItem(item))
			} else {
				hasUncompletedItems = true
				// Create a deep copy of the item for the uncompleted journal
				uncompletedDay.Items = append(uncompletedDay.Items, deepCopyItem(item))
			}
		}

		if hasCompletedItems {
			completedJournal.Days = append(completedJournal.Days, completedDay)
		}

		if hasUncompletedItems {
			uncompletedJournal.Days = append(uncompletedJournal.Days, uncompletedDay)
		}
	}

	return completedJournal, uncompletedJournal
}

// isCompleted checks if a todo item and all its subitems are completed
func isCompleted(item *TodoItem) bool {
	if !item.Completed {
		return false
	}

	// Check if all subitems are completed too
	for _, subItem := range item.SubItems {
		if !isCompleted(subItem) {
			return false
		}
	}

	return true
}

// deepCopyItem creates a deep copy of a todo item
func deepCopyItem(item *TodoItem) *TodoItem {
	copy := &TodoItem{
		Completed:   item.Completed,
		Text:        item.Text,
		SubItems:    []*TodoItem{},
		BulletLines: make([]string, len(item.BulletLines)),
	}

	// Copy bullet lines
	for i, line := range item.BulletLines {
		copy.BulletLines[i] = line
	}

	// Copy subitems
	for _, subItem := range item.SubItems {
		copy.SubItems = append(copy.SubItems, deepCopyItem(subItem))
	}

	return copy
}

// tagCompletedItems adds date tags to completed items
func tagCompletedItems(journal *TodoJournal, currentDate string) {
	for _, day := range journal.Days {
		for _, item := range day.Items {
			if item.Completed && !hasDateTag(item.Text) {
				item.Text += " #" + currentDate
			}
			// Also tag subitems
			tagCompletedSubitemsRecursive(item, currentDate)
		}
	}
}

// tagCompletedSubtasks adds date tags to completed subtasks in uncompleted parent tasks
func tagCompletedSubtasks(journal *TodoJournal) {
	for _, day := range journal.Days {
		for _, item := range day.Items {
			tagCompletedSubitemsRecursive(item, day.Date)
		}
	}
}

// tagCompletedSubitemsRecursive adds date tags to completed subitems recursively
func tagCompletedSubitemsRecursive(item *TodoItem, date string) {
	for _, subItem := range item.SubItems {
		if subItem.Completed && !hasDateTag(subItem.Text) {
			subItem.Text += " #" + date
		}
		tagCompletedSubitemsRecursive(subItem, date)
	}
}

// hasDateTag checks if text already has a date tag
func hasDateTag(text string) bool {
	return dateTagRegex.MatchString(text)
}

// journalToString converts a journal to string format
func journalToString(journal *TodoJournal, useBlankLinesAfterHeaders bool) string {
	if len(journal.Days) == 0 {
		return ""
	}

	var builder strings.Builder

	for _, day := range journal.Days {
		builder.WriteString("- [[")
		builder.WriteString(day.Date)
		builder.WriteString("]]")

		if useBlankLinesAfterHeaders {
			builder.WriteString("\n\n")
		} else {
			builder.WriteString("\n")
		}

		for _, item := range day.Items {
			writeItemToString(&builder, item, 1)
		}

		// No extra newlines between day sections in compact format
		// The writeItemToString already adds a newline after each item
	}

	return strings.TrimRight(builder.String(), "\n")
}

// writeItemToString writes a todo item to a string builder
func writeItemToString(builder *strings.Builder, item *TodoItem, depth int) {
	// Add indentation
	for i := 0; i < depth; i++ {
		builder.WriteString("  ")
	}

	// Write the item marker
	builder.WriteString("- [")
	if item.Completed {
		builder.WriteString("x")
	} else {
		builder.WriteString(" ")
	}
	builder.WriteString("] ")

	// Write the text
	builder.WriteString(item.Text)
	builder.WriteString("\n")

	// Write bullet lines (preserve original indentation)
	for _, bulletLine := range item.BulletLines {
		builder.WriteString(bulletLine)
		builder.WriteString("\n")
	}

	// Write subitems
	for _, subItem := range item.SubItems {
		writeItemToString(builder, subItem, depth+1)
	}
}
