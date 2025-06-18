// Package core provides shared data structures and constants for the todoer application.
package core

import (
	"regexp"
)

// Constants for parsing
const (
	TodosHeader = "## TODOS"
	DateFormat  = "2006-01-02"
)

// Compiled regex patterns for better performance
var (
	FrontmatterDateRegex = regexp.MustCompile(`(?s)---.*?title:\s*(\d{4}-\d{2}-\d{2}).*?---`)
	NextSectionRegex     = regexp.MustCompile(`\n\n## `)
	DayHeaderRegex       = regexp.MustCompile(`- \[\[(\d{4}-\d{2}-\d{2})\]\]`)
	TodoItemRegex        = regexp.MustCompile(`^(\s*)- \[([ x])\] (.+)$`)
	BulletEntryRegex     = regexp.MustCompile(`^(\s*)- (.+)$`)
	ContinuationRegex    = regexp.MustCompile(`^(\s+)(.+)$`)
	DateTagRegex         = regexp.MustCompile(`#\d{4}-\d{2}-\d{2}`)
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
