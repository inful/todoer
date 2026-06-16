# Background

This document describes the concepts and design of todoer.

## Processing model

Todoer operates on markdown journal files that contain a dedicated
todos section. The main steps are:

1. Read the journal file.
2. Extract the frontmatter date using a configurable key.
3. Locate the todos section header (default `## Todos`, configurable).
4. Parse the todos section into a structured representation.
5. Classify tasks as completed or uncompleted.
6. Tag completed tasks with the completion date.
7. Generate a new file containing uncompleted tasks using a template.
8. If the target file already exists (e.g. created by hand or by an
   earlier `new` / `add` / `tui` run), splice the new uncompleted
   items into the target's existing todos via `MergeCarryover`
   instead of overwriting. Re-running the command is idempotent.
9. Write changes back to disk and create backups where appropriate.

The parser maintains the hierarchical structure of tasks and subtasks
based on indentation and bullets.

## Completion rules

Todoer considers a task completed only if:

- The task itself is marked as completed with `[x]`.
- All subtasks and nested subtasks are also marked as completed.

This rule applies when deciding whether a task remains in the original
file or is carried over to the new file.

## Section handling

Only the configured todos section is processed.

- The default section header is `## Todos`.
- The header can be changed via configuration or options.
- Content outside this section is preserved unchanged.
- The file's line ending style (LF or CRLF) is preserved by the
  frontmatter I/O helpers; editing metadata does not silently
  rewrite line endings.

If no todos section is found, processing continues without modifying the
file structure outside the todos area.

## Carryover and idempotency

The daily flow (`new`, `tui`, `add`) all share the same
`processJournal` path with `merge=true`. The flow is:

- If today's journal does not exist, the source (yesterday's journal
  or an empty stub) is processed and the target is created.
- If today's journal already exists, the source's uncompleted items
  are merged into the target's existing todos via
  `core.MergeCarryover`. Items already present (matched by `(day,
  text)` after date-tag stripping) are not duplicated. Day sections
  are sorted by date.

This means re-running `todoer new` is always safe: the second run
either creates the file (if it was missing) or merges into the
existing one (if it was already there). The matching is intentionally
simple; per ADR-0001, the markdown-only heuristic trades a small
risk of false-positive merges for deterministic, debuggable
behaviour.

## TUI carryover view

When the TUI opens on a journal where today's section is missing or
empty, it falls back to the most recent non-empty day. That fallback
view is read-only: pressing `space`, `x`, or `d` is blocked with
the status `Cannot edit carryover items`. To edit a carryover
item, run `todoer new` first to bring it into today.

The display day is sticky: if the user adds a new todo while in
the carryover view, the new todo is appended to today's section
but the view stays on the carryover day. The new todo is visible
after `s` to save and `r` to reload, or immediately if the user
was already on today.

## Date handling

Todoer uses dates in several ways:

- Frontmatter date: extracted from a configurable field (for example
  `title` or `date`).
- Template date: the logical date used when rendering templates.
- Previous date: the date of the journal that serves as the source for
  carried-over tasks when creating a new journal.

Date values are validated to ensure they conform to the expected
`YYYY-MM-DD` format.

## Todo statistics

When processing a journal, todoer can compute statistics about the
todos, including:

- Total number of incomplete todos.
- Number of completed and uncompleted todos.
- Number of uncompleted top-level todos.
- Dates associated with todos.
- Oldest incomplete todo date and span in days.

These values are exposed to templates as variables for reporting.

## Template system

The template system is based on Go `text/template` with a set of custom
variables and functions.

Key aspects:

- Templates can use date-related variables for the current and previous
  journal.
- The `.TODOS` placeholder controls where uncompleted tasks are
  inserted.
- Additional functions support date arithmetic, formatting, string
  handling, and simple arithmetic.
- Custom variables from configuration are available under `.Custom`.

## Testing

Todoer includes automated tests at several levels:

- CLI tests exercising the compiled binary and command-line behavior.
- Core package tests covering parsing, statistics, and template
  functions.
- Generator tests validating the library interface.
- Integration tests using sample input and expected output files.

For detailed instructions on running tests, see `TESTING.md` and
`TEST_SUMMARY.md`.

