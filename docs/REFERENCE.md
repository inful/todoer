# Reference

This document provides a reference for the todoer CLI, journal format,
templates, and public APIs.

## CLI reference

### Global flags

These flags are shared by relevant commands and should be placed before
the subcommand.

- `--root-dir PATH` - override the journals root directory.
- `--template-file PATH` - override the template file for this run.
- `--print-path` - print only the resulting file path to standard
  output (where applicable).

### `todoer new`

Create a new daily journal file and carry over incomplete todos from the
most recent previous journal.

Synopsis:

```bash
todoer [--root-dir PATH] [--template-file PATH] [--print-path] new [--backup]
```

Options:

- `--backup` - preserve a `.bak` copy of the source journal before
  applying the carryover update. The default is to update the source
  in place without a backup; use this flag when you want a one-shot
  safety net (e.g. before a large manual edit).

If today's journal already exists, `new` still runs the carryover
pipeline and merges any missing items into the existing target
(`processJournal` with `merge=true`). Re-running `new` is
idempotent: items already in today stay, missing items from
yesterday are appended, and nothing is duplicated.

### `todoer add`

Add a todo item directly to today's journal.

If today's journal does not exist yet, todoer first creates it using the
same transfer behavior as `new`, then appends the todo item.

Synopsis:

```bash
todoer [--root-dir PATH] [--template-file PATH] [--print-path] add TODO_TEXT... [--backup]
```

Options:

- `TODO_TEXT...` - todo text to append (multi-word text can be passed
  without quotes).
- `--backup` - preserve a `.bak` copy of the source journal when
  today's journal has to be created for the first time (same
  semantics as `new --backup`).

### `todoer process`

Process a journal file into a new target file using a template.

This is the explicit, overwrite-style entry point. The daily flow
(`new`, `tui`, `add`) uses the same code path but with
`merge=true`, which splices the carryover into an existing
target instead of overwriting it.

Synopsis:

```bash
todoer [--template-file PATH] [--print-path] process SOURCE TARGET [--template-date YYYY-MM-DD]
```

Options:

- `SOURCE` - input journal file.
- `TARGET` - output file for uncompleted tasks.
- `--template-date YYYY-MM-DD` - logical date used for template variables.

### `todoer tui`

Open a minimal terminal UI for today's todo lifecycle.

If today's journal does not exist yet, todoer first creates it using the
same transfer behavior as `new`, then opens the UI.

Synopsis:

```bash
todoer [--root-dir PATH] [--template-file PATH] tui [--backup]
```

Options:

- `--backup` - preserve a `.bak` copy of the source journal when
  today's journal has to be created for the first time.

Core keys:

- `j`/`k` or arrow keys - move selection.
- `/` - filter/search todos by text.
- `c` - clear active filter.
- `space` - toggle completed/uncompleted.
- `a` - add a new todo.
- `d` - delete selected todo.
- `s` - save.
- `r` - reload from disk.
- `q` - save (if dirty) and quit.

Carryover behaviour: if today's section is missing or empty, the
TUI falls back to the most recent non-empty day and shows it in
read-only mode. Pressing `space`, `x`, or `d` on a carryover
item is blocked and shows `Cannot edit carryover items`. To
edit a carryover item, run `todoer new` first to bring it into
today.

When the user adds a new todo from the carryover view, the
new todo is appended to today's section but the view stays
on the carryover day (sticky display day). The new todo is
visible after `s` to save and `r` to reload, or immediately if
the user was already on today.

### `todoer preview`

Render a template with a sample todos section and optional custom
variables.

Synopsis:

```bash
todoer [--template-file PATH] preview [--date YYYY-MM-DD] \
  [--todos-file PATH | --todos-string STRING] [--custom-vars JSON]
```

Options:

- `--date YYYY-MM-DD` - date used for date-related template variables.
- `--todos-file PATH` - file containing a todos section.
- `--todos-string STRING` - inline todos section string.
- `--custom-vars JSON` - JSON object for custom variables.

## Journal format

Todoer expects markdown journals with a dedicated todos section. The
recommended structure is:

```markdown
---
title: "YYYY-MM-DD"
---

# Journal Title

## Todos

- [[YYYY-MM-DD]]
  - [ ] Uncompleted task
  - [x] Completed task
  - [ ] Another task
    - [ ] Subtask
    - [x] Completed subtask

## Other sections...
```

Rules:

- Todos are grouped under date headers of the form `- [[YYYY-MM-DD]]`.
- Incomplete tasks use `[ ]` and completed tasks use `[x]` checkboxes.
- Indentation determines hierarchy of tasks and subtasks.
- Only the configured todos section (default header `## Todos`) is
  processed. Other sections are preserved.
- A task is considered complete only if the task itself and all
  subtasks are marked as completed.

## Template variables

Todoer templates use Go `text/template` with a set of variables
available in the template context.

### Current date variables

- `{{.Date}}` - current date in `YYYY-MM-DD` format.
- `{{.DateShort}}` - short date format, for example `06/20/25`.
- `{{.DateLong}}` - long date format, for example `June 20, 2025`.
- `{{.Year}}` - year, for example `2025`.
- `{{.Month}}` - month number, for example `06`.
- `{{.MonthName}}` - month name, for example `June`.
- `{{.Day}}` - day of month, for example `20`.
- `{{.DayName}}` - day name, for example `Friday`.
- `{{.WeekNumber}}` - ISO week number, for example `25`.

### Previous date variables

Empty if there is no previous journal.

- `{{.PreviousDate}}` - previous journal date in `YYYY-MM-DD` format.
- `{{.PreviousDateShort}}` - short format.
- `{{.PreviousDateLong}}` - long format.
- `{{.PreviousYear}}` - previous year.
- `{{.PreviousMonth}}` - previous month number.
- `{{.PreviousMonthName}}` - previous month name.
- `{{.PreviousDay}}` - previous day of month.
- `{{.PreviousDayName}}` - previous day name.
- `{{.PreviousWeekNumber}}` - previous week number.

### Content variables

- `{{.TODOS}}` - uncompleted tasks section content.

### Todo statistics variables

Statistics are derived from the source journal when processing or
creating a new journal.

- `{{.TotalTodos}}` - number of incomplete todos being carried over.
- `{{.CompletedTodos}}` - number of completed todos in the source
  journal.
- `{{.UncompletedTodos}}` - number of uncompleted todos in the source
  journal.
- `{{.UncompletedTopLevelTodos}}` - number of uncompleted top-level
  todos.
- `{{.TodoDates}}` - list of unique dates that todos came from.
- `{{.OldestTodoDate}}` - date of the oldest incomplete todo, or empty
  if none.
- `{{.TodoDaysSpan}}` - number of days between the oldest incomplete
  todo and the current date.

### Custom variables

Custom variables are provided via configuration and exposed under the
`.Custom` field.

- `{{.Custom.VariableName}}` - value of a user-defined variable.

Constraints:

- Names must be valid Go template identifiers.
- Names must not conflict with built-in variable names.
- Supported value types: strings, integers, floats, booleans, and arrays
  of these types.

## Template functions

Todoer registers additional template functions to support date
operations, string handling, and other utilities.

### Date arithmetic

```go
{{addDays "2025-01-15" 5}}        // 2025-01-20
{{subDays "2025-01-15" 3}}        // 2025-01-12
{{addWeeks "2025-01-15" 2}}       // 2025-01-29
{{addMonths "2025-01-15" 1}}      // 2025-02-15
{{daysDiff "2025-01-15" "2025-01-20"}}  // 5
```

### Date formatting and queries

```go
{{formatDate .Date "Monday, January 02, 2006"}}
{{weekday .Date}}
{{isWeekend .Date}}
{{isMonday .Date}}
{{isTuesday .Date}}
{{isWednesday .Date}}
{{isThursday .Date}}
{{isFriday .Date}}
{{isSaturday .Date}}
{{isSunday .Date}}
```

### String functions

```go
{{upper "hello world"}}
{{lower "HELLO WORLD"}}
{{title "hello world"}}
{{trim "  spaced  "}}
{{replace "old" "new" "old text"}}
{{contains "hello world" "world"}}
{{hasPrefix "hello" "he"}}
{{hasSuffix "world" "ld"}}
{{split " " "hello world"}}
{{join ", " .TodoDates}}
{{repeat "abc" 3}}
{{len "hello"}}
```

### Utility functions

```go
{{default "fallback" .EmptyValue}}
{{empty .SomeValue}}
{{notEmpty .SomeValue}}
{{seq 1 5}}
{{dict "key1" "value1" "key2" "value2"}}
```

### Randomization

```go
{{shuffle "line1\nline2\nline3"}}
{{shuffleLines (split "\n" "a\nb\nc")}}
```

### Arithmetic

```go
{{add 5 3}}
{{sub 10 4}}
{{mul 6 7}}
{{div 15 3}}  // returns 0 for division by zero
```

## Template selection and defaults

Template resolution order:

1. Template specified via `--template-file` or `template_file` in
   configuration.
2. `$XDG_CONFIG_HOME/todoer/template.md` if present.
3. Built-in embedded default template.

If a template defines the todos section header but omits the
`{{.TODOS}}` placeholder, uncompleted tasks are inserted into that
section automatically.

## Library API (summary)

This section summarizes the main library entry points. See `LIBRARY.md`
for detailed examples.

### Generator package

Package import path:

```go
"git.luguber.info/inful/todoer/pkg/generator"
```

Key types and functions:

- `NewGeneratorWithOptions(templateContent, templateDate string, opts ...Option) (*Generator, error)`
- `NewGeneratorFromFileWithOptions(templateFile, templateDate string, opts ...Option) (*Generator, error)`
- `WithPreviousDate(previousDate string) Option`
- `WithCustomVariables(vars map[string]interface{}) Option`
- `WithFrontmatterDateKey(key string) Option`
- `WithTodosHeader(header string) Option`
- `(*Generator) Process(originalContent string) (*ProcessResult, error)`
- `(*Generator) ProcessFile(filename string) (*ProcessResult, error)`
- `(*Generator) WithOptions(opts ...Option) (*Generator, error)`

`ProcessResult` has the fields:

- `ModifiedOriginal io.Reader` - modified source content with completed
  tasks tagged.
- `NewFile io.Reader` - generated file content with uncompleted tasks.

### Core template API

Package import path:

```go
"git.luguber.info/inful/todoer/pkg/core"
```

Main entry points for template rendering and journal manipulation:

- `CreateFromTemplate(opts TemplateOptions) (string, error)`
- `ExtractFrontmatterMetadata(content string) (map[string]string, bool, error)`
- `UpsertFrontmatterMetadata(content string, updates map[string]string) (string, error)`
- `ParseTodosSection(content string) (*TodoJournal, error)`
- `JournalToString(journal *TodoJournal) string`
- `SplitJournal(journal *TodoJournal) (completed, uncompleted *TodoJournal)`
- `MergeCarryover(source, target *TodoJournal) *TodoJournal`
- `FindDaySection(journal *TodoJournal, date string) *DaySection`
- `FindOrCreateDaySection(journal *TodoJournal, date string) *DaySection`
- `RemoveItemFromDays(days []*DaySection, target *TodoItem) []*DaySection`
- `RemoveItemRecursive(items []*TodoItem, target *TodoItem) ([]*TodoItem, bool)`

`TemplateOptions` groups:

- `Content` - template content.
- `TodosContent` - todos section content.
- `CurrentDate` - date used for date variables.
- `PreviousDate` - optional previous journal date.
- `Journal` - optional journal structure for statistics.
- `CustomVars` - optional custom variables map.

`MergeCarryover` is the load-bearing piece of the ADR-0001
sync engine. It takes a source journal and a target journal,
appends any source items that are not already in the target
(matched by `(day, text)` after date-tag stripping), and
returns the merged target. Items are deep-copied so the
caller's journals are not aliased. Day sections are sorted
by date. The function is safe to call repeatedly: it is
idempotent by construction.

## Frontmatter metadata

The carryover metadata lives in the journal's YAML frontmatter
block. Todoer writes and reads these keys; the `core`
package provides `ExtractFrontmatterMetadata` and
`UpsertFrontmatterMetadata` helpers that preserve unrelated
keys, their order, and the document's line ending style
(LF or CRLF).

| Key | When | Purpose |
| --- | --- | --- |
| `todoer_carryover_to` | Future (ADR-0001 acceptance) | Target date the source carried into. |
| `todoer_carryover_updated_at` | Future (ADR-0001 acceptance) | Timestamp of the last carryover sync. |
| `todoer_source_fingerprint` | Only with `TODOER_FINGERPRINT=1` (spike) | SHA-256 of the source content at sync time. Used to detect external changes to the source between syncs. |
| `todoer_source_fingerprint_algo` | Only with `TODOER_FINGERPRINT=1` (spike) | Algorithm used for the fingerprint (`sha256`). |
| `todoer_source_fingerprint_at` | Only with `TODOER_FINGERPRINT=1` (spike) | Timestamp of the last fingerprint. |

The fingerprint spike is off by default. With the toggle on,
`processJournal` logs a `Fingerprint mismatch` message when
the recorded fingerprint does not match the current source
content. Per ADR-0001, the fingerprint is a hint, not a
gating check: a mismatch does not fail the sync, it just
forces a conservative re-merge (the default behaviour).

## Environment variables

- `TODOER_ROOT_DIR` - override the configured root directory.
- `TODOER_TEMPLATE_FILE` - override the configured template file.
- `XDG_CONFIG_HOME` - XDG-style config directory; falls back to
  `~/.config` when unset.
- `TODOER_FINGERPRINT` - set to `1` to enable the fingerprint spike.

