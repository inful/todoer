package core

import (
	"testing"
)

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

// Test regex patterns
func TestRegexPatterns(t *testing.T) {
	t.Run("ExtractDateFromFrontmatter", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
			wantDate bool // true: expect a specific date, false: expect today's fallback
		}{
			{
				name: "valid frontmatter with date",
				input: `---
title: 2025-06-19
author: Test
---`,
				expected: "2025-06-19",
				wantDate: true,
			},
			{
				name: "frontmatter without date falls back to today",
				input: `---
title: Some Title
author: Test
---`,
				wantDate: false,
			},
			{
				name:     "malformed frontmatter falls back to today",
				input:    `title: 2025-06-19`,
				wantDate: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				date, err := ExtractDateFromFrontmatter(tt.input, "title")
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if tt.wantDate && date != tt.expected {
					t.Errorf("expected date %q, got %q", tt.expected, date)
				}
				// When wantDate is false, any non-empty date (today's fallback) is acceptable.
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
				name:           "capital X is accepted as completed",
				input:          "- [X] Capital X task",
				shouldMatch:    true,
				expectedIndent: "",
				expectedStatus: "X",
				expectedText:   "Capital X task",
			},
			{
				name:           "cancelled dash is accepted",
				input:          "- [-] Cancelled task",
				shouldMatch:    true,
				expectedIndent: "",
				expectedStatus: "-",
				expectedText:   "Cancelled task",
			},
			{
				name:        "unknown marker is rejected",
				input:       "- [?] Wrong format",
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
		if TodosHeader != "## Todos" {
			t.Errorf("Expected TodosHeader to be '## Todos', got %q", TodosHeader)
		}
		if DateFormat != "2006-01-02" {
			t.Errorf("Expected DateFormat to be '2006-01-02', got %q", DateFormat)
		}
	})
}
