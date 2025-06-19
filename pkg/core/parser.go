// Package core provides shared parsing functionality for the todoer application.
package core

import (
	"fmt"
	"strings"
	"time"
)

// ValidateDate validates that a date string is in the correct format
func ValidateDate(dateStr string) error {
	_, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return fmt.Errorf("invalid date format '%s', expected YYYY-MM-DD", dateStr)
	}
	return nil
}

// parserState holds the state during parsing to reduce parameter passing
type parserState struct {
	currentDay         *DaySection // The current day being parsed
	currentIndentStack []int       // A stack of indentation levels for the current hierarchy of todo items
	currentItemStack   []*TodoItem // A stack of todo items corresponding to the indent stack
}

// newParserState creates a new parser state
func newParserState() *parserState {
	return &parserState{
		currentDay:         nil,
		currentIndentStack: []int{},
		currentItemStack:   []*TodoItem{},
	}
}

// reset resets the parser state for a new day, clearing the indentation and item stacks.
func (ps *parserState) reset() {
	ps.currentIndentStack = []int{}
	ps.currentItemStack = []*TodoItem{}
}

// ParseTodosSection parses the Todos section into a structured format
func ParseTodosSection(content string) (*TodoJournal, error) {
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

// processLine processes a single line of the Todos section
func processLine(journal *TodoJournal, state *parserState, line string, lineNum int) error {
	trimmedLine := strings.TrimSpace(line)
	if trimmedLine == "" {
		return nil
	}

	// Check for day header
	if dateMatch := DayHeaderRegex.FindStringSubmatch(trimmedLine); dateMatch != nil {
		return processDayHeader(journal, state, dateMatch[1])
	}

	// Skip processing if we don't have a current day
	if state.currentDay == nil {
		return nil
	}

	// Check for todo item first
	if todoMatch := TodoItemRegex.FindStringSubmatch(line); todoMatch != nil {
		return processTodoItem(state, todoMatch)
	}

	// Check for bullet entry (- something that's not a todo)
	if bulletMatch := BulletEntryRegex.FindStringSubmatch(line); bulletMatch != nil {
		return processAssociatedLine(state, line, bulletMatch)
	}

	// Check for continuation line (indented text that's part of a bullet or todo)
	if contMatch := ContinuationRegex.FindStringSubmatch(line); contMatch != nil {
		return processAssociatedLine(state, line, contMatch)
	}

	// If we have a current day and the line is not empty but doesn't match any pattern,
	// it's an unparseable line.
	return fmt.Errorf("unparseable line %d: %q", lineNum, line)
}

// processDayHeader processes a day header line
func processDayHeader(journal *TodoJournal, state *parserState, dateStr string) error {
	// Validate the date format
	if err := ValidateDate(dateStr); err != nil {
		return fmt.Errorf("invalid date in day header: %w", err)
	}

	state.currentDay = createNewDaySection(journal, state.currentDay, dateStr)
	state.reset()
	return nil
}

// processTodoItem processes a todo item line
func processTodoItem(state *parserState, todoMatch []string) error {
	item := createTodoItem(todoMatch)
	indentLevel := GetIndentLevel(todoMatch[1])
	state.currentIndentStack, state.currentItemStack = addItemToHierarchy(
		state.currentDay, item, indentLevel, state.currentIndentStack, state.currentItemStack)
	return nil
}

// processAssociatedLine processes a line that is associated with a todo item,
// like a bullet point or a continuation line. It finds the correct parent todo item
// based on indentation and appends the line to its BulletLines.
func processAssociatedLine(state *parserState, line string, matches []string) error {
	if len(state.currentItemStack) > 0 {
		normalizedLine := NormalizeIndentation(line)
		indent := GetIndentLevel(matches[1])
		targetItem := findTargetItemForBullet(state.currentItemStack, state.currentIndentStack, indent)
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
// If no such parent is found (e.g., the bullet's indentation is less than or
// equal to all items in the stack), it attaches the bullet to the most recently
// added todo item as a reasonable default.
//
// Example:
//   - [ ] Todo item (indent 0)
//   - Bullet entry (indent 2) -> attaches to todo above
//   - [ ] Sub todo (indent 4)
//   - Sub bullet (indent 6) -> attaches to sub todo above
func findTargetItemForBullet(currentItemStack []*TodoItem, currentIndentStack []int, bulletIndent int) *TodoItem {
	// Use the minimum length to avoid out-of-bounds access
	minLen := len(currentItemStack)
	if len(currentIndentStack) < minLen {
		minLen = len(currentIndentStack)
	}
	for i := minLen - 1; i >= 0; i-- {
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
