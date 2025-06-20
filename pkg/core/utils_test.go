package core

import (
	"strings"
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
			name:     "empty date should return empty variables",
			dateStr:  "",
			expected: DateVariables{},
		},
		{
			name:     "invalid date should return empty variables",
			dateStr:  "invalid-date",
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

// Test CalculateTodoStatistics function
func TestCalculateTodoStatistics(t *testing.T) {
	tests := []struct {
		name        string
		journal     *TodoJournal
		currentDate string
		expected    TodoStatistics
	}{
		{
			name:        "nil journal should return empty statistics",
			journal:     nil,
			currentDate: "2025-06-20",
			expected:    TodoStatistics{},
		},
		{
			name:        "empty journal should return empty statistics",
			journal:     &TodoJournal{},
			currentDate: "2025-06-20",
			expected:    TodoStatistics{},
		},
		{
			name: "journal with only incomplete todos should calculate correctly",
			journal: &TodoJournal{
				Days: []*DaySection{
					{
						Date: "2025-06-18",
						Items: []*TodoItem{
							{Completed: false, Text: "Task 1"},
							{Completed: false, Text: "Task 2"},
						},
					},
					{
						Date: "2025-06-19",
						Items: []*TodoItem{
							{Completed: false, Text: "Task 3"},
						},
					},
				},
			},
			currentDate: "2025-06-20",
			expected: TodoStatistics{
				TotalTodos:     3,
				CompletedTodos: 0,
				TodoDates:      []string{"2025-06-18", "2025-06-19"},
				OldestTodoDate: "2025-06-18",
				TodoDaysSpan:   2,
			},
		},
		{
			name: "journal with mixed todos should calculate correctly",
			journal: &TodoJournal{
				Days: []*DaySection{
					{
						Date: "2025-06-17",
						Items: []*TodoItem{
							{Completed: false, Text: "Incomplete task"},
							{Completed: true, Text: "Completed task"},
						},
					},
					{
						Date: "2025-06-19",
						Items: []*TodoItem{
							{Completed: false, Text: "Another incomplete"},
						},
					},
				},
			},
			currentDate: "2025-06-20",
			expected: TodoStatistics{
				TotalTodos:     2,
				CompletedTodos: 1,
				TodoDates:      []string{"2025-06-17", "2025-06-19"},
				OldestTodoDate: "2025-06-17",
				TodoDaysSpan:   3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTodoStatistics(tt.journal, tt.currentDate)

			if result.TotalTodos != tt.expected.TotalTodos {
				t.Errorf("TotalTodos = %d, want %d", result.TotalTodos, tt.expected.TotalTodos)
			}
			if result.CompletedTodos != tt.expected.CompletedTodos {
				t.Errorf("CompletedTodos = %d, want %d", result.CompletedTodos, tt.expected.CompletedTodos)
			}
			if result.OldestTodoDate != tt.expected.OldestTodoDate {
				t.Errorf("OldestTodoDate = %s, want %s", result.OldestTodoDate, tt.expected.OldestTodoDate)
			}
			if result.TodoDaysSpan != tt.expected.TodoDaysSpan {
				t.Errorf("TodoDaysSpan = %d, want %d", result.TodoDaysSpan, tt.expected.TodoDaysSpan)
			}
			if len(result.TodoDates) != len(tt.expected.TodoDates) {
				t.Errorf("TodoDates length = %d, want %d", len(result.TodoDates), len(tt.expected.TodoDates))
			} else {
				for i, date := range result.TodoDates {
					if date != tt.expected.TodoDates[i] {
						t.Errorf("TodoDates[%d] = %s, want %s", i, date, tt.expected.TodoDates[i])
					}
				}
			}
		})
	}
}

// Test calculateDaysSpan function
func TestCalculateDaysSpan(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		endDate   string
		expected  int
	}{
		{
			name:      "same date should return 0",
			startDate: "2025-06-20",
			endDate:   "2025-06-20",
			expected:  0,
		},
		{
			name:      "consecutive dates should return 1",
			startDate: "2025-06-19",
			endDate:   "2025-06-20",
			expected:  1,
		},
		{
			name:      "week span should return 7",
			startDate: "2025-06-13",
			endDate:   "2025-06-20",
			expected:  7,
		},
		{
			name:      "empty start date should return 0",
			startDate: "",
			endDate:   "2025-06-20",
			expected:  0,
		},
		{
			name:      "invalid date should return 0",
			startDate: "invalid",
			endDate:   "2025-06-20",
			expected:  0,
		},
		{
			name:      "end before start should return 0",
			startDate: "2025-06-20",
			endDate:   "2025-06-19",
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateDaysSpan(tt.startDate, tt.endDate)
			if result != tt.expected {
				t.Errorf("calculateDaysSpan(%s, %s) = %d, want %d", tt.startDate, tt.endDate, result, tt.expected)
			}
		})
	}
}

// Test custom variables functionality
func TestMergeCustomVariables(t *testing.T) {
	tests := []struct {
		name       string
		data       *TemplateData
		customVars map[string]interface{}
		expected   map[string]interface{}
	}{
		{
			name: "nil data should not panic",
			data: nil,
			customVars: map[string]interface{}{
				"TestVar": "test_value",
			},
			expected: nil,
		},
		{
			name: "nil custom vars should not modify data",
			data: &TemplateData{Date: "2025-06-20"},
			customVars: nil,
			expected: nil,
		},
		{
			name: "valid custom variables should be merged",
			data: &TemplateData{Date: "2025-06-20"},
			customVars: map[string]interface{}{
				"ProjectName": "MyProject",
				"Version":     "1.0.0",
				"Debug":       true,
			},
			expected: map[string]interface{}{
				"ProjectName": "MyProject",
				"Version":     "1.0.0",
				"Debug":       true,
			},
		},
		{
			name: "multiple calls should merge correctly",
			data: &TemplateData{Date: "2025-06-20"},
			customVars: map[string]interface{}{
				"First":  "value1",
				"Second": "value2",
			},
			expected: map[string]interface{}{
				"First":  "value1",
				"Second": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MergeCustomVariables(tt.data, tt.customVars)
			
			if tt.expected == nil {
				if tt.data != nil && tt.data.Custom != nil {
					t.Errorf("Expected Custom to be nil, but got %v", tt.data.Custom)
				}
				return
			}
			
			if tt.data == nil {
				t.Errorf("Data is nil but expected custom variables")
				return
			}
			
			if tt.data.Custom == nil {
				t.Errorf("Custom map is nil but expected values")
				return
			}
			
			for key, expectedValue := range tt.expected {
				actualValue, exists := tt.data.Custom[key]
				if !exists {
					t.Errorf("Expected custom variable '%s' not found", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Custom variable '%s' = %v, want %v", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestValidateCustomVariables(t *testing.T) {
	tests := []struct {
		name        string
		customVars  map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil custom vars should be valid",
			customVars:  nil,
			expectError: false,
		},
		{
			name:        "empty custom vars should be valid",
			customVars:  map[string]interface{}{},
			expectError: false,
		},
		{
			name: "valid custom variables should pass",
			customVars: map[string]interface{}{
				"ProjectName": "MyProject",
				"Version":     "1.0.0",
				"Debug":       true,
				"Count":       42,
				"Rate":        3.14,
				"Tags":        []string{"tag1", "tag2"},
			},
			expectError: false,
		},
		{
			name: "reserved name should fail",
			customVars: map[string]interface{}{
				"Date": "2025-06-20",
			},
			expectError: true,
			errorMsg:    "reserved",
		},
		{
			name: "invalid variable name should fail",
			customVars: map[string]interface{}{
				"123Invalid": "value",
			},
			expectError: true,
			errorMsg:    "not a valid",
		},
		{
			name: "unsupported type should fail",
			customVars: map[string]interface{}{
				"InvalidType": complex(1, 2),
			},
			expectError: true,
			errorMsg:    "unsupported type",
		},
		{
			name: "valid variable names should pass",
			customVars: map[string]interface{}{
				"validName":       "value",
				"Valid_Name":      "value",
				"_validName":      "value",
				"validName123":    "value",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCustomVariables(tt.customVars)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', but got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestIsValidVariableName(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		expected bool
	}{
		{"empty string", "", false},
		{"valid simple name", "validName", true},
		{"valid with underscore", "valid_name", true},
		{"valid starting with underscore", "_validName", true},
		{"valid with numbers", "validName123", true},
		{"invalid starting with number", "123invalid", false},
		{"invalid with special chars", "invalid-name", false},
		{"invalid with spaces", "invalid name", false},
		{"valid all caps", "VALID_NAME", true},
		{"valid mixed case", "ValidName", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidVariableName(tt.varName)
			if result != tt.expected {
				t.Errorf("isValidVariableName(%q) = %v, want %v", tt.varName, result, tt.expected)
			}
		})
	}
}

func TestIsSupportedVariableType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"string", "test", true},
		{"int", 42, true},
		{"int64", int64(42), true},
		{"float64", 3.14, true},
		{"bool", true, true},
		{"[]string", []string{"a", "b"}, true},
		{"[]int", []int{1, 2}, true},
		{"[]interface{} with strings", []interface{}{"a", "b"}, true},
		{"[]interface{} with mixed valid types", []interface{}{"a", 1, true}, true},
		{"[]interface{} with invalid type", []interface{}{"a", complex(1, 2)}, false},
		{"complex number", complex(1, 2), false},
		{"map", map[string]string{"key": "value"}, false},
		{"struct", struct{ Name string }{Name: "test"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSupportedVariableType(tt.value)
			if result != tt.expected {
				t.Errorf("isSupportedVariableType(%T) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}
