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
