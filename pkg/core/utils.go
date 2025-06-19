// Package core provides shared utility functions for the todoer application.
package core

import (
	"strings"
	"time"
)

// Constants for indentation handling
const (
	// TabSpaces defines how many spaces a tab character represents
	TabSpaces = 2
)

// GetIndentLevel calculates the indentation level of a line (number of leading spaces/tabs).
// Treats tabs as TabSpaces (2) spaces for consistency.
// Returns 0 for nil or empty strings.
func GetIndentLevel(line string) int {
	if line == "" {
		return 0
	}

	indent := 0
	for _, char := range line {
		if char == ' ' {
			indent++
		} else if char == '\t' {
			indent += TabSpaces
		} else {
			break
		}
	}
	return indent
}

// NormalizeIndentation converts tabs to spaces for consistent indentation handling.
// Uses TabSpaces constant to ensure consistency across the application.
// Returns empty string for empty input.
func NormalizeIndentation(line string) string {
	if line == "" {
		return ""
	}
	return strings.ReplaceAll(line, "\t", strings.Repeat(" ", TabSpaces))
}

// DeepCopyItem creates a deep copy of a todo item and all its nested content.
// Returns nil if the input item is nil.
// Pre-allocates slices for better performance with large hierarchies.
func DeepCopyItem(item *TodoItem) *TodoItem {
	if item == nil {
		return nil
	}

	copy := &TodoItem{
		Completed:   item.Completed,
		Text:        item.Text,
		SubItems:    make([]*TodoItem, 0, len(item.SubItems)),
		BulletLines: make([]string, 0, len(item.BulletLines)),
	}

	// Copy bullet lines efficiently
	if len(item.BulletLines) > 0 {
		copy.BulletLines = append(copy.BulletLines, item.BulletLines...)
	}

	// Copy subitems recursively
	for _, subItem := range item.SubItems {
		if copiedSubItem := DeepCopyItem(subItem); copiedSubItem != nil {
			copy.SubItems = append(copy.SubItems, copiedSubItem)
		}
	}

	return copy
}

// IsCompleted checks if a todo item and all its subitems are completed.
// Returns false if the item is nil or if any subitem is not completed.
// Uses recursive checking to ensure the entire hierarchy is completed.
func IsCompleted(item *TodoItem) bool {
	if item == nil || !item.Completed {
		return false
	}

	// Check if all subitems are completed too
	for _, subItem := range item.SubItems {
		if !IsCompleted(subItem) {
			return false
		}
	}

	return true
}

// HasDateTag checks if text already contains a date tag in the format #YYYY-MM-DD.
// Returns false for empty strings.
func HasDateTag(text string) bool {
	if text == "" {
		return false
	}
	return DateTagRegex.MatchString(text)
}

// CountTotalItems recursively counts all todo items in a slice, including nested subitems.
// This is useful for getting statistics about the total number of tasks.
func CountTotalItems(items []*TodoItem) int {
	if len(items) == 0 {
		return 0
	}

	count := len(items) // Count top-level items
	for _, item := range items {
		if item != nil && len(item.SubItems) > 0 {
			count += CountTotalItems(item.SubItems) // Add nested items recursively
		}
	}
	return count
}

// CountCompletedItems recursively counts all completed todo items in a slice.
// Only counts items where IsCompleted returns true (item and all subitems completed).
func CountCompletedItems(items []*TodoItem) int {
	if len(items) == 0 {
		return 0
	}

	count := 0
	for _, item := range items {
		if IsCompleted(item) {
			count++
		}
		// Always check subitems, even if parent is not completed
		if item != nil && len(item.SubItems) > 0 {
			count += CountCompletedItems(item.SubItems)
		}
	}
	return count
}

// GetMaxIndentLevel finds the maximum indentation level in a slice of todo items.
// This can be useful for formatting or layout calculations.
func GetMaxIndentLevel(items []*TodoItem, currentLevel int) int {
	if len(items) == 0 {
		return currentLevel
	}

	maxLevel := currentLevel
	for _, item := range items {
		if item != nil && len(item.SubItems) > 0 {
			subMaxLevel := GetMaxIndentLevel(item.SubItems, currentLevel+1)
			if subMaxLevel > maxLevel {
				maxLevel = subMaxLevel
			}
		}
	}
	return maxLevel
}

// DateVariables holds formatted date variants for template usage
type DateVariables struct {
	Short      string // 06/20/25
	Long       string // June 20, 2025
	Year       string // 2025
	Month      string // 06
	MonthName  string // June
	Day        string // 20
	DayName    string // Friday
	WeekNumber int    // 25 (week of year)
}

// FormatDateVariables creates formatted date variants from a date string in YYYY-MM-DD format.
// Returns empty DateVariables if the date string is empty or invalid.
func FormatDateVariables(dateStr string) DateVariables {
	vars := DateVariables{}

	if dateStr == "" {
		return vars
	}

	date, err := time.Parse(DateFormat, dateStr)
	if err != nil {
		return vars
	}

	vars.Short = date.Format("01/02/06")
	vars.Long = date.Format("January 2, 2006")
	vars.Year = date.Format("2006")
	vars.Month = date.Format("01")
	vars.MonthName = date.Format("January")
	vars.Day = date.Format("02")
	vars.DayName = date.Format("Monday")

	// Calculate week number (ISO 8601 week)
	_, week := date.ISOWeek()
	vars.WeekNumber = week

	return vars
}

// TodoStatistics holds calculated statistics about todos for template usage
type TodoStatistics struct {
	TotalTodos     int      // Total number of incomplete todos
	CompletedTodos int      // Number of completed todos
	TodoDates      []string // Unique dates that todos came from
	OldestTodoDate string   // Date of the oldest incomplete todo
	TodoDaysSpan   int      // Number of days spanned by todos
}

// CalculateTodoStatistics analyzes a journal and calculates statistics for template usage.
// Returns statistics about incomplete and completed todos, date spans, etc.
func CalculateTodoStatistics(journal *TodoJournal, currentDate string) TodoStatistics {
	stats := TodoStatistics{}

	if journal == nil || journal.IsEmpty() {
		return stats
	}

	// Split journal into completed and incomplete todos
	completed, incomplete := SplitJournal(journal)

	// Calculate basic counts
	stats.TotalTodos = CountTotalItems(getAllTodosFromJournal(incomplete))
	stats.CompletedTodos = CountTotalItems(getAllTodosFromJournal(completed))

	// Extract unique dates from both completed and incomplete todos
	dateSet := make(map[string]bool)
	var oldestDate string

	// Add dates from incomplete todos (these are the active todo dates)
	for _, day := range incomplete.Days {
		if day != nil && !day.IsEmpty() {
			dateSet[day.Date] = true
			if oldestDate == "" || day.Date < oldestDate {
				oldestDate = day.Date
			}
		}
	}

	// Also add dates from completed todos for a complete picture
	for _, day := range completed.Days {
		if day != nil && !day.IsEmpty() {
			dateSet[day.Date] = true
			if oldestDate == "" || day.Date < oldestDate {
				oldestDate = day.Date
			}
		}
	}

	// Convert date set to sorted slice
	for date := range dateSet {
		stats.TodoDates = append(stats.TodoDates, date)
	}

	// Sort dates
	if len(stats.TodoDates) > 1 {
		for i := 0; i < len(stats.TodoDates)-1; i++ {
			for j := i + 1; j < len(stats.TodoDates); j++ {
				if stats.TodoDates[i] > stats.TodoDates[j] {
					stats.TodoDates[i], stats.TodoDates[j] = stats.TodoDates[j], stats.TodoDates[i]
				}
			}
		}
	}

	stats.OldestTodoDate = oldestDate

	// Calculate days span if we have an oldest date
	if oldestDate != "" && currentDate != "" {
		stats.TodoDaysSpan = calculateDaysSpan(oldestDate, currentDate)
	}

	return stats
}

// getAllTodosFromJournal extracts all todo items from a journal as a flat slice
func getAllTodosFromJournal(journal *TodoJournal) []*TodoItem {
	var allTodos []*TodoItem

	if journal == nil {
		return allTodos
	}

	for _, day := range journal.Days {
		if day != nil {
			allTodos = append(allTodos, day.Items...)
		}
	}

	return allTodos
}

// calculateDaysSpan calculates the number of days between two dates in YYYY-MM-DD format
func calculateDaysSpan(startDate, endDate string) int {
	if startDate == "" || endDate == "" {
		return 0
	}

	start, err := time.Parse(DateFormat, startDate)
	if err != nil {
		return 0
	}

	end, err := time.Parse(DateFormat, endDate)
	if err != nil {
		return 0
	}

	if end.Before(start) {
		return 0
	}

	// Calculate difference in days
	diff := end.Sub(start)
	return int(diff.Hours() / 24)
}
