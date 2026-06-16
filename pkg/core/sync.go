package core

import (
	"sort"
	"strings"
)

// carryoverItemKey returns a deterministic matching key for a todo item
// used by the carryover merge to detect duplicates. The key is the
// item's text with date tags stripped and surrounding whitespace
// removed, prefixed with the item's day date so items in different days
// do not collide.
//
// The key is intentionally simple: it trades a small false-positive
// risk (e.g. two genuinely different tasks with the same text on the
// same day) for deterministic, easily-debuggable behaviour. That is
// acceptable per ADR-0001's 'markdown-only task matching' decision.
func carryoverItemKey(day string, item *TodoItem) string {
	text := strings.TrimSpace(item.Text)
	text = DateTagRegex.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)
	return day + "\x00" + text
}

// collectCarryoverKeys returns the set of carryover keys present in a
// journal, walked recursively so nested subitems are also covered.
func collectCarryoverKeys(journal *TodoJournal) map[string]bool {
	keys := make(map[string]bool)
	if journal == nil {
		return keys
	}
	for _, day := range journal.Days {
		if day == nil {
			continue
		}
		collectKeysInto(day.Items, day.Date, keys)
	}
	return keys
}

func collectKeysInto(items []*TodoItem, day string, keys map[string]bool) {
	for _, item := range items {
		if item == nil {
			continue
		}
		keys[carryoverItemKey(day, item)] = true
		// Subitems are deduped by their own (day, text) key, not
		// by a parent-prefixed key. That is a known limit: two
		// parents in the target with the same subitem text will
		// collide. The ADR accepts this trade-off for
		// deterministic, debuggable behaviour.
		for _, sub := range item.SubItems {
			if sub == nil {
				continue
			}
			keys[carryoverItemKey(day, sub)] = true
			if len(sub.SubItems) > 0 {
				collectKeysInto(sub.SubItems, day, keys)
			}
		}
	}
}

// MergeCarryover returns target with the source's todos appended into
// the matching day, skipping any item whose key is already present in
// target. The target is modified in place and also returned for
// convenience; day sections are sorted by date so the resulting
// journal is monotonic.
//
// Source items are deep-copied before being appended, so the caller's
// journal is not aliased into the target.
//
// Nil inputs are tolerated: a nil source returns the target unchanged
// (or an empty journal if target is also nil); a nil target is treated
// as an empty journal.
func MergeCarryover(source, target *TodoJournal) *TodoJournal {
	if target == nil {
		target = &TodoJournal{Days: []*DaySection{}}
	}
	if source == nil {
		sortDays(target)
		return target
	}

	existing := collectCarryoverKeys(target)

	for _, day := range source.Days {
		if day == nil {
			continue
		}
		mergeDayInto(day, target, existing)
	}

	sortDays(target)
	return target
}

// mergeDayInto copies the items from srcDay that are not already
// present in target into the matching target day (or a freshly
// created day if the date does not yet exist). Returns the target
// day that received the new items, or nil if nothing was added.
//
// Subitems come along with their parent via DeepCopyItem. We do not
// dedup subitems across different parents; that would require
// walking the target's tree on every insert and is out of scope for
// the ADR's 'markdown-only task matching' heuristic.
func mergeDayInto(srcDay *DaySection, target *TodoJournal, existing map[string]bool) *DaySection {
	var targetDay *DaySection
	for _, d := range target.Days {
		if d != nil && d.Date == srcDay.Date {
			targetDay = d
			break
		}
	}

	var added bool
	for _, item := range srcDay.Items {
		if item == nil {
			continue
		}
		if existing[carryoverItemKey(srcDay.Date, item)] {
			continue
		}
		cloned := DeepCopyItem(item)
		if targetDay == nil {
			targetDay = &DaySection{Date: srcDay.Date, Items: []*TodoItem{}}
			target.Days = append(target.Days, targetDay)
		}
		targetDay.Items = append(targetDay.Items, cloned)
		existing[carryoverItemKey(srcDay.Date, item)] = true
		added = true
	}

	if !added {
		return nil
	}
	return targetDay
}

// SortJournalDays sorts the journal's day sections by their date in
// ascending order. Days with empty date strings sort last. The
// function is safe on a nil journal and on a journal with nil day
// entries. The sort is in place.
func SortJournalDays(journal *TodoJournal) {
	if journal == nil {
		return
	}
	sort.SliceStable(journal.Days, func(i, j int) bool {
		if journal.Days[i] == nil {
			return false
		}
		if journal.Days[j] == nil {
			return true
		}
		if journal.Days[i].Date == "" {
			return false
		}
		if journal.Days[j].Date == "" {
			return true
		}
		return journal.Days[i].Date < journal.Days[j].Date
	})
}

// sortDays is a private alias used by MergeCarryover so the
// sort helper is callable from the same package without
// touching the exported name.
func sortDays(journal *TodoJournal) {
	SortJournalDays(journal)
}
