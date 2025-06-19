package core

import (
	"testing"
)

// Test GetIndentLevel function
func TestGetIndentLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string should return 0",
			input:    "",
			expected: 0,
		},
		{
			name:     "no indentation should return 0",
			input:    "no indentation",
			expected: 0,
		},
		{
			name:     "two spaces should return 2",
			input:    "  two spaces",
			expected: 2,
		},
		{
			name:     "four spaces should return 4",
			input:    "    four spaces",
			expected: 4,
		},
		{
			name:     "one tab should return TabSpaces",
			input:    "\tone tab",
			expected: TabSpaces,
		},
		{
			name:     "two tabs should return TabSpaces * 2",
			input:    "\t\ttwo tabs",
			expected: TabSpaces * 2,
		},
		{
			name:     "mixed tabs and spaces",
			input:    "\t  mixed",
			expected: TabSpaces + 2,
		},
		{
			name:     "only whitespace",
			input:    "    ",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIndentLevel(tt.input)
			if result != tt.expected {
				t.Errorf("GetIndentLevel(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test NormalizeIndentation function
func TestNormalizeIndentation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string should return empty",
			input:    "",
			expected: "",
		},
		{
			name:     "no tabs should remain unchanged",
			input:    "    no tabs here",
			expected: "    no tabs here",
		},
		{
			name:     "single tab should become spaces",
			input:    "\tsingle tab",
			expected: "  single tab",
		},
		{
			name:     "multiple tabs should become spaces",
			input:    "\t\tmultiple tabs",
			expected: "    multiple tabs",
		},
		{
			name:     "mixed tabs and spaces",
			input:    "\t  mixed",
			expected: "    mixed",
		},
		{
			name:     "tabs in middle of text",
			input:    "text\twith\ttabs",
			expected: "text  with  tabs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeIndentation(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeIndentation(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test DeepCopyItem function
func TestDeepCopyItem(t *testing.T) {
	t.Run("nil item should return nil", func(t *testing.T) {
		result := DeepCopyItem(nil)
		if result != nil {
			t.Errorf("DeepCopyItem(nil) = %v, expected nil", result)
		}
	})

	t.Run("simple item should be copied", func(t *testing.T) {
		original := &TodoItem{
			Completed: true,
			Text:      "Test task",
		}

		copy := DeepCopyItem(original)
		if copy == nil {
			t.Fatal("DeepCopyItem returned nil for valid item")
		}

		// Check that values are copied
		if copy.Completed != original.Completed {
			t.Errorf("Completed not copied correctly: got %v, expected %v", copy.Completed, original.Completed)
		}
		if copy.Text != original.Text {
			t.Errorf("Text not copied correctly: got %q, expected %q", copy.Text, original.Text)
		}

		// Check that it's a different object
		if copy == original {
			t.Error("DeepCopyItem returned the same object, not a copy")
		}

		// Modify original to ensure independence
		original.Completed = false
		original.Text = "Modified"
		if copy.Completed == original.Completed || copy.Text == original.Text {
			t.Error("Copy is not independent of original")
		}
	})

	t.Run("item with bullet lines should be copied", func(t *testing.T) {
		original := &TodoItem{
			Text:        "Task with notes",
			BulletLines: []string{"- Note 1", "- Note 2"},
		}

		copy := DeepCopyItem(original)
		if copy == nil {
			t.Fatal("DeepCopyItem returned nil")
		}

		if len(copy.BulletLines) != len(original.BulletLines) {
			t.Errorf("BulletLines length mismatch: got %d, expected %d", len(copy.BulletLines), len(original.BulletLines))
		}

		for i, line := range copy.BulletLines {
			if line != original.BulletLines[i] {
				t.Errorf("BulletLines[%d] mismatch: got %q, expected %q", i, line, original.BulletLines[i])
			}
		}

		// Test independence
		original.BulletLines[0] = "Modified"
		if copy.BulletLines[0] == original.BulletLines[0] {
			t.Error("BulletLines slice is not independent")
		}
	})

	t.Run("item with subitems should be deep copied", func(t *testing.T) {
		original := &TodoItem{
			Text: "Parent task",
			SubItems: []*TodoItem{
				{Text: "Subtask 1", Completed: true},
				{Text: "Subtask 2", Completed: false},
			},
		}

		copy := DeepCopyItem(original)
		if copy == nil {
			t.Fatal("DeepCopyItem returned nil")
		}

		if len(copy.SubItems) != len(original.SubItems) {
			t.Errorf("SubItems length mismatch: got %d, expected %d", len(copy.SubItems), len(original.SubItems))
		}

		for i, subItem := range copy.SubItems {
			if subItem == original.SubItems[i] {
				t.Errorf("SubItem[%d] is the same object, not a copy", i)
			}
			if subItem.Text != original.SubItems[i].Text {
				t.Errorf("SubItem[%d] text mismatch: got %q, expected %q", i, subItem.Text, original.SubItems[i].Text)
			}
		}

		// Test independence
		original.SubItems[0].Text = "Modified"
		if copy.SubItems[0].Text == original.SubItems[0].Text {
			t.Error("SubItems are not independent")
		}
	})
}

// Test IsCompleted function
func TestIsCompleted(t *testing.T) {
	t.Run("nil item should return false", func(t *testing.T) {
		result := IsCompleted(nil)
		if result {
			t.Error("IsCompleted(nil) should return false")
		}
	})

	t.Run("uncompleted item should return false", func(t *testing.T) {
		item := &TodoItem{Completed: false}
		result := IsCompleted(item)
		if result {
			t.Error("IsCompleted for uncompleted item should return false")
		}
	})

	t.Run("completed item without subitems should return true", func(t *testing.T) {
		item := &TodoItem{Completed: true}
		result := IsCompleted(item)
		if !result {
			t.Error("IsCompleted for completed item without subitems should return true")
		}
	})

	t.Run("completed item with completed subitems should return true", func(t *testing.T) {
		item := &TodoItem{
			Completed: true,
			SubItems: []*TodoItem{
				{Completed: true},
				{Completed: true},
			},
		}
		result := IsCompleted(item)
		if !result {
			t.Error("IsCompleted for completed item with all completed subitems should return true")
		}
	})

	t.Run("completed item with uncompleted subitems should return false", func(t *testing.T) {
		item := &TodoItem{
			Completed: true,
			SubItems: []*TodoItem{
				{Completed: true},
				{Completed: false},
			},
		}
		result := IsCompleted(item)
		if result {
			t.Error("IsCompleted for completed item with uncompleted subitems should return false")
		}
	})
}

// Test HasDateTag function
func TestHasDateTag(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string should return false",
			input:    "",
			expected: false,
		},
		{
			name:     "text without date tag should return false",
			input:    "Task without date",
			expected: false,
		},
		{
			name:     "text with valid date tag should return true",
			input:    "Task #2025-06-19",
			expected: true,
		},
		{
			name:     "text with multiple date tags should return true",
			input:    "Task #2025-06-19 and #2025-06-20",
			expected: true,
		},
		{
			name:     "text with invalid date format should return false",
			input:    "Task #25-06-19",
			expected: false,
		},
		{
			name:     "date tag at beginning should return true",
			input:    "#2025-06-19 Task",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasDateTag(tt.input)
			if result != tt.expected {
				t.Errorf("HasDateTag(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test CountTotalItems function
func TestCountTotalItems(t *testing.T) {
	t.Run("empty slice should return 0", func(t *testing.T) {
		result := CountTotalItems([]*TodoItem{})
		if result != 0 {
			t.Errorf("CountTotalItems([]) = %d, expected 0", result)
		}
	})

	t.Run("nil slice should return 0", func(t *testing.T) {
		result := CountTotalItems(nil)
		if result != 0 {
			t.Errorf("CountTotalItems(nil) = %d, expected 0", result)
		}
	})

	t.Run("flat list should return correct count", func(t *testing.T) {
		items := []*TodoItem{
			{Text: "Task 1"},
			{Text: "Task 2"},
			{Text: "Task 3"},
		}
		result := CountTotalItems(items)
		if result != 3 {
			t.Errorf("CountTotalItems with 3 items = %d, expected 3", result)
		}
	})

	t.Run("nested items should be counted recursively", func(t *testing.T) {
		items := []*TodoItem{
			{
				Text: "Task 1",
				SubItems: []*TodoItem{
					{Text: "Subtask 1.1"},
					{Text: "Subtask 1.2"},
				},
			},
			{Text: "Task 2"},
		}
		// 2 top-level + 2 subitems = 4 total
		result := CountTotalItems(items)
		if result != 4 {
			t.Errorf("CountTotalItems with nested items = %d, expected 4", result)
		}
	})
}

// Test CountCompletedItems function
func TestCountCompletedItems(t *testing.T) {
	t.Run("empty slice should return 0", func(t *testing.T) {
		result := CountCompletedItems([]*TodoItem{})
		if result != 0 {
			t.Errorf("CountCompletedItems([]) = %d, expected 0", result)
		}
	})

	t.Run("no completed items should return 0", func(t *testing.T) {
		items := []*TodoItem{
			{Completed: false, Text: "Task 1"},
			{Completed: false, Text: "Task 2"},
		}
		result := CountCompletedItems(items)
		if result != 0 {
			t.Errorf("CountCompletedItems with no completed items = %d, expected 0", result)
		}
	})

	t.Run("all completed items should return correct count", func(t *testing.T) {
		items := []*TodoItem{
			{Completed: true, Text: "Task 1"},
			{Completed: true, Text: "Task 2"},
		}
		result := CountCompletedItems(items)
		if result != 2 {
			t.Errorf("CountCompletedItems with all completed = %d, expected 2", result)
		}
	})

	t.Run("mixed completion status should count correctly", func(t *testing.T) {
		items := []*TodoItem{
			{Completed: true, Text: "Task 1"},
			{Completed: false, Text: "Task 2"},
			{Completed: true, Text: "Task 3"},
		}
		result := CountCompletedItems(items)
		if result != 2 {
			t.Errorf("CountCompletedItems with mixed status = %d, expected 2", result)
		}
	})
}

// Test GetMaxIndentLevel function
func TestGetMaxIndentLevel(t *testing.T) {
	t.Run("empty slice should return current level", func(t *testing.T) {
		result := GetMaxIndentLevel([]*TodoItem{}, 0)
		if result != 0 {
			t.Errorf("GetMaxIndentLevel([], 0) = %d, expected 0", result)
		}

		result = GetMaxIndentLevel([]*TodoItem{}, 2)
		if result != 2 {
			t.Errorf("GetMaxIndentLevel([], 2) = %d, expected 2", result)
		}
	})

	t.Run("flat list should return current level", func(t *testing.T) {
		items := []*TodoItem{
			{Text: "Task 1"},
			{Text: "Task 2"},
		}
		result := GetMaxIndentLevel(items, 1)
		if result != 1 {
			t.Errorf("GetMaxIndentLevel with flat list = %d, expected 1", result)
		}
	})

	t.Run("nested items should return max level", func(t *testing.T) {
		items := []*TodoItem{
			{
				Text: "Task 1",
				SubItems: []*TodoItem{
					{
						Text: "Subtask 1.1",
						SubItems: []*TodoItem{
							{Text: "Sub-subtask 1.1.1"},
						},
					},
				},
			},
			{Text: "Task 2"},
		}
		// Level 0 (current) + 2 levels of nesting = 2
		result := GetMaxIndentLevel(items, 0)
		if result != 2 {
			t.Errorf("GetMaxIndentLevel with 2 levels of nesting = %d, expected 2", result)
		}
	})
}

// Test TabSpaces constant
func TestTabSpacesConstant(t *testing.T) {
	if TabSpaces != 2 {
		t.Errorf("TabSpaces = %d, expected 2", TabSpaces)
	}
}

// Test FormatDateVariables function
func TestFormatDateVariables(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected DateVariables
	}{
		{
			name:    "valid date should format correctly",
			dateStr: "2025-06-20",
			expected: DateVariables{
				Short:      "06/20/25",
				Long:       "June 20, 2025",
				Year:       "2025",
				Month:      "06",
				MonthName:  "June",
				Day:        "20",
				DayName:    "Friday",
				WeekNumber: 25,
			},
		},
		{
			name:    "empty date should return empty variables",
			dateStr: "",
			expected: DateVariables{},
		},
		{
			name:    "invalid date should return empty variables",
			dateStr: "invalid-date",
			expected: DateVariables{},
		},
		{
			name:    "new year date should handle week correctly",
			dateStr: "2025-01-01",
			expected: DateVariables{
				Short:      "01/01/25",
				Long:       "January 1, 2025",
				Year:       "2025",
				Month:      "01",
				MonthName:  "January",
				Day:        "01",
				DayName:    "Wednesday",
				WeekNumber: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDateVariables(tt.dateStr)

			if result != tt.expected {
				t.Errorf("FormatDateVariables() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}
