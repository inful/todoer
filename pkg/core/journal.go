// Package core provides shared journal manipulation functionality for the todoer application.
package core

import (
	"strings"
)

// SplitJournal splits the journal into completed and uncompleted tasks
func SplitJournal(journal *TodoJournal) (*TodoJournal, *TodoJournal) {
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
			if IsCompleted(item) {
				hasCompletedItems = true
				// Create a deep copy of the item for the completed journal
				completedDay.Items = append(completedDay.Items, DeepCopyItem(item))
			} else {
				hasUncompletedItems = true
				// Create a deep copy of the item for the uncompleted journal
				uncompletedDay.Items = append(uncompletedDay.Items, DeepCopyItem(item))
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

// TagCompletedItems adds date tags to completed items
func TagCompletedItems(journal *TodoJournal, currentDate string) {
	for _, day := range journal.Days {
		for _, item := range day.Items {
			if item.Completed && !HasDateTag(item.Text) {
				item.Text += " #" + currentDate
			}
			// Also tag subitems
			tagCompletedSubitemsRecursive(item, currentDate)
		}
	}
}

// TagCompletedSubtasks adds date tags to completed subtasks in uncompleted parent tasks
func TagCompletedSubtasks(journal *TodoJournal, originalDate string) {
	for _, day := range journal.Days {
		for _, item := range day.Items {
			tagCompletedSubitemsRecursive(item, originalDate)
		}
	}
}

// tagCompletedSubitemsRecursive adds date tags to completed subitems recursively
func tagCompletedSubitemsRecursive(item *TodoItem, date string) {
	for _, subItem := range item.SubItems {
		if subItem.Completed && !HasDateTag(subItem.Text) {
			subItem.Text += " #" + date
		}
		tagCompletedSubitemsRecursive(subItem, date)
	}
}

// JournalToString converts a journal to string format
func JournalToString(journal *TodoJournal) string {
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
