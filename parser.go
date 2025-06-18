// Package main provides parsing functionality for TODO items in markdown journal files.
//
// This parser handles hierarchical TODO structures with the following features:
// - Validates date formats in day headers
// - Supports nested TODO items with proper indentation
// - Handles mixed content (bullet points, continuations)
// - Normalizes tabs to spaces for consistent indentation
// - Provides comprehensive error reporting with context
// - Efficiently deep copies TODO items when splitting journals
//
// The parser uses a state machine approach to handle complex markdown structures
// while maintaining performance and readability.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// Constants for parsing
const (
	TodosHeader = "## TODOS"
	DateFormat  = "2006-01-02"
)

// validateDate validates that a date string is in the correct format
func validateDate(dateStr string) error {
	_, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return fmt.Errorf("invalid date format '%s', expected YYYY-MM-DD", dateStr)
	}
	return err
}

// parserState holds the state during parsing to reduce parameter passing
type parserState struct {
	currentDay         *DaySection
	currentIndentStack []int
	currentItemStack   []*TodoItem
}

// newParserState creates a new parser state
func newParserState() *parserState {
	return &parserState{
		currentDay:         nil,
		currentIndentStack: []int{},
		currentItemStack:   []*TodoItem{},
	}
}

// reset resets the parser state for a new day
func (ps *parserState) reset() {
	ps.currentIndentStack = []int{}
	ps.currentItemStack = []*TodoItem{}
}

// Compiled regex patterns for better performance
var (
	frontmatterDateRegex = regexp.MustCompile(`(?s)---.*?title:\s*(\d{4}-\d{2}-\d{2}).*?---`)
	nextSectionRegex     = regexp.MustCompile(`\n\n## `)
	dayHeaderRegex       = regexp.MustCompile(`- \[\[(\d{4}-\d{2}-\d{2})\]\]`)
	todoItemRegex        = regexp.MustCompile(`^(\s*)- \[([ x])\] (.+)$`)
	bulletEntryRegex     = regexp.MustCompile(`^(\s*)- (.+)$`)
	continuationRegex    = regexp.MustCompile(`^(\s+)(.+)$`)
	dateTagRegex         = regexp.MustCompile(`#\d{4}-\d{2}-\d{2}`)
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

// parseTodosSection parses the TODOS section into a structured format
func parseTodosSection(content string) (*TodoJournal, error) {
	journal := &TodoJournal{
		Days: []*DaySection{},
	}

	lines := strings.Split(content, "\n")
	state := newParserState()

	for lineNum, line := range lines {
		if err := processLine(journal, state, line, lineNum+1); err != nil {
			return nil, err
		}
	}

	// Add the last day if it exists
	if state.currentDay != nil {
		journal.Days = append(journal.Days, state.currentDay)
	}

	return journal, nil
}

// processLine processes a single line of the TODOS section
func processLine(journal *TodoJournal, state *parserState, line string, lineNum int) error {
	trimmedLine := strings.TrimSpace(line)
	if trimmedLine == "" {
		return nil
	}

	// Check for day header
	if dateMatch := dayHeaderRegex.FindStringSubmatch(trimmedLine); dateMatch != nil {
		return processDayHeader(journal, state, dateMatch[1])
	}

	// Skip processing if we don't have a current day
	if state.currentDay == nil {
		return nil
	}

	// Check for todo item first
	if todoMatch := todoItemRegex.FindStringSubmatch(line); todoMatch != nil {
		return processTodoItem(state, todoMatch)
	}

	// Check for bullet entry (- something that's not a todo)
	if bulletMatch := bulletEntryRegex.FindStringSubmatch(line); bulletMatch != nil {
		return processBulletEntry(state, line, bulletMatch)
	}

	// Check for continuation line (indented text that's part of a bullet or todo)
	if contMatch := continuationRegex.FindStringSubmatch(line); contMatch != nil {
		return processContinuationLine(state, line, contMatch)
	}

	return nil
}

// processDayHeader processes a day header line
func processDayHeader(journal *TodoJournal, state *parserState, dateStr string) error {
	// Validate the date format
	if err := validateDate(dateStr); err != nil {
		return fmt.Errorf("invalid date in day header: %w", err)
	}

	state.currentDay = createNewDaySection(journal, state.currentDay, dateStr)
	state.reset()
	return nil
}

// processTodoItem processes a todo item line
func processTodoItem(state *parserState, todoMatch []string) error {
	item := createTodoItem(todoMatch)
	indentLevel := getIndentLevel(todoMatch[1])
	state.currentIndentStack, state.currentItemStack = addItemToHierarchy(
		state.currentDay, item, indentLevel, state.currentIndentStack, state.currentItemStack)
	return nil
}

// processBulletEntry processes a bullet entry line
func processBulletEntry(state *parserState, line string, bulletMatch []string) error {
	if len(state.currentItemStack) > 0 {
		normalizedLine := normalizeIndentation(line)
		bulletIndent := getIndentLevel(bulletMatch[1])
		targetItem := findTargetItemForBullet(state.currentItemStack, state.currentIndentStack, bulletIndent)
		if targetItem != nil {
			targetItem.BulletLines = append(targetItem.BulletLines, normalizedLine)
		}
	}
	return nil
}

// processContinuationLine processes a continuation line
func processContinuationLine(state *parserState, line string, contMatch []string) error {
	if len(state.currentItemStack) > 0 {
		normalizedLine := normalizeIndentation(line)
		contIndent := getIndentLevel(contMatch[1])
		targetItem := findTargetItemForBullet(state.currentItemStack, state.currentIndentStack, contIndent)
		if targetItem != nil {
			targetItem.BulletLines = append(targetItem.BulletLines, normalizedLine)
		}
	}
	return nil
}

// findTargetItemForBullet finds the appropriate todo item to attach a bullet entry to
// based on indentation level. It traverses the current item stack from the most
// recent item backwards to find the first item whose indentation level is less
// than the bullet's indentation, indicating it should be the parent.
//
// Example:
//   - [ ] Todo item (indent 0)
//   - Bullet entry (indent 2) -> attaches to todo above
//   - [ ] Sub todo (indent 4)
//   - Sub bullet (indent 6) -> attaches to sub todo above
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

// getIndentLevel calculates the indentation level of a line (number of leading spaces/tabs)
// Treats tabs as 2 spaces for consistency
func getIndentLevel(line string) int {
	indent := 0
	for _, char := range line {
		if char == ' ' {
			indent++
		} else if char == '\t' {
			indent += 2 // Treat tab as 2 spaces
		} else {
			break
		}
	}
	return indent
}

// normalizeIndentation converts tabs to spaces for consistent indentation handling
func normalizeIndentation(line string) string {
	return strings.ReplaceAll(line, "\t", "  ")
}

// deepCopyItem creates a deep copy of a todo item
// Pre-allocates slices for better performance with large hierarchies
func deepCopyItem(item *TodoItem) *TodoItem {
	if item == nil {
		return nil
	}

	copy := &TodoItem{
		Completed:   item.Completed,
		Text:        item.Text,
		SubItems:    make([]*TodoItem, 0, len(item.SubItems)),
		BulletLines: make([]string, len(item.BulletLines)),
	}

	// Copy bullet lines
	copy.BulletLines = append(copy.BulletLines[:0], item.BulletLines...)

	// Copy subitems recursively
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
func tagCompletedSubtasks(journal *TodoJournal, originalDate string) {
	for _, day := range journal.Days {
		for _, item := range day.Items {
			tagCompletedSubitemsRecursive(item, originalDate)
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
func journalToString(journal *TodoJournal) string {
	if len(journal.Days) == 0 {
		return ""
	}

	var builder strings.Builder

	for _, day := range journal.Days {
		builder.WriteString("- [[")
		builder.WriteString(day.Date)
		builder.WriteString("]]\n")

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
		return "", "", "", fmt.Errorf("invalid %s section format: expected blank line after header at position %d",
			TodosHeader, headerEndIndex)
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
	tagCompletedItems(completedJournal, originalDate)

	// Add date tags to completed subtasks in uncompleted tasks
	tagCompletedSubtasks(uncompletedJournal, originalDate)

	// Convert back to string format
	completedTodos := journalToString(completedJournal)
	uncompletedTodos := journalToString(uncompletedJournal)

	// If there are no completed tasks, show "Moved to [[date]]"
	if len(completedJournal.Days) == 0 {
		completedTodos = "Moved to [[" + currentDate + "]]"
	}

	return completedTodos, uncompletedTodos, nil
}

// TemplateData holds the data to be passed to the template
type TemplateData struct {
	Date  string
	TODOS string
}

// createFromTemplate creates file content from a template using Go template syntax
func createFromTemplate(templateFile, todosContent, currentDate string) (string, error) {
	// Validate inputs
	if templateFile == "" {
		return "", fmt.Errorf("template file path cannot be empty")
	}

	if err := validateDate(currentDate); err != nil {
		return "", fmt.Errorf("invalid current date: %w", err)
	}

	// Read the template file
	templateBytes, err := os.ReadFile(templateFile)
	if err != nil {
		return "", fmt.Errorf("failed to read template file '%s': %w", templateFile, err)
	}

	templateContent := string(templateBytes)

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

	return result.String(), nil
}

// createFromTemplateContent creates file content from template content using Go template syntax
func createFromTemplateContent(templateContent, todosContent, currentDate string) (string, error) {
	// Validate inputs
	if templateContent == "" {
		return "", fmt.Errorf("template content cannot be empty")
	}

	if err := validateDate(currentDate); err != nil {
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
