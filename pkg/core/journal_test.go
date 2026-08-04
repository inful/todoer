// Package core provides shared journal manipulation functionality for the todoer application.
package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test data helpers
//
// createTestTodoItem is the canonical constructor used by all
// journal_test.go and parser_test.go tests in this package. It
// matches the original parser_test.go helper: BulletLines is
// always a non-nil empty slice, never nil, so that the parser's
// append() and range calls behave identically regardless of how
// the test item was constructed.
func createTestTodoItem(text string, completed bool, subitems ...*TodoItem) *TodoItem {
	return &TodoItem{
		Text:        text,
		Completed:   completed,
		SubItems:    subitems,
		BulletLines: []BulletLine{},
	}
}

func createTestTodoItemWithBullets(text string, completed bool, bulletLines []BulletLine, subitems ...*TodoItem) *TodoItem {
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
	t.Run("nil journal should not panic", func(_ *testing.T) {
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

	t.Run("nil day should be skipped", func(_ *testing.T) {
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
	t.Run("nil journal should not panic", func(_ *testing.T) {
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

	t.Run("nil day should be skipped", func(_ *testing.T) {
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

func TestMoveUndatedTodosToCurrentDate(t *testing.T) {
	t.Run("moves both completed and uncompleted undated todos", func(t *testing.T) {
		undatedCompleted := createTestTodoItem("Done task", true)
		undatedUncompleted := createTestTodoItem("Carry task", false)

		journal := createTestJournal(
			createTestDaySection("", undatedCompleted, undatedUncompleted),
		)

		result := MoveUndatedTodosToCurrentDate(journal, "2026-03-16")

		if len(result.Days) != 1 {
			t.Fatalf("expected one day section, got %d", len(result.Days))
		}

		day := result.Days[0]
		if day.Date != "2026-03-16" {
			t.Fatalf("expected day date 2026-03-16, got %s", day.Date)
		}

		if len(day.Items) != 2 {
			t.Fatalf("expected two items in moved day, got %d", len(day.Items))
		}

		if !day.Items[0].Completed || day.Items[0].Text != "Done task" {
			t.Fatalf("expected completed undated todo to be preserved, got %+v", day.Items[0])
		}

		if day.Items[1].Completed || day.Items[1].Text != "Carry task" {
			t.Fatalf("expected uncompleted undated todo to be preserved, got %+v", day.Items[1])
		}
	})

	t.Run("appends undated todos to existing current-date section", func(t *testing.T) {
		existing := createTestTodoItem("Existing dated task", false)
		undatedCompleted := createTestTodoItem("Done task", true)

		journal := createTestJournal(
			createTestDaySection("2026-03-16", existing),
			createTestDaySection("", undatedCompleted),
		)

		result := MoveUndatedTodosToCurrentDate(journal, "2026-03-16")

		if len(result.Days) != 1 {
			t.Fatalf("expected one merged day section, got %d", len(result.Days))
		}

		items := result.Days[0].Items
		if len(items) != 2 {
			t.Fatalf("expected two merged items, got %d", len(items))
		}

		if items[0].Text != "Existing dated task" || items[1].Text != "Done task" {
			t.Fatalf("unexpected merged item order/content: %q, %q", items[0].Text, items[1].Text)
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
		bulletLines := []BulletLine{
			{Indent: 0, Text: "* Additional info"},
			{Indent: 0, Text: "* More details"},
		}
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
		bulletLines := []BulletLine{
			{Indent: 0, Text: "* Detail 1"},
			{Indent: 0, Text: "* Detail 2"},
		}
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
		// The bullet lives one IndentSpaces level beyond Middle Task;
		// the writer adds +1, so we store Indent=0 here. The bullet
		// ends up at the same depth as the Deep Task subitem.
		bulletLines := []BulletLine{{Indent: 0, Text: "* Some detail"}}
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
	t.Run("nil item should not panic", func(_ *testing.T) {
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

func TestFindDaySection(t *testing.T) {
	if got := FindDaySection(nil, "2026-03-16"); got != nil {
		t.Fatalf("expected nil journal to return nil, got %+v", got)
	}

	day := &DaySection{Date: "2026-03-16", Items: []*TodoItem{{Text: "x"}}}
	journal := &TodoJournal{Days: []*DaySection{day}}

	if got := FindDaySection(journal, "2099-01-01"); got != nil {
		t.Fatalf("expected no-match to return nil, got %+v", got)
	}
	if got := FindDaySection(journal, "2026-03-16"); got != day {
		t.Fatalf("expected match to return the day section, got %+v", got)
	}
	// Nil day in the slice is tolerated.
	journal2 := &TodoJournal{Days: []*DaySection{nil, day}}
	if got := FindDaySection(journal2, "2026-03-16"); got != day {
		t.Fatalf("expected match to skip nil entries, got %+v", got)
	}
}

func TestFindOrCreateDaySection(t *testing.T) {
	existing := &DaySection{Date: "2026-03-16", Items: []*TodoItem{{Text: "x"}}}
	journal := &TodoJournal{Days: []*DaySection{existing}}

	got := FindOrCreateDaySection(journal, "2026-03-16")
	if got != existing {
		t.Fatalf("expected to return the existing day section, got %+v", got)
	}

	got2 := FindOrCreateDaySection(journal, "2026-03-17")
	if got2 == nil || got2.Date != "2026-03-17" {
		t.Fatalf("expected a new day section for 2026-03-17, got %+v", got2)
	}
	if len(journal.Days) != 2 {
		t.Fatalf("expected journal to now have 2 day sections, got %d", len(journal.Days))
	}
}

func TestRemoveItemFromDays(t *testing.T) {
	target := &TodoItem{Text: "target"}
	other := &TodoItem{Text: "other"}
	sub := &TodoItem{Text: "subtarget"}
	day := &DaySection{
		Date: "2026-03-16",
		Items: []*TodoItem{
			other,
			{Text: "parent", SubItems: []*TodoItem{sub, {Text: "subother"}}},
		},
	}
	days := []*DaySection{day}

	// Remove a nested item.
	updated := RemoveItemFromDays(days, sub)
	if len(updated) != 1 {
		t.Fatalf("expected one day, got %d", len(updated))
	}
	if len(updated[0].Items) != 2 {
		t.Fatalf("expected parent + other to remain, got %d", len(updated[0].Items))
	}
	if len(updated[0].Items[1].SubItems) != 1 {
		t.Fatalf("expected one subitem left, got %d", len(updated[0].Items[1].SubItems))
	}

	// Remove a top-level item.
	updated2 := RemoveItemFromDays(updated, other)
	if len(updated2[0].Items) != 1 {
		t.Fatalf("expected one item left, got %d", len(updated2[0].Items))
	}

	// Remove a missing target is a no-op.
	updated3 := RemoveItemFromDays(updated2, target)
	if len(updated3[0].Items) != 1 {
		t.Fatalf("expected unchanged items, got %d", len(updated3[0].Items))
	}

	// Nil days and nil items in the slice are tolerated.
	updated4 := RemoveItemFromDays([]*DaySection{nil, day}, target)
	if len(updated4) != 2 {
		t.Fatalf("expected nil-day pass-through, got %d", len(updated4))
	}
}

func TestRemoveItemRecursive_NilSafety(t *testing.T) {
	items, removed := RemoveItemRecursive(nil, &TodoItem{Text: "x"})
	if removed {
		t.Fatalf("expected no removal on nil items")
	}
	if items != nil {
		t.Fatalf("expected nil items to be returned as-is")
	}

	items, removed = RemoveItemRecursive([]*TodoItem{nil}, &TodoItem{Text: "x"})
	if removed {
		t.Fatalf("expected no removal when only nil items are present")
	}
	_ = items
}

// TestJournalParseSerializeRoundTrip asserts that parsing a todos section and
// then serializing it produces a journal that, when parsed again, yields the
// same structure. Catches drift between the parser and the writer.
func TestJournalParseSerializeRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "simple",
			body: "- [[2025-05-12]]\n  - [ ] An unfinished todo\n  - [x] A completed todo\n- [[2025-05-11]]\n  - [ ] Unfinished\n    - [ ] Unfinished subtask\n  - [ ] Unfinished 2\n    - [x] Completed subtask\n    - [ ] Uncompleted subtask\n",
		},
		{
			name: "single_completed",
			body: "- [[2025-05-12]]\n  - [x] Done\n",
		},
		{
			name: "single_undated",
			body: "- [ ] No date here\n- [x] Done without date\n",
		},
		{
			name: "bullet_entries",
			body: "- [[2025-05-12]]\n  - [x] Task with bullets\n    * detail one\n    * detail two\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first, err := ParseTodosSection(tc.body)
			if err != nil {
				t.Fatalf("first parse: %v", err)
			}
			serialized := JournalToString(first)
			second, err := ParseTodosSection(serialized)
			if err != nil {
				t.Fatalf("second parse: %v\nserialized:\n%s", err, serialized)
			}

			if !journalEqual(first, second) {
				t.Fatalf("round-trip mismatch\nfirst:\n%+v\nsecond:\n%+v\nserialized:\n%s", first, second, serialized)
			}
		})
	}
}

func journalEqual(a, b *TodoJournal) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	if len(a.Days) != len(b.Days) {
		return false
	}
	for i := range a.Days {
		if !dayEqual(a.Days[i], b.Days[i]) {
			return false
		}
	}
	return true
}

func dayEqual(a, b *DaySection) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	if a.Date != b.Date {
		return false
	}
	if len(a.Items) != len(b.Items) {
		return false
	}
	for i := range a.Items {
		if !itemEqual(a.Items[i], b.Items[i]) {
			return false
		}
	}
	return true
}

func itemEqual(a, b *TodoItem) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	if a.Completed != b.Completed || a.Text != b.Text {
		return false
	}
	if len(a.BulletLines) != len(b.BulletLines) {
		return false
	}
	for i := range a.BulletLines {
		if a.BulletLines[i] != b.BulletLines[i] {
			return false
		}
	}
	if len(a.SubItems) != len(b.SubItems) {
		return false
	}
	for i := range a.SubItems {
		if !itemEqual(a.SubItems[i], b.SubItems[i]) {
			return false
		}
	}
	return true
}

// TestJournalRoundTripFromTestdata walks the integration testdata fixtures
// and asserts that the same structure survives a parse -> serialize -> parse
// round trip. This catches drift between the parser and the writer that
// would silently mangle user data on save.
func TestJournalRoundTripFromTestdata(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "tests", "testdata"))
	if err != nil {
		t.Skipf("cannot resolve testdata root: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("cannot read testdata root: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		inputPath := filepath.Join(root, e.Name(), "input.md")
		body, err := os.ReadFile(inputPath)
		if err != nil {
			t.Logf("skip %s: %v", e.Name(), err)
			continue
		}
		_, todosSection, _, err := ExtractTodosSectionWithHeader(string(body), TodosHeader)
		if err != nil {
			t.Logf("skip %s: extract failed: %v", e.Name(), err)
			continue
		}
		if strings.TrimSpace(todosSection) == "" {
			continue
		}

		t.Run(e.Name(), func(t *testing.T) {
			first, err := ParseTodosSection(todosSection)
			if err != nil {
				t.Fatalf("first parse: %v", err)
			}
			serialized := JournalToString(first)
			second, err := ParseTodosSection(serialized)
			if err != nil {
				t.Fatalf("second parse failed for %s\nserialized:\n%s", e.Name(), serialized)
			}
			if !journalEqual(first, second) {
				t.Fatalf("round-trip mismatch for %s\nfirst:\n%+v\nsecond:\n%+v\nserialized:\n%s", e.Name(), first, second, serialized)
			}
		})
	}
}

// TestIssue13_BulletLineIndentPreservedAcrossCarryover is the
// regression test for issue #13: a todo with an associated bullet
// line or continuation was losing its indentation when the parent
// was moved into today's section (which adds one IndentSpaces level
// to the parent's depth). The fix stores each BulletLine with its
// indent relative to the parent so the writer can re-indent it
// when the parent's depth changes.
func TestIssue13_BulletLineIndentPreservedAcrossCarryover(t *testing.T) {
	t.Run("bullet under top-level todo stays nested after carryover", func(t *testing.T) {
		input := `- [[2026-03-17]]
- [ ] issue
  - subitem
`
		completed, uncompleted, _, err := ProcessTodosSectionWithStats(input, "2026-03-17", "2026-03-18")
		if err != nil {
			t.Fatalf("ProcessTodosSectionWithStats: %v", err)
		}
		_ = completed // empty completed side is fine
		want := "- [[2026-03-17]]\n  - [ ] issue\n    - subitem"
		if uncompleted != want {
			t.Errorf("bullet indent lost during carryover.\nwant:\n%s\ngot:\n%s", want, uncompleted)
		}
	})

	t.Run("continuation lines stay nested after carryover", func(t *testing.T) {
		input := `- [[2026-03-17]]
- [ ] issue
  continuation text
  more continuation
`
		completed, uncompleted, _, err := ProcessTodosSectionWithStats(input, "2026-03-17", "2026-03-18")
		if err != nil {
			t.Fatalf("ProcessTodosSectionWithStats: %v", err)
		}
		_ = completed
		want := "- [[2026-03-17]]\n  - [ ] issue\n    continuation text\n    more continuation"
		if uncompleted != want {
			t.Errorf("continuation indent lost during carryover.\nwant:\n%s\ngot:\n%s", want, uncompleted)
		}
	})

	t.Run("nested bullets under nested bullets preserve relative depth", func(t *testing.T) {
		// A bullet that is one IndentSpaces level deeper than the
		// parent bullet stays one level deeper than the parent in the
		// output, even though the parent's absolute depth changes.
		input := `- [[2026-03-17]]
- [ ] parent
  - bullet level 1
    - bullet level 2
`
		completed, uncompleted, _, err := ProcessTodosSectionWithStats(input, "2026-03-17", "2026-03-18")
		if err != nil {
			t.Fatalf("ProcessTodosSectionWithStats: %v", err)
		}
		_ = completed
		want := "- [[2026-03-17]]\n  - [ ] parent\n    - bullet level 1\n      - bullet level 2"
		if uncompleted != want {
			t.Errorf("nested bullet relative depth lost.\nwant:\n%s\ngot:\n%s", want, uncompleted)
		}
	})

	t.Run("bullet under subitem is re-indented when subitem moves", func(t *testing.T) {
		// Subitems are written with depth+1 = 3 levels. A bullet on
		// that subitem should also be at one IndentSpaces level beyond
		// the subitem (i.e., level 4).
		item := createTestTodoItem("Top Task", false)
		subBullet := []BulletLine{{Indent: 0, Text: "* note under subitem"}}
		subitem := createTestTodoItemWithBullets("Middle", false, subBullet)
		item.SubItems = append(item.SubItems, subitem)

		day := createTestDaySection("2026-03-18", item)
		journal := createTestJournal(day)
		got := JournalToString(journal)
		want := "- [[2026-03-18]]\n  - [ ] Top Task\n    - [ ] Middle\n      * note under subitem"
		if got != want {
			t.Errorf("bullet-under-subitem indent wrong.\nwant:\n%s\ngot:\n%s", want, got)
		}
	})

	t.Run("parse -> write -> parse round-trips bullet indent", func(t *testing.T) {
		input := `- [[2026-03-17]]
  - [ ] issue
    - subitem
      - sub-subitem
`
		first, err := ParseTodosSection(input)
		if err != nil {
			t.Fatalf("first parse: %v", err)
		}
		serialized := JournalToString(first)
		second, err := ParseTodosSection(serialized)
		if err != nil {
			t.Fatalf("second parse: %v\nserialized:\n%s", err, serialized)
		}
		if !journalEqual(first, second) {
			t.Errorf("bullet indent not stable across round-trip.\nfirst:\n%+v\nsecond:\n%+v\nserialized:\n%s", first, second, serialized)
		}
	})
}
