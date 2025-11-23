# Reference

This document provides a reference for the todoer CLI, journal format,
templates, and public APIs.

## CLI reference

### `todoer new`

Create a new daily journal file and carry over incomplete todos from the
most recent previous journal.

Synopsis:

```bash
todoer new [--root-dir PATH] [--template-file PATH] [--print-path]
```

Options:

- `--root-dir PATH` - override the journals root directory.
- `--template-file PATH` - override the template file for this run.
- `--print-path` - print the created file path to standard output.

### `todoer process`

Process a journal file into a new target file using a template.

Synopsis:

```bash
todoer process SOURCE TARGET [--template-file PATH] [--template-date YYYY-MM-DD] [--print-path]
```

Options:

- `SOURCE` - input journal file.
- `TARGET` - output file for uncompleted tasks.
- `--template-file PATH` - template file used for the target file.
- `--template-date YYYY-MM-DD` - logical date used for template variables.
- `--print-path` - print the target file path to standard output.

### `todoer preview`

Render a template with a sample todos section and optional custom
variables.

Synopsis:

```bash
todoer preview [--template-file PATH] [--date YYYY-MM-DD] \
  [--todos-file PATH | --todos-string STRING] [--custom-vars JSON]
```

Options:

- `--template-file PATH` - template file to render.
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

Main entry point for template rendering:

- `CreateFromTemplate(opts TemplateOptions) (string, error)`

`TemplateOptions` groups:

- `Content` - template content.
- `TodosContent` - todos section content.
- `CurrentDate` - date used for date variables.
- `PreviousDate` - optional previous journal date.
- `Journal` - optional journal structure for statistics.
- `CustomVars` - optional custom variables map.

