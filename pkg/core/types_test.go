package core

import (
	"testing"
)

// Test TodoItem methods
func TestTodoItem_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		item     *TodoItem
		expected bool
	}{
		{
			name:     "nil item should be empty",
			item:     nil,
			expected: true,
		},
		{
			name: "item with only empty fields should be empty",
			item: &TodoItem{
				Completed:   false,
				Text:        "",
				SubItems:    []*TodoItem{},
				BulletLines: []string{},
			},
			expected: true,
		},
		{
			name: "item with text should not be empty",
			item: &TodoItem{
				Completed:   false,
				Text:        "Some task",
				SubItems:    []*TodoItem{},
				BulletLines: []string{},
			},
			expected: false,
		},
		{
			name: "item with subitems should not be empty",
			item: &TodoItem{
				Completed: false,
				Text:      "",
				SubItems: []*TodoItem{
					{Text: "Subtask"},
				},
				BulletLines: []string{},
			},
			expected: false,
		},
		{
			name: "item with bullet lines should not be empty",
			item: &TodoItem{
				Completed:   false,
				Text:        "",
				SubItems:    []*TodoItem{},
				BulletLines: []string{"- Some note"},
			},
			expected: false,
		},
		{
			name: "item with all fields populated should not be empty",
			item: &TodoItem{
				Completed: true,
				Text:      "Main task",
				SubItems: []*TodoItem{
					{Text: "Subtask"},
				},
				BulletLines: []string{"- Note 1", "- Note 2"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.IsEmpty()
			if result != tt.expected {
				t.Errorf("IsEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestTodoItem_HasSubItems(t *testing.T) {
	tests := []struct {
		name     string
		item     *TodoItem
		expected bool
	}{
		{
			name:     "nil item should not have subitems",
			item:     nil,
			expected: false,
		},
		{
			name: "item with empty subitems should not have subitems",
			item: &TodoItem{
				Text:     "Task",
				SubItems: []*TodoItem{},
			},
			expected: false,
		},
		{
			name: "item with nil subitems should not have subitems",
			item: &TodoItem{
				Text:     "Task",
				SubItems: nil,
			},
			expected: false,
		},
		{
			name: "item with subitems should have subitems",
			item: &TodoItem{
				Text: "Task",
				SubItems: []*TodoItem{
					{Text: "Subtask 1"},
					{Text: "Subtask 2"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.HasSubItems()
			if result != tt.expected {
				t.Errorf("HasSubItems() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestTodoItem_HasBulletLines(t *testing.T) {
	tests := []struct {
		name     string
		item     *TodoItem
		expected bool
	}{
		{
			name:     "nil item should not have bullet lines",
			item:     nil,
			expected: false,
		},
		{
			name: "item with empty bullet lines should not have bullet lines",
			item: &TodoItem{
				Text:        "Task",
				BulletLines: []string{},
			},
			expected: false,
		},
		{
			name: "item with nil bullet lines should not have bullet lines",
			item: &TodoItem{
				Text:        "Task",
				BulletLines: nil,
			},
			expected: false,
		},
		{
			name: "item with bullet lines should have bullet lines",
			item: &TodoItem{
				Text:        "Task",
				BulletLines: []string{"- Note 1", "- Note 2"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.HasBulletLines()
			if result != tt.expected {
				t.Errorf("HasBulletLines() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test DaySection methods
func TestDaySection_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		day      *DaySection
		expected bool
	}{
		{
			name:     "nil day section should be empty",
			day:      nil,
			expected: true,
		},
		{
			name: "day section with empty items should be empty",
			day: &DaySection{
				Date:  "2025-06-19",
				Items: []*TodoItem{},
			},
			expected: true,
		},
		{
			name: "day section with nil items should be empty",
			day: &DaySection{
				Date:  "2025-06-19",
				Items: nil,
			},
			expected: true,
		},
		{
			name: "day section with items should not be empty",
			day: &DaySection{
				Date: "2025-06-19",
				Items: []*TodoItem{
					{Text: "Task 1"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.day.IsEmpty()
			if result != tt.expected {
				t.Errorf("IsEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDaySection_ItemCount(t *testing.T) {
	tests := []struct {
		name     string
		day      *DaySection
		expected int
	}{
		{
			name:     "nil day section should have 0 items",
			day:      nil,
			expected: 0,
		},
		{
			name: "day section with empty items should have 0 items",
			day: &DaySection{
				Date:  "2025-06-19",
				Items: []*TodoItem{},
			},
			expected: 0,
		},
		{
			name: "day section with nil items should have 0 items",
			day: &DaySection{
				Date:  "2025-06-19",
				Items: nil,
			},
			expected: 0,
		},
		{
			name: "day section with multiple items should return correct count",
			day: &DaySection{
				Date: "2025-06-19",
				Items: []*TodoItem{
					{Text: "Task 1"},
					{Text: "Task 2"},
					{Text: "Task 3"},
				},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.day.ItemCount()
			if result != tt.expected {
				t.Errorf("ItemCount() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test TodoJournal methods
func TestTodoJournal_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		journal  *TodoJournal
		expected bool
	}{
		{
			name:     "nil journal should be empty",
			journal:  nil,
			expected: true,
		},
		{
			name: "journal with empty days should be empty",
			journal: &TodoJournal{
				Days: []*DaySection{},
			},
			expected: true,
		},
		{
			name: "journal with nil days should be empty",
			journal: &TodoJournal{
				Days: nil,
			},
			expected: true,
		},
		{
			name: "journal with days should not be empty",
			journal: &TodoJournal{
				Days: []*DaySection{
					{Date: "2025-06-19"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.journal.IsEmpty()
			if result != tt.expected {
				t.Errorf("IsEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestTodoJournal_DayCount(t *testing.T) {
	tests := []struct {
		name     string
		journal  *TodoJournal
		expected int
	}{
		{
			name:     "nil journal should have 0 days",
			journal:  nil,
			expected: 0,
		},
		{
			name: "journal with empty days should have 0 days",
			journal: &TodoJournal{
				Days: []*DaySection{},
			},
			expected: 0,
		},
		{
			name: "journal with nil days should have 0 days",
			journal: &TodoJournal{
				Days: nil,
			},
			expected: 0,
		},
		{
			name: "journal with multiple days should return correct count",
			journal: &TodoJournal{
				Days: []*DaySection{
					{Date: "2025-06-19"},
					{Date: "2025-06-20"},
					{Date: "2025-06-21"},
				},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.journal.DayCount()
			if result != tt.expected {
				t.Errorf("DayCount() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test regex patterns
func TestRegexPatterns(t *testing.T) {
	t.Run("FrontmatterDateRegex", func(t *testing.T) {
		tests := []struct {
			name        string
			input       string
			expected    string
			shouldMatch bool
		}{
			{
				name: "valid frontmatter with date",
				input: `---
title: 2025-06-19
author: Test
---`,
				expected:    "2025-06-19",
				shouldMatch: true,
			},
			{
				name: "frontmatter without date",
				input: `---
title: Some Title
author: Test
---`,
				expected:    "",
				shouldMatch: false,
			},
			{
				name:        "malformed frontmatter",
				input:       `title: 2025-06-19`,
				expected:    "",
				shouldMatch: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				matches := FrontmatterDateRegex.FindStringSubmatch(tt.input)
				if tt.shouldMatch {
					if len(matches) < 2 {
						t.Errorf("Expected to find date match, but got %v", matches)
					} else if matches[1] != tt.expected {
						t.Errorf("Expected date %q, got %q", tt.expected, matches[1])
					}
				} else {
					if len(matches) >= 2 {
						t.Errorf("Expected no match, but got %v", matches)
					}
				}
			})
		}
	})

	t.Run("DayHeaderRegex", func(t *testing.T) {
		tests := []struct {
			name        string
			input       string
			expected    string
			shouldMatch bool
		}{
			{
				name:        "valid day header",
				input:       "- [[2025-06-19]]",
				expected:    "2025-06-19",
				shouldMatch: true,
			},
			{
				name:        "invalid day header format",
				input:       "- [2025-06-19]",
				expected:    "",
				shouldMatch: false,
			},
			{
				name:        "text with day header",
				input:       "Some text - [[2025-06-19]] more text",
				expected:    "2025-06-19",
				shouldMatch: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				matches := DayHeaderRegex.FindStringSubmatch(tt.input)
				if tt.shouldMatch {
					if len(matches) < 2 {
						t.Errorf("Expected to find date match, but got %v", matches)
					} else if matches[1] != tt.expected {
						t.Errorf("Expected date %q, got %q", tt.expected, matches[1])
					}
				} else {
					if len(matches) >= 2 {
						t.Errorf("Expected no match, but got %v", matches)
					}
				}
			})
		}
	})

	t.Run("TodoItemRegex", func(t *testing.T) {
		tests := []struct {
			name           string
			input          string
			shouldMatch    bool
			expectedIndent string
			expectedStatus string
			expectedText   string
		}{
			{
				name:           "completed todo with no indentation",
				input:          "- [x] Complete task",
				shouldMatch:    true,
				expectedIndent: "",
				expectedStatus: "x",
				expectedText:   "Complete task",
			},
			{
				name:           "uncompleted todo with indentation",
				input:          "  - [ ] Incomplete task",
				shouldMatch:    true,
				expectedIndent: "  ",
				expectedStatus: " ",
				expectedText:   "Incomplete task",
			},
			{
				name:        "bullet point (not todo)",
				input:       "- Some bullet point",
				shouldMatch: false,
			},
			{
				name:        "malformed todo",
				input:       "- [X] Wrong format",
				shouldMatch: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				matches := TodoItemRegex.FindStringSubmatch(tt.input)
				if tt.shouldMatch {
					if len(matches) < 4 {
						t.Errorf("Expected to find todo match with 4 groups, but got %v", matches)
					} else {
						if matches[1] != tt.expectedIndent {
							t.Errorf("Expected indent %q, got %q", tt.expectedIndent, matches[1])
						}
						if matches[2] != tt.expectedStatus {
							t.Errorf("Expected status %q, got %q", tt.expectedStatus, matches[2])
						}
						if matches[3] != tt.expectedText {
							t.Errorf("Expected text %q, got %q", tt.expectedText, matches[3])
						}
					}
				} else {
					if len(matches) >= 4 {
						t.Errorf("Expected no match, but got %v", matches)
					}
				}
			})
		}
	})

	t.Run("DateTagRegex", func(t *testing.T) {
		tests := []struct {
			name        string
			input       string
			shouldMatch bool
		}{
			{
				name:        "text with date tag",
				input:       "Complete task #2025-06-19",
				shouldMatch: true,
			},
			{
				name:        "multiple date tags",
				input:       "Task #2025-06-19 and #2025-06-20",
				shouldMatch: true,
			},
			{
				name:        "no date tag",
				input:       "Task without date",
				shouldMatch: false,
			},
			{
				name:        "malformed date tag",
				input:       "Task #25-06-19",
				shouldMatch: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				matches := DateTagRegex.FindString(tt.input)
				if tt.shouldMatch {
					if matches == "" {
						t.Errorf("Expected to find date tag match, but got empty string")
					}
				} else {
					if matches != "" {
						t.Errorf("Expected no match, but got %q", matches)
					}
				}
			})
		}
	})
}

// Test constants
func TestConstants(t *testing.T) {
	t.Run("Constants are defined correctly", func(t *testing.T) {
		if TodosHeader != "## TODOS" {
			t.Errorf("Expected TodosHeader to be '## TODOS', got %q", TodosHeader)
		}
		if DateFormat != "2006-01-02" {
			t.Errorf("Expected DateFormat to be '2006-01-02', got %q", DateFormat)
		}
		if CompletedMarker != "x" {
			t.Errorf("Expected CompletedMarker to be 'x', got %q", CompletedMarker)
		}
		if UncompletedMarker != " " {
			t.Errorf("Expected UncompletedMarker to be ' ', got %q", UncompletedMarker)
		}
	})
}
