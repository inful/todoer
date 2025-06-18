// Package generator provides a library interface for processing TODO journal files.
package generator

import (
	"fmt"
	"io"
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

// Compiled regex patterns for better performance
var (
	frontmatterDateRegex = regexp.MustCompile(`(?s)---.*?title:\s*(\d{4}-\d{2}-\d{2}).*?---`)
	nextSectionRegex     = regexp.MustCompile(`\n\n## `)
	dayHeaderRegex       = regexp.MustCompile(`- \[\[(\d{4}-\d{2}-\d{2})\]\]`)
	todoItemRegex        = regexp.MustCompile(`^(\s*)- \[([ x])\] (.+)$`)
	bulletEntryRegex     = regexp.MustCompile(`^(\s*)- (.+)$`)
	continuationRegex    = regexp.MustCompile(`^(\s+)(.+)$`)
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

// TemplateData holds the data to be passed to the template
type TemplateData struct {
	Date  string
	TODOS string
}

// validateDate validates that a date string is in the correct format
func validateDate(dateStr string) error {
	_, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return fmt.Errorf("date '%s' must be in format %s", dateStr, DateFormat)
	}
	return nil
}

// Generator represents a TODO journal generator that can process journal files
// and generate both modified original content and new uncompleted task files
type Generator struct {
	templateContent string
	templateDate    string
	currentDate     string
}

// NewGenerator creates a new Generator instance with the specified template and dates
func NewGenerator(templateContent, templateDate string) (*Generator, error) {
	// Validate the template date format
	if err := validateDate(templateDate); err != nil {
		return nil, fmt.Errorf("invalid template date: %w", err)
	}

	// Use current date for completion tagging
	currentDate := time.Now().Format(DateFormat)

	return &Generator{
		templateContent: templateContent,
		templateDate:    templateDate,
		currentDate:     currentDate,
	}, nil
}

// NewGeneratorFromFile creates a new Generator by reading the template from a file
func NewGeneratorFromFile(templateFile, templateDate string) (*Generator, error) {
	templateBytes, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file '%s': %w", templateFile, err)
	}

	return NewGenerator(string(templateBytes), templateDate)
}

// ProcessResult holds the results of processing a journal
type ProcessResult struct {
	ModifiedOriginal io.Reader
	NewFile          io.Reader
}

// Process processes the original journal content and returns readers for both outputs
func (g *Generator) Process(originalContent string) (*ProcessResult, error) {
	// Extract the date from frontmatter
	date, err := extractDateFromFrontmatter(originalContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract date from frontmatter: %w", err)
	}

	// Extract TODOS section
	beforeTodos, todosSection, afterTodos, err := extractTodosSection(originalContent)
	if err != nil {
		return nil, fmt.Errorf("failed to extract TODOS section: %w", err)
	}

	// Process the TODOS section
	completedTodos, uncompletedTodos, err := processTodosSection(todosSection, date, g.templateDate)
	if err != nil {
		return nil, fmt.Errorf("failed to process TODOS section: %w", err)
	}

	// Create the completed file content
	completedFileContent := beforeTodos + completedTodos + afterTodos

	// Create the uncompleted file content using the template
	uncompletedFileContent, err := g.createFromTemplateWithDate(uncompletedTodos, date)
	if err != nil {
		return nil, fmt.Errorf("failed to create content from template: %w", err)
	}

	return &ProcessResult{
		ModifiedOriginal: strings.NewReader(completedFileContent),
		NewFile:          strings.NewReader(uncompletedFileContent),
	}, nil
}

// ProcessFile processes a journal file and returns readers for both outputs
func (g *Generator) ProcessFile(filename string) (*ProcessResult, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %w", filename, err)
	}

	return g.Process(string(content))
}

// createFromTemplateWithDate creates file content from the generator's template using a specific date
func (g *Generator) createFromTemplateWithDate(todosContent string, dateToUse string) (string, error) {
	return createFromTemplateContent(g.templateContent, todosContent, dateToUse)
}

// ExtractDateFromFrontmatter extracts the date from the frontmatter title
func ExtractDateFromFrontmatter(content string) (string, error) {
	return extractDateFromFrontmatter(content)
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

// ExtractTodosSection extracts the TODOS section from the file content
func ExtractTodosSection(content string) (string, string, string, error) {
	return extractTodosSection(content)
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

// CreateFromTemplateContent creates file content from template content using Go template syntax
func CreateFromTemplateContent(templateContent, todosContent, currentDate string) (string, error) {
	return createFromTemplateContent(templateContent, todosContent, currentDate)
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

	// If TODOS is empty, set it to empty string to avoid extra blank lines
	if strings.TrimSpace(todosContent) == "" {
		data.TODOS = ""
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

// parserState represents the current state while parsing a TODOS section
type parserState struct {
	currentDay         *DaySection
	currentItemStack   []*TodoItem // Stack for tracking nested items
	currentIndentStack []int       // Stack for tracking indentation levels
}

// newParserState creates a new parser state
func newParserState() *parserState {
	return &parserState{
		currentItemStack:   make([]*TodoItem, 0),
		currentIndentStack: make([]int, 0),
	}
}

// reset resets the parser state for a new day
func (ps *parserState) reset() {
	ps.currentItemStack = ps.currentItemStack[:0]
	ps.currentIndentStack = ps.currentIndentStack[:0]
}

// parseTodosSection parses the TODOS section and returns a TodoJournal
func parseTodosSection(content string) (*TodoJournal, error) {
	lines := strings.Split(content, "\n")
	journal := &TodoJournal{Days: make([]*DaySection, 0)}
	state := newParserState()

	for lineNum, line := range lines {
		if err := processLine(journal, state, line, lineNum+1); err != nil {
			return nil, err
		}
	}

	return journal, nil
}

// processLine processes a single line in the parser state machine
func processLine(journal *TodoJournal, state *parserState, line string, lineNum int) error {
	// Check for day header first
	if dayMatches := dayHeaderRegex.FindStringSubmatch(line); dayMatches != nil {
		return processDayHeader(journal, state, dayMatches[1])
	}

	// Skip empty lines
	if strings.TrimSpace(line) == "" {
		return nil
	}

	// Check for todo item
	if todoMatches := todoItemRegex.FindStringSubmatch(line); todoMatches != nil {
		return processTodoItem(state, todoMatches)
	}

	// Check for bullet entry
	if bulletMatches := bulletEntryRegex.FindStringSubmatch(line); bulletMatches != nil {
		return processBulletEntry(state, line, bulletMatches)
	}

	// Check for continuation line
	if contMatches := continuationRegex.FindStringSubmatch(line); contMatches != nil {
		return processContinuationLine(state, line, contMatches)
	}

	return nil
}

// processDayHeader processes a day header line
func processDayHeader(journal *TodoJournal, state *parserState, dateStr string) error {
	if err := validateDate(dateStr); err != nil {
		return fmt.Errorf("invalid date in day header '%s': %w", dateStr, err)
	}

	state.currentDay = createNewDaySection(journal, state.currentDay, dateStr)
	state.reset()
	return nil
}

// processTodoItem processes a todo item line
func processTodoItem(state *parserState, todoMatch []string) error {
	item := createTodoItem(todoMatch)
	indentLevel := getIndentLevel(todoMatch[1])

	state.currentItemStack, state.currentIndentStack = addItemToHierarchy(
		state.currentDay, item, indentLevel, state.currentItemStack, state.currentIndentStack)

	return nil
}

// processBulletEntry processes a bullet entry line
func processBulletEntry(state *parserState, line string, bulletMatch []string) error {
	bulletIndent := getIndentLevel(bulletMatch[1])
	targetItem := findTargetItemForBullet(state.currentItemStack, state.currentIndentStack, bulletIndent)

	if targetItem != nil {
		targetItem.BulletLines = append(targetItem.BulletLines, normalizeIndentation(line))
	}

	return nil
}

// processContinuationLine processes a continuation line
func processContinuationLine(state *parserState, line string, contMatch []string) error {
	if len(state.currentItemStack) > 0 {
		lastItem := state.currentItemStack[len(state.currentItemStack)-1]
		if len(lastItem.BulletLines) > 0 {
			lastIdx := len(lastItem.BulletLines) - 1
			lastItem.BulletLines[lastIdx] += "\n" + normalizeIndentation(line)
		} else {
			lastItem.BulletLines = append(lastItem.BulletLines, normalizeIndentation(line))
		}
	}
	return nil
}

// findTargetItemForBullet finds the appropriate todo item for a bullet entry
func findTargetItemForBullet(currentItemStack []*TodoItem, currentIndentStack []int, bulletIndent int) *TodoItem {
	for i := len(currentIndentStack) - 1; i >= 0; i-- {
		if bulletIndent > currentIndentStack[i] {
			return currentItemStack[i]
		}
	}
	return nil
}

// createNewDaySection creates and adds a new day section to the journal
func createNewDaySection(journal *TodoJournal, currentDay *DaySection, date string) *DaySection {
	newDay := &DaySection{
		Date:  date,
		Items: make([]*TodoItem, 0),
	}
	journal.Days = append(journal.Days, newDay)
	return newDay
}

// createTodoItem creates a TodoItem from regex matches
func createTodoItem(matches []string) *TodoItem {
	return &TodoItem{
		Completed:   matches[2] == "x",
		Text:        matches[3],
		SubItems:    make([]*TodoItem, 0),
		BulletLines: make([]string, 0),
	}
}

// addItemToHierarchy adds an item to the current day's hierarchy and updates stacks
func addItemToHierarchy(currentDay *DaySection, item *TodoItem, indentLevel int,
	currentItemStack []*TodoItem, currentIndentStack []int) ([]*TodoItem, []int) {

	// Determine where to add this item based on indentation
	if indentLevel == 0 {
		// Top-level item
		currentDay.Items = append(currentDay.Items, item)
		return []*TodoItem{item}, []int{0}
	}

	// Find the correct parent for this nested item
	var targetParent *TodoItem
	newStack := make([]*TodoItem, 0)
	newIndentStack := make([]int, 0)

	for i := len(currentIndentStack) - 1; i >= 0; i-- {
		if indentLevel > currentIndentStack[i] {
			targetParent = currentItemStack[i]
			// Keep the stack up to this parent
			newStack = append(newStack, currentItemStack[:i+1]...)
			newIndentStack = append(newIndentStack, currentIndentStack[:i+1]...)
			break
		}
	}

	if targetParent == nil && len(currentDay.Items) > 0 {
		// If no suitable parent found but we have items, use the last top-level item
		for i := len(currentDay.Items) - 1; i >= 0; i-- {
			if currentDay.Items[i] != nil {
				targetParent = currentDay.Items[i]
				newStack = []*TodoItem{targetParent}
				newIndentStack = []int{0}
				break
			}
		}
	}

	// Add the item to the target parent
	if targetParent != nil {
		targetParent.SubItems = append(targetParent.SubItems, item)
	} else {
		// No valid parent found, add as top-level item
		currentDay.Items = append(currentDay.Items, item)
		return []*TodoItem{item}, []int{0}
	}

	// Update the stacks
	newStack = append(newStack, item)
	newIndentStack = append(newIndentStack, indentLevel)

	return newStack, newIndentStack
}

// splitJournal splits a journal into completed and uncompleted parts
func splitJournal(journal *TodoJournal) (*TodoJournal, *TodoJournal) {
	completedJournal := &TodoJournal{Days: make([]*DaySection, 0)}
	uncompletedJournal := &TodoJournal{Days: make([]*DaySection, 0)}

	for _, day := range journal.Days {
		completedDay := &DaySection{Date: day.Date, Items: make([]*TodoItem, 0)}
		uncompletedDay := &DaySection{Date: day.Date, Items: make([]*TodoItem, 0)}

		for _, item := range day.Items {
			completedItem, uncompletedItem := splitTodoItem(item)
			if completedItem != nil {
				completedDay.Items = append(completedDay.Items, completedItem)
			}
			if uncompletedItem != nil {
				uncompletedDay.Items = append(uncompletedDay.Items, uncompletedItem)
			}
		}

		if len(completedDay.Items) > 0 {
			completedJournal.Days = append(completedJournal.Days, completedDay)
		}
		if len(uncompletedDay.Items) > 0 {
			uncompletedJournal.Days = append(uncompletedJournal.Days, uncompletedDay)
		}
	}

	return completedJournal, uncompletedJournal
}

// splitTodoItem splits a todo item into completed and uncompleted parts
func splitTodoItem(item *TodoItem) (*TodoItem, *TodoItem) {
	var completedItem, uncompletedItem *TodoItem

	// Process completed tasks
	if isCompleted(item) {
		completedItem = deepCopyItem(item)
	}

	// Process uncompleted tasks or tasks with uncompleted subtasks
	hasUncompletedSubtasks := false
	var uncompletedSubItems []*TodoItem
	var completedSubItems []*TodoItem

	for _, subItem := range item.SubItems {
		completedSub, uncompletedSub := splitTodoItem(subItem)
		if completedSub != nil {
			completedSubItems = append(completedSubItems, completedSub)
		}
		if uncompletedSub != nil {
			uncompletedSubItems = append(uncompletedSubItems, uncompletedSub)
			hasUncompletedSubtasks = true
		}
	}

	// Create uncompleted item if task is not completed or has uncompleted subtasks
	if !item.Completed || hasUncompletedSubtasks {
		uncompletedItem = deepCopyItem(item)
		uncompletedItem.SubItems = uncompletedSubItems
	}

	// Update completed item's subitems if it exists
	if completedItem != nil && len(completedSubItems) > 0 {
		completedItem.SubItems = completedSubItems
	}

	return completedItem, uncompletedItem
}

// IsCompleted checks if a todo item is completed
func IsCompleted(item *TodoItem) bool {
	return isCompleted(item)
}

// isCompleted checks if a todo item is completed
func isCompleted(item *TodoItem) bool {
	return item.Completed
}

// getIndentLevel calculates the indentation level from whitespace
func getIndentLevel(indentStr string) int {
	indentCount := 0
	for _, char := range indentStr {
		if char == '\t' {
			indentCount += 4 // Convert tabs to 4 spaces equivalent
		} else if char == ' ' {
			indentCount++
		}
	}
	return indentCount
}

// normalizeIndentation normalizes tabs to spaces for consistent formatting
func normalizeIndentation(line string) string {
	return strings.ReplaceAll(line, "\t", "    ")
}

// deepCopyItem creates a deep copy of a TodoItem
func deepCopyItem(item *TodoItem) *TodoItem {
	newItem := &TodoItem{
		Completed:   item.Completed,
		Text:        item.Text,
		SubItems:    make([]*TodoItem, 0, len(item.SubItems)),
		BulletLines: make([]string, len(item.BulletLines)),
	}

	// Copy bullet lines
	copy(newItem.BulletLines, item.BulletLines)

	// Recursively copy sub-items
	for _, subItem := range item.SubItems {
		newItem.SubItems = append(newItem.SubItems, deepCopyItem(subItem))
	}

	return newItem
}

// tagCompletedItems adds date tags to completed items
func tagCompletedItems(journal *TodoJournal, currentDate string) {
	if err := validateDate(currentDate); err != nil {
		return // Skip tagging if date is invalid
	}

	for _, day := range journal.Days {
		for _, item := range day.Items {
			tagCompletedItemsRecursive(item, currentDate)
		}
	}
}

// tagCompletedItemsRecursive recursively tags completed items
func tagCompletedItemsRecursive(item *TodoItem, currentDate string) {
	if item.Completed {
		dateTag := "#" + currentDate
		if !strings.Contains(item.Text, dateTag) {
			item.Text += " " + dateTag
		}
	}

	for _, subItem := range item.SubItems {
		tagCompletedItemsRecursive(subItem, currentDate)
	}
}

// tagCompletedSubtasks adds date tags to completed subtasks in uncompleted journals
func tagCompletedSubtasks(journal *TodoJournal, originalDate string) {
	for _, day := range journal.Days {
		for _, item := range day.Items {
			tagCompletedSubtasksRecursive(item, originalDate)
		}
	}
}

// tagCompletedSubtasksRecursive recursively tags completed subtasks
func tagCompletedSubtasksRecursive(item *TodoItem, currentDate string) {
	for _, subItem := range item.SubItems {
		if subItem.Completed {
			dateTag := "#" + currentDate
			if !strings.Contains(subItem.Text, dateTag) {
				subItem.Text += " " + dateTag
			}
		}
		tagCompletedSubtasksRecursive(subItem, currentDate)
	}
}

// journalToString converts a TodoJournal back to string format
func journalToString(journal *TodoJournal) string {
	var result strings.Builder

	for dayIndex, day := range journal.Days {
		if dayIndex > 0 {
			result.WriteString("\n")
		}

		// Write day header
		result.WriteString("- [[" + day.Date + "]]\n")

		// Write items for this day
		for _, item := range day.Items {
			writeItem(&result, item, 0)
		}
	}

	return result.String()
}

// writeItem writes a TodoItem and its subitems to the result builder
func writeItem(result *strings.Builder, item *TodoItem, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)
	status := " "
	if item.Completed {
		status = "x"
	}

	result.WriteString(indent + "- [" + status + "] " + item.Text + "\n")

	// Write bullet lines
	for _, bulletLine := range item.BulletLines {
		result.WriteString(bulletLine + "\n")
	}

	// Write sub-items
	for _, subItem := range item.SubItems {
		writeItem(result, subItem, indentLevel+1)
	}
}
