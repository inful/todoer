package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// TodoItem represents a todo item with its completion status and text
type TodoItem struct {
	Indent    string
	Completed bool
	Text      string
	SubItems  []*TodoItem
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
	if len(os.Args) != 3 {
		fmt.Println("Usage: todoer <source_file> <target_file>")
		os.Exit(1)
	}

	sourceFile := os.Args[1]
	targetFile := os.Args[2]

	// Read the source file
	content, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Extract the date from frontmatter
	date, err := extractDateFromFrontmatter(string(content))
	if err != nil {
		fmt.Printf("Error extracting date: %v\n", err)
		os.Exit(1)
	}

	// Extract TODOS section
	beforeTodos, todosSection, afterTodos, err := extractTodosSection(string(content))
	if err != nil {
		fmt.Printf("Error extracting TODOS section: %v\n", err)
		os.Exit(1)
	}

	// Get today's date for tagging
	currentDate := time.Now().Format("2006-01-02")

	// Process the TODOS section
	completedTodos, uncompletedTodos, err := processTodosSection(todosSection, date, currentDate)
	if err != nil {
		fmt.Printf("Error processing TODOS section: %v\n", err)
		os.Exit(1)
	}

	// Create the completed and uncompleted files
	completedFileContent := beforeTodos + completedTodos + afterTodos
	uncompletedFileContent := beforeTodos + uncompletedTodos + afterTodos

	// Write the outputs to files
	err = os.WriteFile(sourceFile, []byte(completedFileContent), 0644)
	if err != nil {
		fmt.Printf("Error writing to source file: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(targetFile, []byte(uncompletedFileContent), 0644)
	if err != nil {
		fmt.Printf("Error writing to target file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully processed journal.\n")
	fmt.Printf("Completed tasks kept in: %s\n", sourceFile)
	fmt.Printf("Uncompleted tasks moved to: %s\n", targetFile)
}

// extractDateFromFrontmatter extracts the date from the frontmatter title
func extractDateFromFrontmatter(content string) (string, error) {
	// Look for the title in frontmatter
	re := regexp.MustCompile(`(?s)---.*?title:\s*(\d{4}-\d{2}-\d{2}).*?---`)
	matches := re.FindStringSubmatch(content)

	if len(matches) < 2 {
		// If no date found in frontmatter, use today's date
		return time.Now().Format("2006-01-02"), nil
	}

	return matches[1], nil
}

// extractTodosSection extracts the TODOS section from the file content
func extractTodosSection(content string) (string, string, string, error) {
	// Find the TODOS section header
	todosHeaderIndex := strings.Index(content, "## TODOS")
	if todosHeaderIndex == -1 {
		return "", "", "", fmt.Errorf("could not find TODOS section in file")
	}

	// Get content before TODOS
	beforeTodos := content[:todosHeaderIndex+8] // Include "## TODOS"

	// Find the first blank line after the header
	headerEndIndex := todosHeaderIndex + 8
	contentAfterHeader := content[headerEndIndex:]
	firstBlankLineIndex := strings.Index(contentAfterHeader, "\n\n")
	if firstBlankLineIndex == -1 {
		return "", "", "", fmt.Errorf("invalid TODOS section format")
	}

	beforeTodos = content[:headerEndIndex+firstBlankLineIndex+2] // Include the blank line

	// Find the next section header (if any)
	afterHeaderContent := content[headerEndIndex+firstBlankLineIndex+2:]
	nextSectionMatch := regexp.MustCompile(`\n\n## `).FindStringIndex(afterHeaderContent)

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

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			continue
		}

		// Check for day header - format: - [[YYYY-MM-DD]]
		dateMatch := regexp.MustCompile(`- \[\[(\d{4}-\d{2}-\d{2})\]\]`).FindStringSubmatch(trimmedLine)
		if dateMatch != nil {
			if currentDay != nil {
				journal.Days = append(journal.Days, currentDay)
			}
			currentDay = &DaySection{
				Date:  dateMatch[1],
				Items: []*TodoItem{},
			}
			currentIndentStack = []int{}
			currentItemStack = []*TodoItem{}
			continue
		}

		// Check for todo item - format: - [ ] text or - [x] text
		todoMatch := regexp.MustCompile(`^(\s*)- \[([ x])\] (.+)$`).FindStringSubmatch(line)
		if todoMatch != nil && currentDay != nil {
			indent := todoMatch[1]
			indentLevel := len(indent)
			completed := todoMatch[2] == "x"
			text := todoMatch[3]

			item := &TodoItem{
				Indent:    indent,
				Completed: completed,
				Text:      text,
				SubItems:  []*TodoItem{},
			}

			// Determine where to add this item in the hierarchy
			if len(currentIndentStack) == 0 {
				// This is a top-level item
				currentDay.Items = append(currentDay.Items, item)
				currentIndentStack = append(currentIndentStack, indentLevel)
				currentItemStack = append(currentItemStack, item)
			} else {
				// Find the parent item based on indentation
				lastIndentLevel := currentIndentStack[len(currentIndentStack)-1]

				if indentLevel > lastIndentLevel {
					// This is a child of the previous item
					parentItem := currentItemStack[len(currentItemStack)-1]
					parentItem.SubItems = append(parentItem.SubItems, item)
					currentIndentStack = append(currentIndentStack, indentLevel)
					currentItemStack = append(currentItemStack, item)
				} else {
					// Pop the stack until we find the right parent level
					for len(currentIndentStack) > 0 && indentLevel <= currentIndentStack[len(currentIndentStack)-1] {
						currentIndentStack = currentIndentStack[:len(currentIndentStack)-1]
						currentItemStack = currentItemStack[:len(currentItemStack)-1]
					}

					// Now add this item
					if len(currentItemStack) == 0 {
						// This is a top-level item
						currentDay.Items = append(currentDay.Items, item)
					} else {
						// This is a child of some parent
						parentItem := currentItemStack[len(currentItemStack)-1]
						parentItem.SubItems = append(parentItem.SubItems, item)
					}

					currentIndentStack = append(currentIndentStack, indentLevel)
					currentItemStack = append(currentItemStack, item)
				}
			}
		}
	}

	// Don't forget to add the last day
	if currentDay != nil {
		journal.Days = append(journal.Days, currentDay)
	}

	return journal, nil
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
		Indent:    item.Indent,
		Completed: item.Completed,
		Text:      item.Text,
		SubItems:  []*TodoItem{},
	}

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
			tagCompletedSubitemsWithDate(item, currentDate)
		}
	}
}

// tagCompletedSubitemsWithDate adds date tags to completed subitems
func tagCompletedSubitemsWithDate(item *TodoItem, date string) {
	for _, subItem := range item.SubItems {
		if subItem.Completed && !hasDateTag(subItem.Text) {
			subItem.Text += " #" + date
		}
		tagCompletedSubitemsWithDate(subItem, date)
	}
}

// tagCompletedSubtasks adds date tags to completed subtasks in uncompleted parent tasks
func tagCompletedSubtasks(journal *TodoJournal) {
	for _, day := range journal.Days {
		for _, item := range day.Items {
			tagCompletedSubitemsWithOriginalDate(item, day.Date)
		}
	}
}

// tagCompletedSubitemsWithOriginalDate adds date tags to completed subitems using the original date
func tagCompletedSubitemsWithOriginalDate(item *TodoItem, date string) {
	for _, subItem := range item.SubItems {
		if subItem.Completed && !hasDateTag(subItem.Text) {
			subItem.Text += " #" + date
		}
		tagCompletedSubitemsWithOriginalDate(subItem, date)
	}
}

// hasDateTag checks if text already has a date tag
func hasDateTag(text string) bool {
	return regexp.MustCompile(`#\d{4}-\d{2}-\d{2}`).MatchString(text)
}

// journalToString converts a journal to string format
func journalToString(journal *TodoJournal, useBlankLinesAfterHeaders bool) string {
	if len(journal.Days) == 0 {
		return ""
	}

	var builder strings.Builder

	for i, day := range journal.Days {
		if i > 0 {
			builder.WriteString("\n")
		}

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

	// Write subitems
	for _, subItem := range item.SubItems {
		writeItemToString(builder, subItem, depth+1)
	}
}
