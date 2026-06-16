package core

import (
	"testing"
)

func TestMergeCarryover_NilInputs(t *testing.T) {
	// Both nil -> empty journal, sorted.
	got := MergeCarryover(nil, nil)
	if got == nil {
		t.Fatalf("expected a non-nil journal for nil inputs")
	}
	if len(got.Days) != 0 {
		t.Fatalf("expected no days for nil inputs, got %d", len(got.Days))
	}

	// Nil source -> target unchanged.
	target := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "x"}}},
	}}
	got = MergeCarryover(nil, target)
	if got != target {
		t.Fatalf("expected target passthrough on nil source")
	}
}

func TestMergeCarryover_AddsMissingDays(t *testing.T) {
	source := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-15", Items: []*TodoItem{{Text: "yesterday task"}}},
	}}
	target := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "today task"}}},
	}}

	got := MergeCarryover(source, target)

	if len(got.Days) != 2 {
		t.Fatalf("expected 2 days after merge, got %d", len(got.Days))
	}
	if got.Days[0].Date != "2026-03-15" {
		t.Fatalf("expected days sorted, first day = %s, got %s", "2026-03-15", got.Days[0].Date)
	}
	if got.Days[1].Date != "2026-03-16" {
		t.Fatalf("expected days sorted, second day = %s, got %s", "2026-03-16", got.Days[1].Date)
	}
	if len(got.Days[0].Items) != 1 || got.Days[0].Items[0].Text != "yesterday task" {
		t.Fatalf("expected yesterday task appended, got %+v", got.Days[0].Items)
	}
}

func TestMergeCarryover_DedupesSameDaySameText(t *testing.T) {
	source := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "shared task"}}},
	}}
	target := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "shared task"}}},
	}}

	got := MergeCarryover(source, target)
	if len(got.Days) != 1 {
		t.Fatalf("expected one day, got %d", len(got.Days))
	}
	if len(got.Days[0].Items) != 1 {
		t.Fatalf("expected dedup to leave one item, got %d", len(got.Days[0].Items))
	}
}

func TestMergeCarryover_KeepsBothForSameTextDifferentDays(t *testing.T) {
	source := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-15", Items: []*TodoItem{{Text: "shared task"}}},
	}}
	target := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "shared task"}}},
	}}

	got := MergeCarryover(source, target)
	if len(got.Days) != 2 {
		t.Fatalf("expected 2 days, got %d", len(got.Days))
	}
	for _, d := range got.Days {
		if len(d.Items) != 1 {
			t.Fatalf("expected one item per day, got %d for %s", len(d.Items), d.Date)
		}
	}
}

func TestMergeCarryover_StripsDateTagsWhenMatching(t *testing.T) {
	source := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "tagged task #2026-03-15"}}},
	}}
	target := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "tagged task"}}},
	}}

	got := MergeCarryover(source, target)
	if len(got.Days) != 1 {
		t.Fatalf("expected one day, got %d", len(got.Days))
	}
	if len(got.Days[0].Items) != 1 {
		t.Fatalf("expected dedup after date-tag strip, got %d items", len(got.Days[0].Items))
	}
}

func TestMergeCarryover_DeepCopiesItems(t *testing.T) {
	original := &TodoItem{Text: "carry me", Completed: false}
	source := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{original}},
	}}
	target := &TodoJournal{}

	got := MergeCarryover(source, target)
	if len(got.Days) != 1 || len(got.Days[0].Items) != 1 {
		t.Fatalf("expected one item in target, got %+v", got)
	}
	if got.Days[0].Items[0] == original {
		t.Fatalf("expected a deep copy, not a shared pointer")
	}
	if got.Days[0].Items[0].Text != "carry me" {
		t.Fatalf("expected text to be carried over, got %q", got.Days[0].Items[0].Text)
	}
}

func TestMergeCarryover_NestedSubtasks(t *testing.T) {
	// Subitems under a parent: when the parent is added, the
	// subitems are attached to the freshly-appended copy.
	sub := &TodoItem{Text: "subtask", SubItems: []*TodoItem{}}
	parent := &TodoItem{Text: "parent", SubItems: []*TodoItem{sub}}
	source := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{parent}},
	}}
	target := &TodoJournal{}

	got := MergeCarryover(source, target)
	if len(got.Days) != 1 || len(got.Days[0].Items) != 1 {
		t.Fatalf("expected one parent in target, got %+v", got)
	}
	parentCopy := got.Days[0].Items[0]
	if parentCopy == parent {
		t.Fatalf("expected parent deep copy")
	}
	if len(parentCopy.SubItems) != 1 {
		t.Fatalf("expected one subitem attached, got %d", len(parentCopy.SubItems))
	}
	if parentCopy.SubItems[0] == sub {
		t.Fatalf("expected subitem deep copy")
	}
}

func TestMergeCarryover_ReRunIsIdempotent(t *testing.T) {
	source := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{
			{Text: "first"},
			{Text: "second"},
		}},
	}}
	target := &TodoJournal{}

	once := MergeCarryover(source, target)
	twice := MergeCarryover(source, once)

	if len(twice.Days) != 1 || len(twice.Days[0].Items) != 2 {
		t.Fatalf("expected idempotent merge to keep 2 items, got %+v", twice)
	}

	// Mutating the source after merge must not affect the merged
	// result (deep-copy guarantee).
	source.Days[0].Items[0].Text = "mutated"
	if twice.Days[0].Items[0].Text != "first" {
		t.Fatalf("expected deep-copy isolation, got %q", twice.Days[0].Items[0].Text)
	}
}

func TestMergeCarryover_BulletLinesKept(t *testing.T) {
	source := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{
			{Text: "task", BulletLines: []string{"  * detail"}},
		}},
	}}
	target := &TodoJournal{}

	got := MergeCarryover(source, target)
	if len(got.Days) != 1 || len(got.Days[0].Items) != 1 {
		t.Fatalf("expected one item, got %+v", got)
	}
	if len(got.Days[0].Items[0].BulletLines) != 1 {
		t.Fatalf("expected bullet line preserved, got %+v", got.Days[0].Items[0].BulletLines)
	}
}

func TestMergeCarryover_TwoSourceWindows(t *testing.T) {
	// Two source windows: yesterday's unfinished, then today's
	// already-completed. Both should be merged into a target that
	// only had today's planned items.
	yesterdaySource := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-15", Items: []*TodoItem{{Text: "carry me"}}},
	}}
	todaySource := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "today planned"}}},
	}}
	target := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "today planned"}}},
	}}

	step1 := MergeCarryover(yesterdaySource, target)
	step2 := MergeCarryover(todaySource, step1)

	if len(step2.Days) != 2 {
		t.Fatalf("expected 2 days after two source windows, got %d", len(step2.Days))
	}
	if step2.Days[0].Date != "2026-03-15" || len(step2.Days[0].Items) != 1 {
		t.Fatalf("expected yesterday's item present, got %+v", step2.Days[0])
	}
	if step2.Days[1].Date != "2026-03-16" || len(step2.Days[1].Items) != 1 {
		t.Fatalf("expected today's item not duplicated, got %+v", step2.Days[1])
	}
}

func TestMergeCarryover_DaysSortedAcrossMerge(t *testing.T) {
	// Source days arrive out of order; result should be sorted.
	source := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-18", Items: []*TodoItem{{Text: "later"}}},
		{Date: "2026-03-14", Items: []*TodoItem{{Text: "earlier"}}},
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "middle"}}},
	}}
	target := &TodoJournal{}

	got := MergeCarryover(source, target)
	if len(got.Days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(got.Days))
	}
	want := []string{"2026-03-14", "2026-03-16", "2026-03-18"}
	for i, d := range got.Days {
		if d.Date != want[i] {
			t.Fatalf("expected day[%d] = %s, got %s", i, want[i], d.Date)
		}
	}
}

func TestSortDays_EmptyAndNil(_ *testing.T) {
	sortDays(nil)            // no panic
	sortDays(&TodoJournal{}) // no panic
	sortDays(&TodoJournal{Days: []*DaySection{
		{Date: "2026-03-16", Items: []*TodoItem{}},
		nil,
		{Date: "2026-03-15", Items: []*TodoItem{}},
	}})
}

func TestSortJournalDays_OutOfOrderInput(t *testing.T) {
	j := &TodoJournal{Days: []*DaySection{
		{Date: "2026-03-18", Items: []*TodoItem{{Text: "later"}}},
		{Date: "2026-03-14", Items: []*TodoItem{{Text: "earlier"}}},
		{Date: "2026-03-16", Items: []*TodoItem{{Text: "middle"}}},
	}}
	SortJournalDays(j)
	want := []string{"2026-03-14", "2026-03-16", "2026-03-18"}
	for i, day := range j.Days {
		if day.Date != want[i] {
			t.Fatalf("expected day[%d] = %s, got %s", i, want[i], day.Date)
		}
	}
}

func TestSortJournalDays_NilAndEmpty(_ *testing.T) {
	SortJournalDays(nil)                                    // no panic
	SortJournalDays(&TodoJournal{})                         // no panic
	SortJournalDays(&TodoJournal{Days: []*DaySection{nil}}) // no panic
}
