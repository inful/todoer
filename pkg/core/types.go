// Package core provides shared data structures and constants for the todoer application.
package core

import (
	"regexp"
)

// Constants for parsing and formatting
const (
	// TodosHeader is the markdown header that identifies the Todos section
	TodosHeader = "## Todos"
	// DateFormat is the standard date format used throughout the application (YYYY-MM-DD)
	DateFormat = "2006-01-02"
	// CompletedMarker is the character used to mark completed todos
	CompletedMarker = "x"
	// UncompletedMarker is the character used to mark uncompleted todos
	UncompletedMarker = " "
)

// Compiled regex patterns for better performance
var (
	// FrontmatterDateRegex matches dates in YAML frontmatter title fields
	// Pattern: ---...title: YYYY-MM-DD...--- (supports multiline frontmatter)
	FrontmatterDateRegex = regexp.MustCompile(`(?s)---.*?title:\s*(\d{4}-\d{2}-\d{2}).*?---`)

	// NextSectionRegex matches the start of the next markdown section (## Header)
	// Used to find the end of the TODOS section
	NextSectionRegex = regexp.MustCompile(`\n\n## `)

	// DayHeaderRegex matches day headers in the format "- [[YYYY-MM-DD]]"
	DayHeaderRegex = regexp.MustCompile(`- \[\[(\d{4}-\d{2}-\d{2})\]\]`)

	// TodoItemRegex matches todo items: "  - [x] Task text" or "  - [ ] Task text"
	// Captures: (indentation, completion_status, text)
	TodoItemRegex = regexp.MustCompile(`^(\s*)- \[([ x])\] (.+)$`)

	// BulletEntryRegex matches bullet entries: "  - Some text"
	// Captures: (indentation, text)
	BulletEntryRegex = regexp.MustCompile(`^(\s*)- (.+)$`)

	// ContinuationRegex matches indented continuation lines: "    Some text"
	// Captures: (indentation, text)
	ContinuationRegex = regexp.MustCompile(`^(\s+)(.+)$`)

	// DateTagRegex matches date tags in the format "#YYYY-MM-DD"
	DateTagRegex = regexp.MustCompile(`#\d{4}-\d{2}-\d{2}`)
)

// TodoItem represents a todo item with its completion status, text, and hierarchical structure.
// It supports nested subitems and associated bullet points or continuation lines.
type TodoItem struct {
	Completed   bool        // Whether the todo item is completed
	Text        string      // The main text of the todo item
	SubItems    []*TodoItem // Nested todo items (hierarchical structure)
	BulletLines []string    // Non-todo bullet entries and multiline content associated with this item
}

// IsEmpty returns true if the todo item has no meaningful content
func (t *TodoItem) IsEmpty() bool {
	return t == nil || (t.Text == "" && len(t.SubItems) == 0 && len(t.BulletLines) == 0)
}

// HasSubItems returns true if the todo item has any nested subitems
func (t *TodoItem) HasSubItems() bool {
	return t != nil && len(t.SubItems) > 0
}

// HasBulletLines returns true if the todo item has associated bullet points or continuation lines
func (t *TodoItem) HasBulletLines() bool {
	return t != nil && len(t.BulletLines) > 0
}

// DaySection represents a day's todo items with a specific date.
// Each day section contains all todos scheduled for that particular date.
type DaySection struct {
	Date  string      // Date in YYYY-MM-DD format
	Items []*TodoItem // All todo items for this day
}

// IsEmpty returns true if the day section has no todo items
func (d *DaySection) IsEmpty() bool {
	return d == nil || len(d.Items) == 0
}

// ItemCount returns the total number of top-level items in this day section
func (d *DaySection) ItemCount() int {
	if d == nil {
		return 0
	}
	return len(d.Items)
}

// TodoJournal represents the entire journal containing multiple days of todo items.
// It provides the top-level structure for organizing todos by date.
type TodoJournal struct {
	Days []*DaySection // All day sections in chronological order
}

// IsEmpty returns true if the journal has no day sections
func (j *TodoJournal) IsEmpty() bool {
	return j == nil || len(j.Days) == 0
}

// DayCount returns the number of day sections in the journal
func (j *TodoJournal) DayCount() int {
	if j == nil {
		return 0
	}
	return len(j.Days)
}

// TemplateData holds the data to be passed to Go templates when generating journal files.
// It provides the essential variables needed for template rendering.
type TemplateData struct {
	Date         string // Current date in YYYY-MM-DD format
	TODOS        string // Formatted todos content to be inserted into the template
	PreviousDate string // Date of the previous journal that todos came from (YYYY-MM-DD format, empty if no previous journal)
}
