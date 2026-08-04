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
	// NextSectionRegex matches the start of the next markdown section
	// (## Header). The (?m) flag enables multi-line mode so that
	// ^ anchors to the start of any line. The leading \n matches
	// the line terminator of the previous line, so the matched
	// position points at that newline (which is the end of the
	// Todos content). The previous form (\n\n## ) required a blank
	// line in addition, which silently dropped trailing sections
	// when the next header immediately followed the mandatory
	// blank line.
	NextSectionRegex = regexp.MustCompile(`(?m)^\n## `)

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

// BulletLine is a non-todo note or continuation line attached to a
// TodoItem. Indent is the number of IndentSpaces levels deeper than
// the parent item (0 = one IndentSpaces level deeper than the parent's
// bullet marker — the position of a normal sub-bullet; 1 = one level
// deeper than that, etc.). Text is the line content with no leading
// whitespace. Storing the indent relative to the parent lets the
// writer re-indent the line correctly when the parent is moved to a
// different position in the output (e.g. carried over into today's
// section, which adds one level of depth).
type BulletLine struct {
	Indent int
	Text   string
}

// TodoItem represents a todo item with its completion status, text, and hierarchical structure.
// It supports nested subitems and associated bullet points or continuation lines.
type TodoItem struct {
	Completed   bool         // Whether the todo item is completed
	Text        string       // The main text of the todo item
	SubItems    []*TodoItem  // Nested todo items (hierarchical structure)
	BulletLines []BulletLine // Non-todo bullet entries and multiline content associated with this item
}

// IsEmpty returns true if the todo item has no meaningful content
func (t *TodoItem) IsEmpty() bool {
	return t == nil || (t.Text == "" && len(t.SubItems) == 0 && len(t.BulletLines) == 0)
}

// HasSubItems reports whether the item has any nested subitems.
func (t *TodoItem) HasSubItems() bool {
	return t != nil && len(t.SubItems) > 0
}

// HasBulletLines returns true if the todo item has associated bullet points or continuation lines
func (t *TodoItem) HasBulletLines() bool {
	return t != nil && len(t.BulletLines) > 0
}

// DaySection represents a day's todo items grouped under a specific date.
type DaySection struct {
	Date  string      // Date in YYYY-MM-DD format
	Items []*TodoItem // All todo items for this day
}

// IsEmpty returns true if the day section has no todo items
func (d *DaySection) IsEmpty() bool {
	return d == nil || len(d.Items) == 0
}

// ItemCount returns the number of top-level todo items in this day section.
func (d *DaySection) ItemCount() int {
	if d == nil {
		return 0
	}
	return len(d.Items)
}

// TodoJournal is the top-level structure holding all day sections in chronological order.
type TodoJournal struct {
	Days []*DaySection // All day sections in chronological order
}

// IsEmpty returns true if the journal has no day sections
func (j *TodoJournal) IsEmpty() bool {
	return j == nil || len(j.Days) == 0
}

// DayCount returns the number of day sections in the journal.
func (j *TodoJournal) DayCount() int {
	if j == nil {
		return 0
	}
	return len(j.Days)
}

// TemplateData holds the data to be passed to Go templates when generating journal files.
// It provides comprehensive variables for flexible template rendering including date formatting and todo statistics.
type TemplateData struct {
	Date         string // Current date in YYYY-MM-DD format
	TODOS        string // Formatted todos content to be inserted into the template
	PreviousDate string // Date of the previous journal that todos came from (YYYY-MM-DD format, empty if no previous journal)

	// Current date formatting variants
	DateShort  string // 06/20/25
	DateLong   string // June 20, 2025
	Year       string // 2025
	Month      string // 06
	MonthName  string // June
	Day        string // 20
	DayName    string // Friday
	WeekNumber int    // 25 (week of year)

	// Previous date formatting variants (empty if no previous journal)
	PreviousDateShort  string // 06/19/25
	PreviousDateLong   string // June 19, 2025
	PreviousYear       string // 2025
	PreviousMonth      string // 06
	PreviousMonthName  string // June
	PreviousDay        string // 19
	PreviousDayName    string // Thursday
	PreviousWeekNumber int    // 25 (week of year)

	// Todo statistics
	TotalTodos               int      // Total number of incomplete todos being carried over
	CompletedTodos           int      // Number of completed todos found in source journal
	UncompletedTodos         int      // Number of uncompleted todos found in source journal
	UncompletedTopLevelTodos int      // Number of uncompleted top-level todos
	TodoDates                []string // List of unique dates that todos came from (YYYY-MM-DD format)
	OldestTodoDate           string   // Date of the oldest incomplete todo (YYYY-MM-DD format, empty if no todos)
	TodoDaysSpan             int      // Number of days spanned by todos (from oldest to current date)

	// Custom variables (user-defined via config)
	Custom map[string]any // Custom template variables from configuration
}
