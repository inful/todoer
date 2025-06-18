// Package core provides shared journal manipulation functionality for the todoer application.
package core

import (
	"strings"
	"testing"
)

// Test data helpers
func createTestTodoItem(text string, completed bool, subitems ...*TodoItem) *TodoItem {
	return &TodoItem{
		Text:      text,
		Completed: completed,
		SubItems:  subitems,
	}
}

func createTestTodoItemWithBullets(text string, completed bool, bulletLines []string, subitems ...*TodoItem) *TodoItem {
	return &TodoItem{
		Text:        text,
		Completed:   completed,
		BulletLines: bulletLines,
		SubItems:    subitems,
	}
}

func createTestDaySection(date string, items ...*TodoItem) *DaySection {
	return &DaySection{
		Date:  date,
		Items: items,
	}
}

func createTestJournal(days ...*DaySection) *TodoJournal {
	return &TodoJournal{
		Days: days,
	}
}

func TestSplitJournal(t *testing.T) {
	t.Run("nil journal should return empty journals", func(t *testing.T) {
		completed, uncompleted := SplitJournal(nil)

		if completed == nil || len(completed.Days) != 0 {
			t.Error("Expected empty completed journal")
		}
		if uncompleted == nil || len(uncompleted.Days) != 0 {
			t.Error("Expected empty uncompleted journal")
		}
	})

	t.Run("empty journal should return empty journals", func(t *testing.T) {
		journal := createTestJournal()
		completed, uncompleted := SplitJournal(journal)

		if len(completed.Days) != 0 {
			t.Error("Expected empty completed journal")
		}
		if len(uncompleted.Days) != 0 {
			t.Error("Expected empty uncompleted journal")
		}
	})

	t.Run("journal with nil day should skip nil days", func(t *testing.T) {
		journal := &TodoJournal{
			Days: []*DaySection{nil, createTestDaySection("2023-01-01")},
		}
		completed, uncompleted := SplitJournal(journal)

		if len(completed.Days) != 0 {
			t.Error("Expected empty completed journal")
		}
		if len(uncompleted.Days) != 0 {
			t.Error("Expected empty uncompleted journal")
		}
	})

	t.Run("journal with only completed items", func(t *testing.T) {
		completedItem := createTestTodoItem("Task 1", true)
		completedSubitem := createTestTodoItem("Subtask 1", true)
		completedItemWithSub := createTestTodoItem("Task 2", true, completedSubitem)

		day := createTestDaySection("2023-01-01", completedItem, completedItemWithSub)
		journal := createTestJournal(day)

		completed, uncompleted := SplitJournal(journal)

		if len(completed.Days) != 1 {
			t.Error("Expected one day in completed journal")
		}
		if len(completed.Days[0].Items) != 2 {
			t.Error("Expected two items in completed journal")
		}
		if len(uncompleted.Days) != 0 {
			t.Error("Expected empty uncompleted journal")
		}

		// Verify deep copy
		if completed.Days[0].Items[0] == completedItem {
			t.Error("Expected deep copy, not reference")
		}
	})

	t.Run("journal with only uncompleted items", func(t *testing.T) {
		uncompletedItem := createTestTodoItem("Task 1", false)
		uncompletedSubitem := createTestTodoItem("Subtask 1", false)
		uncompletedItemWithSub := createTestTodoItem("Task 2", false, uncompletedSubitem)

		day := createTestDaySection("2023-01-01", uncompletedItem, uncompletedItemWithSub)
		journal := createTestJournal(day)

		completed, uncompleted := SplitJournal(journal)

		if len(completed.Days) != 0 {
			t.Error("Expected empty completed journal")
		}
		if len(uncompleted.Days) != 1 {
			t.Error("Expected one day in uncompleted journal")
		}
		if len(uncompleted.Days[0].Items) != 2 {
			t.Error("Expected two items in uncompleted journal")
		}
	})

	t.Run("journal with mixed completed and uncompleted items", func(t *testing.T) {
		completedItem := createTestTodoItem("Completed Task", true)
		uncompletedItem := createTestTodoItem("Uncompleted Task", false)

		// Item with completed parent but uncompleted subtask should go to uncompleted
		uncompletedSubitem := createTestTodoItem("Uncompleted Subtask", false)
		mixedItem := createTestTodoItem("Mixed Task", true, uncompletedSubitem)

		day := createTestDaySection("2023-01-01", completedItem, uncompletedItem, mixedItem)
		journal := createTestJournal(day)

		completed, uncompleted := SplitJournal(journal)

		if len(completed.Days) != 1 || len(completed.Days[0].Items) != 1 {
			t.Error("Expected one completed item")
		}
		if completed.Days[0].Items[0].Text != "Completed Task" {
			t.Error("Expected completed task in completed journal")
		}

		if len(uncompleted.Days) != 1 || len(uncompleted.Days[0].Items) != 2 {
			t.Error("Expected two uncompleted items")
		}

		// Verify the mixed item went to uncompleted due to uncompleted subtask
		foundMixedItem := false
		for _, item := range uncompleted.Days[0].Items {
			if item.Text == "Mixed Task" {
				foundMixedItem = true
				break
			}
		}
		if !foundMixedItem {
			t.Error("Expected mixed item in uncompleted journal due to uncompleted subtask")
		}
	})

	t.Run("journal with multiple days", func(t *testing.T) {
		// Day 1: only completed
		day1 := createTestDaySection("2023-01-01", createTestTodoItem("Task 1", true))

		// Day 2: only uncompleted
		day2 := createTestDaySection("2023-01-02", createTestTodoItem("Task 2", false))

		// Day 3: mixed
		day3 := createTestDaySection("2023-01-03",
			createTestTodoItem("Task 3a", true),
			createTestTodoItem("Task 3b", false))

		journal := createTestJournal(day1, day2, day3)
		completed, uncompleted := SplitJournal(journal)

		if len(completed.Days) != 2 {
			t.Errorf("Expected 2 days in completed journal, got %d", len(completed.Days))
		}
		if len(uncompleted.Days) != 2 {
			t.Errorf("Expected 2 days in uncompleted journal, got %d", len(uncompleted.Days))
		}
	})
}

func TestTagCompletedItems(t *testing.T) {
	t.Run("nil journal should not panic", func(t *testing.T) {
		TagCompletedItems(nil, "2023-01-01")
		// Should not panic
	})

	t.Run("empty date should not modify journal", func(t *testing.T) {
		item := createTestTodoItem("Task 1", true)
		originalText := item.Text
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedItems(journal, "")

		if item.Text != originalText {
			t.Error("Text should not be modified with empty date")
		}
	})

	t.Run("nil day should be skipped", func(t *testing.T) {
		journal := &TodoJournal{
			Days: []*DaySection{nil},
		}

		TagCompletedItems(journal, "2023-01-01")
		// Should not panic
	})

	t.Run("completed item without date tag should get tagged", func(t *testing.T) {
		item := createTestTodoItem("Task 1", true)
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedItems(journal, "2023-01-01")

		expected := "Task 1 #2023-01-01"
		if item.Text != expected {
			t.Errorf("Expected '%s', got '%s'", expected, item.Text)
		}
	})

	t.Run("completed item with existing date tag should not get another tag", func(t *testing.T) {
		item := createTestTodoItem("Task 1 #2023-01-01", true)
		originalText := item.Text
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedItems(journal, "2023-01-02")

		if item.Text != originalText {
			t.Error("Text should not be modified when date tag already exists")
		}
	})

	t.Run("uncompleted item should not get tagged", func(t *testing.T) {
		item := createTestTodoItem("Task 1", false)
		originalText := item.Text
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedItems(journal, "2023-01-01")

		if item.Text != originalText {
			t.Error("Uncompleted item should not get tagged")
		}
	})

	t.Run("nested completed subitems should get tagged", func(t *testing.T) {
		subitem := createTestTodoItem("Subtask", true)
		item := createTestTodoItem("Parent Task", true, subitem)
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedItems(journal, "2023-01-01")

		expectedParent := "Parent Task #2023-01-01"
		expectedSub := "Subtask #2023-01-01"

		if item.Text != expectedParent {
			t.Errorf("Expected parent text '%s', got '%s'", expectedParent, item.Text)
		}
		if subitem.Text != expectedSub {
			t.Errorf("Expected subitem text '%s', got '%s'", expectedSub, subitem.Text)
		}
	})

	t.Run("deeply nested completed items should get tagged", func(t *testing.T) {
		deepSubitem := createTestTodoItem("Deep Subtask", true)
		subitem := createTestTodoItem("Subtask", true, deepSubitem)
		item := createTestTodoItem("Parent Task", true, subitem)
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedItems(journal, "2023-01-01")

		if !strings.Contains(deepSubitem.Text, "#2023-01-01") {
			t.Error("Deep subitem should be tagged")
		}
	})
}

func TestTagCompletedSubitems(t *testing.T) {
	t.Run("nil journal should not panic", func(t *testing.T) {
		TagCompletedSubitems(nil, "2023-01-01")
		// Should not panic
	})

	t.Run("empty date should not modify journal", func(t *testing.T) {
		subitem := createTestTodoItem("Subtask", true)
		item := createTestTodoItem("Parent Task", false, subitem)
		originalText := subitem.Text
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedSubitems(journal, "")

		if subitem.Text != originalText {
			t.Error("Text should not be modified with empty date")
		}
	})

	t.Run("nil day should be skipped", func(t *testing.T) {
		journal := &TodoJournal{
			Days: []*DaySection{nil},
		}

		TagCompletedSubitems(journal, "2023-01-01")
		// Should not panic
	})

	t.Run("completed subitem should get tagged", func(t *testing.T) {
		subitem := createTestTodoItem("Subtask", true)
		item := createTestTodoItem("Parent Task", false, subitem)
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedSubitems(journal, "2023-01-01")

		expected := "Subtask #2023-01-01"
		if subitem.Text != expected {
			t.Errorf("Expected '%s', got '%s'", expected, subitem.Text)
		}
	})

	t.Run("parent item should not get tagged", func(t *testing.T) {
		subitem := createTestTodoItem("Subtask", true)
		item := createTestTodoItem("Parent Task", true)
		item.SubItems = []*TodoItem{subitem}
		originalParentText := item.Text
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedSubitems(journal, "2023-01-01")

		if item.Text != originalParentText {
			t.Error("Parent item should not get tagged by TagCompletedSubitems")
		}

		expected := "Subtask #2023-01-01"
		if subitem.Text != expected {
			t.Errorf("Expected subitem '%s', got '%s'", expected, subitem.Text)
		}
	})

	t.Run("uncompleted subitem should not get tagged", func(t *testing.T) {
		subitem := createTestTodoItem("Subtask", false)
		item := createTestTodoItem("Parent Task", false, subitem)
		originalText := subitem.Text
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedSubitems(journal, "2023-01-01")

		if subitem.Text != originalText {
			t.Error("Uncompleted subitem should not get tagged")
		}
	})

	t.Run("nested completed subitems should get tagged", func(t *testing.T) {
		deepSubitem := createTestTodoItem("Deep Subtask", true)
		subitem := createTestTodoItem("Subtask", true, deepSubitem)
		item := createTestTodoItem("Parent Task", false, subitem)
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		TagCompletedSubitems(journal, "2023-01-01")

		if !strings.Contains(subitem.Text, "#2023-01-01") {
			t.Error("Subitem should be tagged")
		}
		if !strings.Contains(deepSubitem.Text, "#2023-01-01") {
			t.Error("Deep subitem should be tagged")
		}
	})
}

func TestJournalToString(t *testing.T) {
	t.Run("nil journal should return empty string", func(t *testing.T) {
		result := JournalToString(nil)
		if result != "" {
			t.Error("Expected empty string for nil journal")
		}
	})

	t.Run("empty journal should return empty string", func(t *testing.T) {
		journal := createTestJournal()
		result := JournalToString(journal)
		if result != "" {
			t.Error("Expected empty string for empty journal")
		}
	})

	t.Run("journal with nil day should skip nil days", func(t *testing.T) {
		journal := &TodoJournal{
			Days: []*DaySection{nil},
		}
		result := JournalToString(journal)
		if result != "" {
			t.Error("Expected empty string when skipping nil days")
		}
	})

	t.Run("simple journal should format correctly", func(t *testing.T) {
		item := createTestTodoItem("Task 1", true)
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		result := JournalToString(journal)
		expected := "- [[2023-01-01]]\n  - [x] Task 1"

		if result != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
		}
	})

	t.Run("journal with uncompleted item should format correctly", func(t *testing.T) {
		item := createTestTodoItem("Task 1", false)
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		result := JournalToString(journal)
		expected := "- [[2023-01-01]]\n  - [ ] Task 1"

		if result != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
		}
	})

	t.Run("journal with bullet lines should include bullet lines", func(t *testing.T) {
		bulletLines := []string{"    * Additional info", "    * More details"}
		item := createTestTodoItemWithBullets("Task 1", true, bulletLines)
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		result := JournalToString(journal)
		expected := "- [[2023-01-01]]\n  - [x] Task 1\n    * Additional info\n    * More details"

		if result != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
		}
	})

	t.Run("journal with subitems should format with proper indentation", func(t *testing.T) {
		subitem := createTestTodoItem("Subtask", false)
		item := createTestTodoItem("Parent Task", true, subitem)
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		result := JournalToString(journal)
		expected := "- [[2023-01-01]]\n  - [x] Parent Task\n    - [ ] Subtask"

		if result != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
		}
	})

	t.Run("journal with multiple items should format all items", func(t *testing.T) {
		item1 := createTestTodoItem("Task 1", true)
		item2 := createTestTodoItem("Task 2", false)
		day := createTestDaySection("2023-01-01", item1, item2)
		journal := createTestJournal(day)

		result := JournalToString(journal)
		expected := "- [[2023-01-01]]\n  - [x] Task 1\n  - [ ] Task 2"

		if result != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
		}
	})

	t.Run("journal with multiple days should format all days", func(t *testing.T) {
		item1 := createTestTodoItem("Task 1", true)
		day1 := createTestDaySection("2023-01-01", item1)

		item2 := createTestTodoItem("Task 2", false)
		day2 := createTestDaySection("2023-01-02", item2)

		journal := createTestJournal(day1, day2)

		result := JournalToString(journal)
		expected := "- [[2023-01-01]]\n  - [x] Task 1\n- [[2023-01-02]]\n  - [ ] Task 2"

		if result != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
		}
	})

	t.Run("deeply nested items should format with correct indentation", func(t *testing.T) {
		deepSubitem := createTestTodoItem("Deep Subtask", true)
		subitem := createTestTodoItem("Subtask", false, deepSubitem)
		item := createTestTodoItem("Parent Task", true, subitem)
		day := createTestDaySection("2023-01-01", item)
		journal := createTestJournal(day)

		result := JournalToString(journal)
		expected := "- [[2023-01-01]]\n  - [x] Parent Task\n    - [ ] Subtask\n      - [x] Deep Subtask"

		if result != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
		}
	})
}

func TestWriteItemToString(t *testing.T) {
	t.Run("nil item should not write anything", func(t *testing.T) {
		var builder strings.Builder
		writeItemToString(&builder, nil, 1)

		if builder.String() != "" {
			t.Error("Expected empty string for nil item")
		}
	})

	t.Run("simple completed item should format correctly", func(t *testing.T) {
		var builder strings.Builder
		item := createTestTodoItem("Task 1", true)
		writeItemToString(&builder, item, 1)

		expected := "  - [x] Task 1\n"
		if builder.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, builder.String())
		}
	})

	t.Run("simple uncompleted item should format correctly", func(t *testing.T) {
		var builder strings.Builder
		item := createTestTodoItem("Task 1", false)
		writeItemToString(&builder, item, 1)

		expected := "  - [ ] Task 1\n"
		if builder.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, builder.String())
		}
	})

	t.Run("item with zero depth should have no indentation", func(t *testing.T) {
		var builder strings.Builder
		item := createTestTodoItem("Task 1", true)
		writeItemToString(&builder, item, 0)

		expected := "- [x] Task 1\n"
		if builder.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, builder.String())
		}
	})

	t.Run("item with multiple depth levels should indent correctly", func(t *testing.T) {
		var builder strings.Builder
		item := createTestTodoItem("Task 1", true)
		writeItemToString(&builder, item, 3)

		expected := "      - [x] Task 1\n"
		if builder.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, builder.String())
		}
	})

	t.Run("item with bullet lines should include bullet lines", func(t *testing.T) {
		var builder strings.Builder
		bulletLines := []string{"    * Detail 1", "    * Detail 2"}
		item := createTestTodoItemWithBullets("Task 1", true, bulletLines)
		writeItemToString(&builder, item, 1)

		expected := "  - [x] Task 1\n    * Detail 1\n    * Detail 2\n"
		if builder.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, builder.String())
		}
	})

	t.Run("item with subitems should write subitems recursively", func(t *testing.T) {
		var builder strings.Builder
		subitem := createTestTodoItem("Subtask", false)
		item := createTestTodoItem("Parent Task", true, subitem)
		writeItemToString(&builder, item, 1)

		expected := "  - [x] Parent Task\n    - [ ] Subtask\n"
		if builder.String() != expected {
			t.Errorf("Expected '%s', got '%s'", expected, builder.String())
		}
	})

	t.Run("item with complex nested structure should format correctly", func(t *testing.T) {
		var builder strings.Builder

		// Create nested structure with bullet lines
		deepSubitem := createTestTodoItem("Deep Task", true)
		bulletLines := []string{"      * Some detail"}
		subitem := createTestTodoItemWithBullets("Middle Task", false, bulletLines, deepSubitem)
		item := createTestTodoItem("Top Task", true, subitem)

		writeItemToString(&builder, item, 1)

		expected := "  - [x] Top Task\n    - [ ] Middle Task\n      * Some detail\n      - [x] Deep Task\n"
		if builder.String() != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, builder.String())
		}
	})
}

func TestTagCompletedItemsRecursive(t *testing.T) {
	t.Run("nil item should not panic", func(t *testing.T) {
		tagCompletedItemsRecursive(nil, "2023-01-01")
		// Should not panic
	})

	t.Run("completed item without date tag should get tagged", func(t *testing.T) {
		item := createTestTodoItem("Task 1", true)
		tagCompletedItemsRecursive(item, "2023-01-01")

		expected := "Task 1 #2023-01-01"
		if item.Text != expected {
			t.Errorf("Expected '%s', got '%s'", expected, item.Text)
		}
	})

	t.Run("completed item with existing date tag should not get another tag", func(t *testing.T) {
		item := createTestTodoItem("Task 1 #2023-01-01", true)
		originalText := item.Text
		tagCompletedItemsRecursive(item, "2023-01-02")

		if item.Text != originalText {
			t.Error("Text should not be modified when date tag already exists")
		}
	})

	t.Run("uncompleted item should not get tagged", func(t *testing.T) {
		item := createTestTodoItem("Task 1", false)
		originalText := item.Text
		tagCompletedItemsRecursive(item, "2023-01-01")

		if item.Text != originalText {
			t.Error("Uncompleted item should not get tagged")
		}
	})

	t.Run("completed item with completed subitems should tag both", func(t *testing.T) {
		subitem := createTestTodoItem("Subtask", true)
		item := createTestTodoItem("Parent Task", true, subitem)

		tagCompletedItemsRecursive(item, "2023-01-01")

		if !strings.Contains(item.Text, "#2023-01-01") {
			t.Error("Parent item should be tagged")
		}
		if !strings.Contains(subitem.Text, "#2023-01-01") {
			t.Error("Subitem should be tagged")
		}
	})

	t.Run("uncompleted item with completed subitems should only tag subitems", func(t *testing.T) {
		subitem := createTestTodoItem("Subtask", true)
		item := createTestTodoItem("Parent Task", false, subitem)
		originalParentText := item.Text

		tagCompletedItemsRecursive(item, "2023-01-01")

		if item.Text != originalParentText {
			t.Error("Uncompleted parent should not be tagged")
		}
		if !strings.Contains(subitem.Text, "#2023-01-01") {
			t.Error("Completed subitem should be tagged")
		}
	})
}

// Test constants from journal.go
func TestJournalConstants(t *testing.T) {
	t.Run("constants should have expected values", func(t *testing.T) {
		if DefaultBuilderCapacity != 1024 {
			t.Errorf("Expected DefaultBuilderCapacity to be 1024, got %d", DefaultBuilderCapacity)
		}
		if IndentSpaces != 2 {
			t.Errorf("Expected IndentSpaces to be 2, got %d", IndentSpaces)
		}
	})
}
