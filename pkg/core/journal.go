// Package core provides shared journal manipulation functionality for the todoer application.
package core

import (
	"strings"
)

// Constants for journal operations
const (
	// DefaultBuilderCapacity is the initial capacity for string builders
	DefaultBuilderCapacity = 1024
	// IndentSpaces is the number of spaces per indentation level
	IndentSpaces = 2
)

// SplitJournal splits the journal into completed and uncompleted tasks.
// It returns two separate journals: one containing only completed items and their
// associated bullet points, and another containing only uncompleted items.
// Days with no items of the respective type are omitted from the result.
func SplitJournal(journal *TodoJournal) (*TodoJournal, *TodoJournal) {
	if journal == nil {
		return &TodoJournal{Days: []*DaySection{}}, &TodoJournal{Days: []*DaySection{}}
	}

	// Pre-allocate with estimated capacity for better performance
	completedJournal := &TodoJournal{
		Days: make([]*DaySection, 0, len(journal.Days)),
	}

	uncompletedJournal := &TodoJournal{
		Days: make([]*DaySection, 0, len(journal.Days)),
	}

	for _, day := range journal.Days {
		if day == nil {
			continue
		}

		completedDay := &DaySection{
			Date:  day.Date,
			Items: make([]*TodoItem, 0, len(day.Items)),
		}

		uncompletedDay := &DaySection{
			Date:  day.Date,
			Items: make([]*TodoItem, 0, len(day.Items)),
		}

		hasCompletedItems := false
		hasUncompletedItems := false

		for _, item := range day.Items {
			if IsCompleted(item) {
				hasCompletedItems = true
				// Create a deep copy of the item for the completed journal
				if copiedItem := DeepCopyItem(item); copiedItem != nil {
					completedDay.Items = append(completedDay.Items, copiedItem)
				}
			} else {
				hasUncompletedItems = true
				// Create a deep copy of the item for the uncompleted journal
				if copiedItem := DeepCopyItem(item); copiedItem != nil {
					uncompletedDay.Items = append(uncompletedDay.Items, copiedItem)
				}
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

// TagCompletedItems adds date tags to completed items in the journal.
// It appends a date tag (e.g., "#2025-06-18") to completed items that don't already have one.
// This function processes both top-level items and all nested subitems recursively.
func TagCompletedItems(journal *TodoJournal, currentDate string) {
	if journal == nil || currentDate == "" {
		return
	}

	for _, day := range journal.Days {
		if day == nil {
			continue
		}
		for _, item := range day.Items {
			tagCompletedItemsRecursive(item, currentDate)
		}
	}
}

// TagCompletedSubitems adds date tags to completed subtasks in uncompleted parent tasks.
// This is useful for tracking when subtasks were completed even if the parent task is still pending.
func TagCompletedSubitems(journal *TodoJournal, originalDate string) {
	if journal == nil || originalDate == "" {
		return
	}

	for _, day := range journal.Days {
		if day == nil {
			continue
		}
		for _, item := range day.Items {
			// Only tag subitems, not the parent item itself
			for _, subItem := range item.SubItems {
				tagCompletedItemsRecursive(subItem, originalDate)
			}
		}
	}
}

// tagCompletedItemsRecursive adds date tags to completed items recursively.
// This unified function handles both the main item and all nested subitems.
func tagCompletedItemsRecursive(item *TodoItem, date string) {
	if item == nil {
		return
	}

	if item.Completed && !HasDateTag(item.Text) {
		item.Text += " #" + date
	}

	// Process all subitems recursively
	for _, subItem := range item.SubItems {
		tagCompletedItemsRecursive(subItem, date)
	}
}

// JournalToString converts a journal to string format.
// It formats the journal as a markdown-style todo list with day headers in the format "- [[YYYY-MM-DD]]".
// Returns an empty string if the journal is nil or has no days.
func JournalToString(journal *TodoJournal) string {
	if journal == nil || len(journal.Days) == 0 {
		return ""
	}

	var builder strings.Builder
	// Pre-allocate some capacity to reduce reallocations
	builder.Grow(1024)

	for _, day := range journal.Days {
		if day == nil {
			continue
		}

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

// writeItemToString writes a todo item to a string builder with proper indentation.
// It recursively writes subitems and preserves the original formatting of bullet lines.
func writeItemToString(builder *strings.Builder, item *TodoItem, depth int) {
	if item == nil {
		return
	}

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

// MoveUndatedTodosToCurrentDate moves incomplete todos that don't have a date (empty date string)
// to the specified current date. Completed undated todos are removed.
// This handles the case where users add todos without specifying dates.
func MoveUndatedTodosToCurrentDate(journal *TodoJournal, currentDate string) *TodoJournal {
	if journal == nil || currentDate == "" {
		return journal
	}

	result := &TodoJournal{Days: []*DaySection{}}
	var currentDateDay *DaySection
	var undatedIncompleteTodos []*TodoItem

	// First pass: collect undated incomplete todos and find/keep dated sections
	for _, day := range journal.Days {
		if day.Date == "" {
			// This is an undated section - collect incomplete todos
			for _, item := range day.Items {
				if !item.Completed {
					undatedIncompleteTodos = append(undatedIncompleteTodos, item)
				}
			}
		} else {
			// Keep dated sections
			if day.Date == currentDate {
				currentDateDay = day
			}
			result.Days = append(result.Days, day)
		}
	}

	// If we have undated incomplete todos, add them to the current date section
	if len(undatedIncompleteTodos) > 0 {
		if currentDateDay == nil {
			// Create current date section if it doesn't exist
			currentDateDay = &DaySection{
				Date:  currentDate,
				Items: undatedIncompleteTodos,
			}
			result.Days = append(result.Days, currentDateDay)
		} else {
			// Add undated todos to the existing current date section
			currentDateDay.Items = append(currentDateDay.Items, undatedIncompleteTodos...)
		}
	}

	return result
}
