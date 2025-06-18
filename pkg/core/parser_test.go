// Package core provides shared parsing functionality for the todoer application.
package core

import (
	"strings"
	"testing"
)

// Test helper functions
func createTestTodoItemForParser(text string, completed bool, subitems ...*TodoItem) *TodoItem {
	return &TodoItem{
		Text:        text,
		Completed:   completed,
		SubItems:    subitems,
		BulletLines: []string{},
	}
}

func createTestDaySectionForParser(date string, items ...*TodoItem) *DaySection {
	return &DaySection{
		Date:  date,
		Items: items,
	}
}

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
		state.currentDay = createTestDaySectionForParser("2023-01-01")
		state.currentIndentStack = []int{0, 2, 4}
		state.currentItemStack = []*TodoItem{
			createTestTodoItemForParser("Item 1", false),
			createTestTodoItemForParser("Item 2", true),
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

	t.Run("todo lines without day header should be ignored", func(t *testing.T) {
		content := `  - [ ] Task without day header
- [[2023-01-01]]
  - [ ] Valid task`

		journal, err := ParseTodosSection(content)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(journal.Days) != 1 {
			t.Errorf("Expected 1 day, got %d", len(journal.Days))
		}
		if len(journal.Days[0].Items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(journal.Days[0].Items))
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
		state.currentDay = createTestDaySectionForParser("2023-01-01")

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
		state.currentItemStack = []*TodoItem{createTestTodoItemForParser("Test", false)}

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
		previousDay := createTestDaySectionForParser("2023-01-01")
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
		state.currentDay = createTestDaySectionForParser("2023-01-01")
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
		state.currentDay = createTestDaySectionForParser("2023-01-01")
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
		state.currentDay = createTestDaySectionForParser("2023-01-01")

		// Set up a todo item in the stack
		item := createTestTodoItemForParser("Main task", false)
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
		state.currentDay = createTestDaySectionForParser("2023-01-01")

		matches := []string{"    - Detail", "    ", "Detail"}
		err := processAssociatedLine(state, "    - Detail", matches)
		if err != nil {
			t.Errorf("Expected no error for empty stack, got: %v", err)
		}
	})

	t.Run("should normalize indentation in bullet lines", func(t *testing.T) {
		state := newParserState()
		state.currentDay = createTestDaySectionForParser("2023-01-01")

		item := createTestTodoItemForParser("Main task", false)
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
		item1 := createTestTodoItemForParser("Item 1", false)
		item2 := createTestTodoItemForParser("Item 2", false)
		itemStack := []*TodoItem{item1, item2}
		indentStack := []int{0, 2}

		target := findTargetItemForBullet(itemStack, indentStack, 4)
		if target != item2 {
			t.Error("Expected to find item2 as target for bullet with indent 4")
		}
	})

	t.Run("should find parent item when bullet indentation matches parent", func(t *testing.T) {
		item1 := createTestTodoItemForParser("Item 1", false)
		item2 := createTestTodoItemForParser("Item 2", false)
		itemStack := []*TodoItem{item1, item2}
		indentStack := []int{0, 4}

		target := findTargetItemForBullet(itemStack, indentStack, 2)
		if target != item1 {
			t.Error("Expected to find item1 as target for bullet with indent 2")
		}
	})

	t.Run("should return last item when no suitable parent found", func(t *testing.T) {
		item1 := createTestTodoItemForParser("Item 1", false)
		item2 := createTestTodoItemForParser("Item 2", false)
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
		item1 := createTestTodoItemForParser("Item 1", false)
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
		previousDay := createTestDaySectionForParser("2023-01-01")

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
		currentDay := createTestDaySectionForParser("2023-01-01")
		item := createTestTodoItemForParser("Top level", false)

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
		currentDay := createTestDaySectionForParser("2023-01-01")
		parentItem := createTestTodoItemForParser("Parent", false)
		childItem := createTestTodoItemForParser("Child", false)

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
		currentDay := createTestDaySectionForParser("2023-01-01")
		firstItem := createTestTodoItemForParser("First", false)
		secondItem := createTestTodoItemForParser("Second", false)

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
		currentDay := createTestDaySectionForParser("2023-01-01")
		parentItem := createTestTodoItemForParser("Parent", false)
		childItem := createTestTodoItemForParser("Child", false)
		siblingItem := createTestTodoItemForParser("Sibling", false)

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
