// Package core provides shared utility functions for the todoer application.
package core

import (
	"strings"
)

// GetIndentLevel calculates the indentation level of a line (number of leading spaces/tabs)
// Treats tabs as 2 spaces for consistency (standardized from original parser.go behavior)
func GetIndentLevel(line string) int {
	indent := 0
	for _, char := range line {
		if char == ' ' {
			indent++
		} else if char == '\t' {
			indent += 2 // Treat tab as 2 spaces (standardized)
		} else {
			break
		}
	}
	return indent
}

// NormalizeIndentation converts tabs to spaces for consistent indentation handling
func NormalizeIndentation(line string) string {
	return strings.ReplaceAll(line, "\t", "  ") // 2 spaces per tab (standardized)
}

// DeepCopyItem creates a deep copy of a todo item
// Pre-allocates slices for better performance with large hierarchies
func DeepCopyItem(item *TodoItem) *TodoItem {
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
		copy.SubItems = append(copy.SubItems, DeepCopyItem(subItem))
	}

	return copy
}

// IsCompleted checks if a todo item and all its subitems are completed
func IsCompleted(item *TodoItem) bool {
	if !item.Completed {
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

// HasDateTag checks if text already has a date tag
func HasDateTag(text string) bool {
	return DateTagRegex.MatchString(text)
}
