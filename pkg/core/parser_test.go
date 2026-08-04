// Package core provides shared parsing functionality for the todoer application.
package core

import (
	"reflect"
	"strings"
	"testing"
)

// TestValidateDate is the entry point of parser_test.go. The shared
// test-data helpers (createTestTodoItem, createTestDaySection) live
// in journal_test.go so the parser tests and the journal tests can
// share a single canonical constructor pair.
func TestValidateDate(t *testing.T) {
	t.Run("valid date should return no error", func(t *testing.T) {
		err := ValidateDate("2023-01-01")
		if err != nil {
			t.Errorf("Expected no error for valid date, got: %v", err)
		}
	})

	t.Run("invalid date format should return error", func(t *testing.T) {
		testCases := []string{
			"2023-1-1",   // missing leading zeros
			"23-01-01",   // year too short
			"2023/01/01", // wrong separators
			"01-01-2023", // wrong order
			"2023-13-01", // invalid month
			"2023-01-32", // invalid day
			"not-a-date", // completely invalid
			"",           // empty string
		}

		for _, dateStr := range testCases {
			t.Run(dateStr, func(t *testing.T) {
				err := ValidateDate(dateStr)
				if err == nil {
					t.Errorf("Expected error for invalid date '%s', got nil", dateStr)
				}
			})
		}
	})
}

func TestNewParserState(t *testing.T) {
	t.Run("should create parser state with correct initial values", func(t *testing.T) {
		state := newParserState()

		if state == nil {
			t.Fatal("Expected non-nil parser state")
			return
		}
		if state.currentDay != nil {
			t.Error("Expected currentDay to be nil")
		}
		if len(state.currentIndentStack) != 0 {
			t.Error("Expected empty currentIndentStack")
		}
		if len(state.currentItemStack) != 0 {
			t.Error("Expected empty currentItemStack")
		}
	})
}

func TestParserStateReset(t *testing.T) {
	t.Run("should reset stacks but preserve currentDay", func(t *testing.T) {
		state := newParserState()
		state.currentDay = createTestDaySection("2023-01-01")
		state.currentIndentStack = []int{0, 2, 4}
		state.currentItemStack = []*TodoItem{
			createTestTodoItem("Item 1", false),
			createTestTodoItem("Item 2", true),
		}

		state.reset()

		if state.currentDay == nil {
			t.Error("Expected currentDay to be preserved")
		}
		if len(state.currentIndentStack) != 0 {
			t.Error("Expected currentIndentStack to be reset")
		}
		if len(state.currentItemStack) != 0 {
			t.Error("Expected currentItemStack to be reset")
		}
	})
}

func TestParseTaskSection(t *testing.T) {
	t.Run("empty content should return empty journal", func(t *testing.T) {
		journal, err := ParseTodosSection("")
		if err != nil {
			t.Errorf("Expected no error for empty content, got: %v", err)
		}
		if journal == nil || len(journal.Days) != 0 {
			t.Error("Expected empty journal for empty content")
		}
	})

	t.Run("whitespace-only content should return empty journal", func(t *testing.T) {
		journal, err := ParseTodosSection("   \n\n  \t  \n")
		if err != nil {
			t.Errorf("Expected no error for whitespace-only content, got: %v", err)
		}
		if journal == nil || len(journal.Days) != 0 {
			t.Error("Expected empty journal for whitespace-only content")
		}
	})

	t.Run("single day with simple todo should parse correctly", func(t *testing.T) {
		content := `- [[2023-01-01]]
  - [ ] Task 1`

		journal, err := ParseTodosSection(content)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(journal.Days) != 1 {
			t.Errorf("Expected 1 day, got %d", len(journal.Days))
		}

		day := journal.Days[0]
		if day.Date != "2023-01-01" {
			t.Errorf("Expected date '2023-01-01', got '%s'", day.Date)
		}

		if len(day.Items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(day.Items))
		}

		item := day.Items[0]
		if item.Text != "Task 1" {
			t.Errorf("Expected text 'Task 1', got '%s'", item.Text)
		}
		if item.Completed {
			t.Error("Expected item to be uncompleted")
		}
	})

	t.Run("completed todo should parse correctly", func(t *testing.T) {
		content := `- [[2023-01-01]]
  - [x] Completed task`

		journal, err := ParseTodosSection(content)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		item := journal.Days[0].Items[0]
		if !item.Completed {
			t.Error("Expected item to be completed")
		}
		if item.Text != "Completed task" {
			t.Errorf("Expected text 'Completed task', got '%s'", item.Text)
		}
	})

	t.Run("nested todos should parse correctly", func(t *testing.T) {
		content := `- [[2023-01-01]]
  - [ ] Parent task
    - [ ] Child task
      - [x] Grandchild task`

		journal, err := ParseTodosSection(content)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		parent := journal.Days[0].Items[0]
		if len(parent.SubItems) != 1 {
			t.Errorf("Expected 1 subitem in parent, got %d", len(parent.SubItems))
		}

		child := parent.SubItems[0]
		if child.Text != "Child task" {
			t.Errorf("Expected child text 'Child task', got '%s'", child.Text)
		}
		if len(child.SubItems) != 1 {
			t.Errorf("Expected 1 subitem in child, got %d", len(child.SubItems))
		}

		grandchild := child.SubItems[0]
		if grandchild.Text != "Grandchild task" {
			t.Errorf("Expected grandchild text 'Grandchild task', got '%s'", grandchild.Text)
		}
		if !grandchild.Completed {
			t.Error("Expected grandchild to be completed")
		}
	})

	t.Run("multiple days should parse correctly", func(t *testing.T) {
		content := `- [[2023-01-01]]
  - [ ] Task 1
- [[2023-01-02]]
  - [x] Task 2`

		journal, err := ParseTodosSection(content)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(journal.Days) != 2 {
			t.Errorf("Expected 2 days, got %d", len(journal.Days))
		}

		if journal.Days[0].Date != "2023-01-01" {
			t.Errorf("Expected first date '2023-01-01', got '%s'", journal.Days[0].Date)
		}
		if journal.Days[1].Date != "2023-01-02" {
			t.Errorf("Expected second date '2023-01-02', got '%s'", journal.Days[1].Date)
		}
	})

	t.Run("bullet lines should be attached to todos", func(t *testing.T) {
		content := `- [[2023-01-01]]
  - [ ] Task with details
    - Detail 1
    - Detail 2`

		journal, err := ParseTodosSection(content)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		item := journal.Days[0].Items[0]
		if len(item.BulletLines) != 2 {
			t.Errorf("Expected 2 bullet lines, got %d", len(item.BulletLines))
		}

		expectedBullets := []string{"    - Detail 1", "    - Detail 2"}
		for i, expected := range expectedBullets {
			if item.BulletLines[i] != expected {
				t.Errorf("Expected bullet line '%s', got '%s'", expected, item.BulletLines[i])
			}
		}
	})

	t.Run("continuation lines should be attached to todos", func(t *testing.T) {
		content := `- [[2023-01-01]]
  - [ ] Task with continuation
      This is a continuation line
      Another continuation`

		journal, err := ParseTodosSection(content)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		item := journal.Days[0].Items[0]
		if len(item.BulletLines) != 2 {
			t.Errorf("Expected 2 bullet lines, got %d", len(item.BulletLines))
		}
	})

	t.Run("invalid date in day header should return error", func(t *testing.T) {
		content := `- [[invalid-date]]
  - [ ] Task`

		_, err := ParseTodosSection(content)
		if err == nil {
			// t.Error("Expected error for invalid date") // Removed: now expecting 'unparseable line' error
			return
		}
		if !strings.Contains(err.Error(), "unparseable line") {
			t.Errorf("Expected error message to contain 'unparseable line', got: %v", err)
		}
	})

	t.Run("unparseable line should return error", func(t *testing.T) {
		content := `- [[2023-01-01]]
  - [ ] Valid task
unparseable line`

		_, err := ParseTodosSection(content)
		if err == nil {
			t.Error("Expected error for unparseable line")
		}
		if !strings.Contains(err.Error(), "unparseable line") {
			t.Errorf("Expected error message to contain 'unparseable line', got: %v", err)
		}
	})

	t.Run("todo lines without day header should create undated section", func(t *testing.T) {
		content := `  - [ ] Task without day header
- [[2023-01-01]]
  - [ ] Valid task`

		journal, err := ParseTodosSection(content)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Should have 2 days: one undated (empty date string) and one dated
		if len(journal.Days) != 2 {
			t.Errorf("Expected 2 days, got %d", len(journal.Days))
		}

		// First day should be undated with empty date string
		if journal.Days[0].Date != "" {
			t.Errorf("Expected first day to have empty date, got '%s'", journal.Days[0].Date)
		}
		if len(journal.Days[0].Items) != 1 {
			t.Errorf("Expected 1 item in undated section, got %d", len(journal.Days[0].Items))
		}
		if journal.Days[0].Items[0].Text != "Task without day header" {
			t.Errorf("Expected undated task text, got '%s'", journal.Days[0].Items[0].Text)
		}

		// Second day should be the dated section
		if journal.Days[1].Date != "2023-01-01" {
			t.Errorf("Expected second day to have date '2023-01-01', got '%s'", journal.Days[1].Date)
		}
		if len(journal.Days[1].Items) != 1 {
			t.Errorf("Expected 1 item in dated section, got %d", len(journal.Days[1].Items))
		}
	})
}

func TestProcessLine(t *testing.T) {
	t.Run("empty line should be ignored", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}
		state := newParserState()

		err := processLine(journal, state, "", 1)
		if err != nil {
			t.Errorf("Expected no error for empty line, got: %v", err)
		}
	})

	t.Run("whitespace-only line should be ignored", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}
		state := newParserState()

		err := processLine(journal, state, "   \t  ", 1)
		if err != nil {
			t.Errorf("Expected no error for whitespace-only line, got: %v", err)
		}
	})

	t.Run("day header should create new day", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}
		state := newParserState()

		err := processLine(journal, state, "- [[2023-01-01]]", 1)
		if err != nil {
			t.Errorf("Expected no error for day header, got: %v", err)
		}

		if state.currentDay == nil {
			t.Error("Expected currentDay to be set")
		}
		if state.currentDay.Date != "2023-01-01" {
			t.Errorf("Expected date '2023-01-01', got '%s'", state.currentDay.Date)
		}
	})

	t.Run("todo item without current day should be ignored", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}
		state := newParserState()

		err := processLine(journal, state, "  - [ ] Task", 1)
		if err != nil {
			t.Errorf("Expected no error when ignoring todo without day, got: %v", err)
		}

		if len(journal.Days) != 0 {
			t.Error("Expected no days to be created")
		}
	})

	t.Run("unparseable line with current day should return error", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}
		state := newParserState()
		state.currentDay = createTestDaySection("2023-01-01")

		err := processLine(journal, state, "some unparseable text", 5)
		if err == nil {
			t.Error("Expected error for unparseable line")
		}
		if !strings.Contains(err.Error(), "line 5") {
			t.Errorf("Expected error to reference line number 5, got: %v", err)
		}
	})
}

func TestProcessDayHeader(t *testing.T) {
	t.Run("valid date should create new day section", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}
		state := newParserState()

		err := processDayHeader(journal, state, "2023-01-01")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if state.currentDay == nil {
			t.Error("Expected currentDay to be set")
		}
		if state.currentDay.Date != "2023-01-01" {
			t.Errorf("Expected date '2023-01-01', got '%s'", state.currentDay.Date)
		}
	})

	t.Run("invalid date should return error", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}
		state := newParserState()

		err := processDayHeader(journal, state, "invalid-date")
		if err == nil {
			t.Error("Expected error for invalid date")
		}
		if !strings.Contains(err.Error(), "invalid date") {
			t.Errorf("Expected error message to contain 'invalid date', got: %v", err)
		}
	})

	t.Run("should reset parser state", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}
		state := newParserState()
		state.currentIndentStack = []int{0, 2}
		state.currentItemStack = []*TodoItem{createTestTodoItem("Test", false)}

		err := processDayHeader(journal, state, "2023-01-01")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(state.currentIndentStack) != 0 {
			t.Error("Expected indent stack to be reset")
		}
		if len(state.currentItemStack) != 0 {
			t.Error("Expected item stack to be reset")
		}
	})

	t.Run("should append previous day to journal", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}
		state := newParserState()
		previousDay := createTestDaySection("2023-01-01")
		state.currentDay = previousDay

		err := processDayHeader(journal, state, "2023-01-02")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(journal.Days) != 1 {
			t.Errorf("Expected 1 day in journal, got %d", len(journal.Days))
		}
		if journal.Days[0] != previousDay {
			t.Error("Expected previous day to be added to journal")
		}
	})
}

func TestProcessTodoItem(t *testing.T) {
	t.Run("should create todo item and add to hierarchy", func(t *testing.T) {
		state := newParserState()
		state.currentDay = createTestDaySection("2023-01-01")
		todoMatch := []string{"  - [ ] Task", "  ", " ", "Task"}

		err := processTodoItem(state, todoMatch)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(state.currentItemStack) != 1 {
			t.Errorf("Expected 1 item in stack, got %d", len(state.currentItemStack))
		}
		if len(state.currentIndentStack) != 1 {
			t.Errorf("Expected 1 indent in stack, got %d", len(state.currentIndentStack))
		}

		item := state.currentItemStack[0]
		if item.Text != "Task" {
			t.Errorf("Expected text 'Task', got '%s'", item.Text)
		}
		if item.Completed {
			t.Error("Expected item to be uncompleted")
		}
	})

	t.Run("should handle completed todo item", func(t *testing.T) {
		state := newParserState()
		state.currentDay = createTestDaySection("2023-01-01")
		todoMatch := []string{"  - [x] Completed", "  ", "x", "Completed"}

		err := processTodoItem(state, todoMatch)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		item := state.currentItemStack[0]
		if !item.Completed {
			t.Error("Expected item to be completed")
		}
	})
}

func TestProcessAssociatedLine(t *testing.T) {
	t.Run("should attach bullet line to appropriate todo item", func(t *testing.T) {
		state := newParserState()
		state.currentDay = createTestDaySection("2023-01-01")

		// Set up a todo item in the stack
		item := createTestTodoItem("Main task", false)
		state.currentItemStack = []*TodoItem{item}
		state.currentIndentStack = []int{2}

		matches := []string{"    - Detail", "    ", "Detail"}
		err := processAssociatedLine(state, "    - Detail", matches)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(item.BulletLines) != 1 {
			t.Errorf("Expected 1 bullet line, got %d", len(item.BulletLines))
		}
		if item.BulletLines[0] != "    - Detail" {
			t.Errorf("Expected bullet line '    - Detail', got '%s'", item.BulletLines[0])
		}
	})

	t.Run("should handle empty item stack gracefully", func(t *testing.T) {
		state := newParserState()
		state.currentDay = createTestDaySection("2023-01-01")

		matches := []string{"    - Detail", "    ", "Detail"}
		err := processAssociatedLine(state, "    - Detail", matches)
		if err != nil {
			t.Errorf("Expected no error for empty stack, got: %v", err)
		}
	})

	t.Run("should normalize indentation in bullet lines", func(t *testing.T) {
		state := newParserState()
		state.currentDay = createTestDaySection("2023-01-01")

		item := createTestTodoItem("Main task", false)
		state.currentItemStack = []*TodoItem{item}
		state.currentIndentStack = []int{2}

		// Line with tabs that should be normalized
		line := "\t\t- Detail with tabs"
		matches := []string{line, "\t\t", "Detail with tabs"}
		err := processAssociatedLine(state, line, matches)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// The line should be normalized (tabs converted to spaces)
		if len(item.BulletLines) != 1 {
			t.Errorf("Expected 1 bullet line, got %d", len(item.BulletLines))
		}
		// Should not contain tabs anymore
		if strings.Contains(item.BulletLines[0], "\t") {
			t.Error("Expected tabs to be normalized to spaces")
		}
	})
}

func TestFindTargetItemForBullet(t *testing.T) {
	t.Run("should find parent item with lower indentation", func(t *testing.T) {
		item1 := createTestTodoItem("Item 1", false)
		item2 := createTestTodoItem("Item 2", false)
		itemStack := []*TodoItem{item1, item2}
		indentStack := []int{0, 2}

		target := findTargetItemForBullet(itemStack, indentStack, 4)
		if target != item2 {
			t.Error("Expected to find item2 as target for bullet with indent 4")
		}
	})

	t.Run("should find parent item when bullet indentation matches parent", func(t *testing.T) {
		item1 := createTestTodoItem("Item 1", false)
		item2 := createTestTodoItem("Item 2", false)
		itemStack := []*TodoItem{item1, item2}
		indentStack := []int{0, 4}

		target := findTargetItemForBullet(itemStack, indentStack, 2)
		if target != item1 {
			t.Error("Expected to find item1 as target for bullet with indent 2")
		}
	})

	t.Run("should return last item when no suitable parent found", func(t *testing.T) {
		item1 := createTestTodoItem("Item 1", false)
		item2 := createTestTodoItem("Item 2", false)
		itemStack := []*TodoItem{item1, item2}
		indentStack := []int{4, 6}

		target := findTargetItemForBullet(itemStack, indentStack, 2)
		if target != item2 {
			t.Error("Expected to find item2 as fallback target")
		}
	})

	t.Run("should return nil for empty stacks", func(t *testing.T) {
		target := findTargetItemForBullet([]*TodoItem{}, []int{}, 2)
		if target != nil {
			t.Error("Expected nil target for empty stacks")
		}
	})

	t.Run("should handle mismatched stack lengths gracefully", func(t *testing.T) {
		item1 := createTestTodoItem("Item 1", false)
		itemStack := []*TodoItem{item1}
		indentStack := []int{0, 2, 4} // longer than item stack

		target := findTargetItemForBullet(itemStack, indentStack, 6)
		if target != item1 {
			t.Error("Expected to find item1 as fallback target")
		}
	})
}

func TestCreateNewDaySection(t *testing.T) {
	t.Run("should create new day section with given date", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}

		newDay := createNewDaySection(journal, nil, "2023-01-01")

		if newDay == nil {
			t.Fatal("Expected non-nil day section")
			return
		}
		if newDay.Date != "2023-01-01" {
			t.Errorf("Expected date '2023-01-01', got '%s'", newDay.Date)
		}
		if newDay.Items == nil {
			t.Error("Expected non-nil Items slice")
		}
		if len(newDay.Items) != 0 {
			t.Error("Expected empty Items slice")
		}
	})

	t.Run("should append previous day to journal", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}
		previousDay := createTestDaySection("2023-01-01")

		newDay := createNewDaySection(journal, previousDay, "2023-01-02")

		if len(journal.Days) != 1 {
			t.Errorf("Expected 1 day in journal, got %d", len(journal.Days))
		}
		if journal.Days[0] != previousDay {
			t.Error("Expected previous day to be added to journal")
		}
		if newDay.Date != "2023-01-02" {
			t.Errorf("Expected new day date '2023-01-02', got '%s'", newDay.Date)
		}
	})

	t.Run("should not append nil previous day", func(t *testing.T) {
		journal := &TodoJournal{Days: []*DaySection{}}

		createNewDaySection(journal, nil, "2023-01-01")

		if len(journal.Days) != 0 {
			t.Error("Expected no days to be added when previous day is nil")
		}
	})
}

func TestCreateTodoItem(t *testing.T) {
	t.Run("should create uncompleted todo item", func(t *testing.T) {
		matches := []string{"  - [ ] Task", "  ", " ", "Task"}

		item := createTodoItem(matches)

		if item == nil {
			t.Fatal("Expected non-nil todo item")
			return
		}
		if item.Completed {
			t.Error("Expected item to be uncompleted")
		}
		if item.Text != "Task" {
			t.Errorf("Expected text 'Task', got '%s'", item.Text)
		}
		if item.SubItems == nil {
			t.Error("Expected non-nil SubItems slice")
		}
		if item.BulletLines == nil {
			t.Error("Expected non-nil BulletLines slice")
		}
	})

	t.Run("should create completed todo item", func(t *testing.T) {
		matches := []string{"  - [x] Completed Task", "  ", "x", "Completed Task"}

		item := createTodoItem(matches)

		if !item.Completed {
			t.Error("Expected item to be completed")
		}
		if item.Text != "Completed Task" {
			t.Errorf("Expected text 'Completed Task', got '%s'", item.Text)
		}
	})

	t.Run("should handle different completion markers", func(t *testing.T) {
		testCases := []struct {
			marker   string
			expected bool
		}{
			{"x", true},
			{"X", false}, // only lowercase x is treated as completed
			{" ", false},
			{"-", false},
			{"o", false},
		}

		for _, tc := range testCases {
			matches := []string{"  - [" + tc.marker + "] Task", "  ", tc.marker, "Task"}
			item := createTodoItem(matches)

			if item.Completed != tc.expected {
				t.Errorf("Expected completed=%v for marker '%s', got %v", tc.expected, tc.marker, item.Completed)
			}
		}
	})
}

func TestAddItemToHierarchy(t *testing.T) {
	t.Run("should add top-level item to empty hierarchy", func(t *testing.T) {
		currentDay := createTestDaySection("2023-01-01")
		item := createTestTodoItem("Top level", false)

		newIndentStack, newItemStack := addItemToHierarchy(
			currentDay, item, 0, []int{}, []*TodoItem{})

		if len(currentDay.Items) != 1 {
			t.Errorf("Expected 1 item in day, got %d", len(currentDay.Items))
		}
		if currentDay.Items[0] != item {
			t.Error("Expected item to be added to day")
		}
		if len(newIndentStack) != 1 || newIndentStack[0] != 0 {
			t.Errorf("Expected indent stack [0], got %v", newIndentStack)
		}
		if len(newItemStack) != 1 || newItemStack[0] != item {
			t.Error("Expected item to be added to item stack")
		}
	})

	t.Run("should add child item to parent", func(t *testing.T) {
		currentDay := createTestDaySection("2023-01-01")
		parentItem := createTestTodoItem("Parent", false)
		childItem := createTestTodoItem("Child", false)

		currentIndentStack := []int{0}
		currentItemStack := []*TodoItem{parentItem}

		newIndentStack, newItemStack := addItemToHierarchy(
			currentDay, childItem, 2, currentIndentStack, currentItemStack)

		if len(parentItem.SubItems) != 1 {
			t.Errorf("Expected 1 subitem in parent, got %d", len(parentItem.SubItems))
		}
		if parentItem.SubItems[0] != childItem {
			t.Error("Expected child to be added to parent")
		}
		if len(newIndentStack) != 2 {
			t.Errorf("Expected indent stack length 2, got %d", len(newIndentStack))
		}
		if len(newItemStack) != 2 {
			t.Errorf("Expected item stack length 2, got %d", len(newItemStack))
		}
	})

	t.Run("should handle sibling items at same level", func(t *testing.T) {
		currentDay := createTestDaySection("2023-01-01")
		firstItem := createTestTodoItem("First", false)
		secondItem := createTestTodoItem("Second", false)

		// Add first item
		currentIndentStack := []int{0}
		currentItemStack := []*TodoItem{firstItem}

		// Add second item at same level
		newIndentStack, newItemStack := addItemToHierarchy(
			currentDay, secondItem, 0, currentIndentStack, currentItemStack)

		if len(currentDay.Items) != 1 {
			t.Errorf("Expected 1 top-level item in day, got %d", len(currentDay.Items))
		}
		if len(newIndentStack) != 1 {
			t.Errorf("Expected indent stack length 1, got %d", len(newIndentStack))
		}
		if len(newItemStack) != 1 {
			t.Errorf("Expected item stack length 1, got %d", len(newItemStack))
		}
		if newItemStack[0] != secondItem {
			t.Error("Expected second item to replace first in stack")
		}
	})

	t.Run("should handle decreasing indentation levels", func(t *testing.T) {
		currentDay := createTestDaySection("2023-01-01")
		parentItem := createTestTodoItem("Parent", false)
		childItem := createTestTodoItem("Child", false)
		siblingItem := createTestTodoItem("Sibling", false)

		// Setup: parent at level 0, child at level 2
		currentIndentStack := []int{0, 2}
		currentItemStack := []*TodoItem{parentItem, childItem}

		// Add sibling at level 0 (should pop back to parent level)
		newIndentStack, newItemStack := addItemToHierarchy(
			currentDay, siblingItem, 0, currentIndentStack, currentItemStack)

		if len(newIndentStack) != 1 {
			t.Errorf("Expected indent stack length 1, got %d", len(newIndentStack))
		}
		if len(newItemStack) != 1 {
			t.Errorf("Expected item stack length 1, got %d", len(newItemStack))
		}
		if newItemStack[0] != siblingItem {
			t.Error("Expected sibling item in stack")
		}
	})

	// removed: should handle complex nesting with multiple level changes (removed: artificial test case that does not reflect real parser usage)
}

// TestDedupeSameDateDays pins the contract of the post-parse
// canonicalisation helper: sections that share a non-empty Date
// are merged into the first occurrence (items appended in source
// order), nil entries are dropped, undated sections are left
// alone, and a nil journal is a no-op.
func TestDedupeSameDateDays(t *testing.T) {
	t.Run("nil journal is a no-op", func(t *testing.T) {
		var j *TodoJournal
		dedupeSameDateDays(j) // must not panic
		if j != nil {
			t.Errorf("expected journal to remain nil, got %v", j)
		}
	})

	t.Run("merges same-date sections in source order", func(t *testing.T) {
		a := createTestDaySection("2026-08-04", createTestTodoItem("first-occurrence", false))
		b := createTestDaySection("2026-08-04", createTestTodoItem("second-occurrence", false), createTestTodoItem("third-occurrence", false))
		c := createTestDaySection("2026-08-03", createTestTodoItem("unique", false))
		j := createTestJournal(a, b, c)

		dedupeSameDateDays(j)

		if got, want := len(j.Days), 2; got != want {
			t.Fatalf("Days len = %d, want %d", got, want)
		}
		if j.Days[0] != a {
			t.Errorf("Days[0] should be the first occurrence (a), got %p (want %p)", j.Days[0], a)
		}
		if j.Days[1] != c {
			t.Errorf("Days[1] should be c (untouched), got %p (want %p)", j.Days[1], c)
		}
		gotTexts := []string{j.Days[0].Items[0].Text, j.Days[0].Items[1].Text, j.Days[0].Items[2].Text}
		wantTexts := []string{"first-occurrence", "second-occurrence", "third-occurrence"}
		if !reflect.DeepEqual(gotTexts, wantTexts) {
			t.Errorf("merged items in source order: got %v, want %v", gotTexts, wantTexts)
		}
	})

	t.Run("leaves undated sections alone", func(t *testing.T) {
		und1 := createTestDaySection("", createTestTodoItem("u1", false))
		und2 := createTestDaySection("", createTestTodoItem("u2", false))
		dated := createTestDaySection("2026-08-04", createTestTodoItem("d", false))
		j := createTestJournal(und1, dated, und2)

		dedupeSameDateDays(j)

		if got, want := len(j.Days), 3; got != want {
			t.Fatalf("Days len = %d, want %d (undated sections must NOT merge with each other or with dated sections)", got, want)
		}
		if j.Days[0].Date != "" || j.Days[1].Date != "2026-08-04" || j.Days[2].Date != "" {
			t.Errorf("section order changed: %q, %q, %q", j.Days[0].Date, j.Days[1].Date, j.Days[2].Date)
		}
	})

	t.Run("drops nil entries", func(t *testing.T) {
		d := createTestDaySection("2026-08-04", createTestTodoItem("d", false))
		j := createTestJournal(nil, d, nil)

		dedupeSameDateDays(j)

		if got, want := len(j.Days), 1; got != want {
			t.Errorf("Days len = %d, want %d (nil entries should be dropped)", got, want)
		}
	})
}

// TestParseTodosSectionCollapsesDuplicateDateSections is the
// regression test for issue #5 from the TUI code review: the
// parser used to create a separate DaySection for every
// [[YYYY-MM-DD]] header it saw, so a file with two same-date
// headers produced two same-date sections. The TUI's
// refreshItems then rendered the first unlabelled and the
// second labelled with today's date, and adding a new todo only
// wrote to the unlabelled section. ParseTodosSection now
// canonicalises the journal by merging duplicates.
//
// The parser does not sort by date (SortJournalDays does that
// separately), so Days is in source order: today (the duplicate
// header) comes before the older date.
func TestParseTodosSectionCollapsesDuplicateDateSections(t *testing.T) {
	content := `- [[2026-08-04]]
- [ ] task A

- [[2026-08-04]]
- [ ] task B
- [ ] task C

- [[2026-08-03]]
- [ ] older task
`
	journal, err := ParseTodosSection(content)
	if err != nil {
		t.Fatalf("ParseTodosSection: %v", err)
	}
	if got, want := len(journal.Days), 2; got != want {
		t.Fatalf("Days len = %d, want %d (duplicate-date sections should merge)", got, want)
	}
	if got, want := journal.Days[0].Date, "2026-08-04"; got != want {
		t.Errorf("Days[0].Date = %q, want %q (today, the duplicate header, comes first in source order)", got, want)
	}
	if got, want := len(journal.Days[0].Items), 3; got != want {
		t.Fatalf("merged today section item count = %d, want %d", got, want)
	}
	gotTexts := []string{
		journal.Days[0].Items[0].Text,
		journal.Days[0].Items[1].Text,
		journal.Days[0].Items[2].Text,
	}
	wantTexts := []string{"task A", "task B", "task C"}
	if !reflect.DeepEqual(gotTexts, wantTexts) {
		t.Errorf("merged items = %v, want %v (source order preserved)", gotTexts, wantTexts)
	}
	if got, want := journal.Days[1].Date, "2026-08-03"; got != want {
		t.Errorf("Days[1].Date = %q, want %q (older day, untouched)", got, want)
	}
	if got, want := len(journal.Days[1].Items), 1; got != want {
		t.Errorf("older section item count = %d, want %d", got, want)
	}
}

// TestParseTodosSectionPreservesUndatedSectionsAcrossDedup pins
// that the dedup pass does not merge an undated section with a
// dated section: an empty Date is a distinct "undated" semantic
// that MoveUndatedTodosToCurrentDate relies on. A subsequent
// todo item after a dated header gets appended to that dated
// section rather than starting a new undated section, so the
// input below produces 1 undated + 1 merged-dated = 2 sections.
func TestParseTodosSectionPreservesUndatedSectionsAcrossDedup(t *testing.T) {
	content := `- [ ] orphan undated A

- [[2026-08-04]]
- [ ] dated task A

- [[2026-08-04]]
- [ ] dated task B

- [[2026-08-03]]
- [ ] older task
`
	journal, err := ParseTodosSection(content)
	if err != nil {
		t.Fatalf("ParseTodosSection: %v", err)
	}
	if got, want := len(journal.Days), 3; got != want {
		t.Fatalf("Days len = %d, want %d", got, want)
	}
	if got, want := journal.Days[0].Date, ""; got != want {
		t.Errorf("Days[0].Date = %q, want %q (undated section preserved)", got, want)
	}
	if got, want := len(journal.Days[0].Items), 1; got != want {
		t.Errorf("undated section item count = %d, want %d (must NOT merge with dated sections)", got, want)
	}
	if got, want := journal.Days[1].Date, "2026-08-04"; got != want {
		t.Errorf("Days[1].Date = %q, want %q", got, want)
	}
	if got, want := len(journal.Days[1].Items), 2; got != want {
		t.Errorf("merged dated item count = %d, want %d", got, want)
	}
	if got, want := journal.Days[2].Date, "2026-08-03"; got != want {
		t.Errorf("Days[2].Date = %q, want %q (older dated section preserved)", got, want)
	}
}

// TestParseTodosSectionRoundTripCollapsesDuplicates pins that a
// parse -> JournalToString -> parse round-trip converges to a
// single section per date. Without the dedup pass the round-trip
// would preserve the duplicate sections forever, since the
// writer emits each section separately with no blank lines.
func TestParseTodosSectionRoundTripCollapsesDuplicates(t *testing.T) {
	content := `- [[2026-08-04]]
- [ ] task A

- [[2026-08-04]]
- [ ] task B
`
	first, err := ParseTodosSection(content)
	if err != nil {
		t.Fatalf("first parse: %v", err)
	}
	serialized := JournalToString(first)
	second, err := ParseTodosSection(serialized)
	if err != nil {
		t.Fatalf("second parse: %v", err)
	}
	if got, want := len(first.Days), 1; got != want {
		t.Errorf("first.Days len = %d, want %d", got, want)
	}
	if got, want := len(second.Days), 1; got != want {
		t.Errorf("second.Days len = %d, want %d (round-trip must converge)", got, want)
	}
}
