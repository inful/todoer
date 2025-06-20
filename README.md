# Todoer - Daily Journal Todo Management

A CLI tool for managing todos in daily journal files. Automatically carries over incomplete tasks between journal entries while preserving completed tasks with date annotations.

## Quick Start

```bash
# Build from source
go build -o todoer ./cmd/todoer

# Create today's journal (uses previous journal as source)
./todoer new

# Process any journal file
./todoer process source.md target.md
```

## Installation

### From Source

```bash
git clone <repository-url>
cd todoer
go build -o todoer ./cmd/todoer
```

### Usage Requirements

- Go 1.24.3 or later (for building)
- Journals must use the specific todo format (see below)

## Features

- **Automatic daily journal creation** in `YYYY/MM/YYYY-MM-DD.md` structure
- **Smart todo carryover** from previous journals
- **Configurable templates** for journal structure
- **Flexible configuration** via files, environment variables, or CLI flags
- **Preserves completed tasks** with date annotations
- **Maintains todo hierarchy** and organization

## Todo Format

Todoer requires journals to organize todos by date sections using this specific format:

- Todos organized under date headers: `- [[YYYY-MM-DD]]`
- Standard markdown checkboxes: `[ ]` for incomplete, `[x]` for complete
- Hierarchical structure with proper indentation
- Only the `## Todos` section is processed; other content remains untouched

**Processing behavior:**

- **Incomplete todos** are moved to the new journal
- **Completed todos** remain in the original journal with date tags added
- A todo is only "complete" when both the item AND all subtasks are marked `[x]`

For detailed examples and format specifications, see the `testdata/` directory in the repository.

## Usage

### Create Daily Journals

The `new` command creates today's journal file and carries over incomplete todos from the most recent previous journal:

```bash
# Create today's journal in current directory
./todoer new

# Create in specific directory
./todoer new --root-dir "~/Documents/journals"

# Use custom template
./todoer new --template-file "my_template.md"
```

**Journal Structure**: Files are created as `YYYY/MM/YYYY-MM-DD.md` (e.g., `2025/06/2025-06-20.md`)

**Behavior:**

- Finds the most recent journal file before today
- Moves incomplete todos to the new journal  
- Updates the previous journal, marking completed todos with date tags
- Creates a backup of the original file
- If no previous journal exists, creates from template

### Process Existing Journals

The `process` command works with any journal files:

```bash
# Basic processing
./todoer process source.md target.md

# With custom template and date
./todoer process source.md target.md --template-file "template.md" --template-date "2025-06-20"
```

This will:

1. Parse the source journal file
2. Move incomplete tasks to the target file
3. Keep completed tasks in the source file with date tags
4. Create a backup of the original source file

## Configuration

Todoer supports configuration through multiple methods, with the following priority order (highest to lowest):

1. **CLI flags** (`--root-dir`, `--template-file`)
2. **Environment variables** (`TODOER_ROOT_DIR`, `TODOER_TEMPLATE_FILE`)
3. **Configuration file** (`$XDG_CONFIG_HOME/todoer/config.toml`)
4. **Built-in defaults**

### Configuration File

Create a configuration file at `$XDG_CONFIG_HOME/todoer/config.toml` (usually `~/.config/todoer/config.toml`):

```toml
# Root directory where journal files will be stored
root_dir = "~/Documents/journals"

# Template file to use for new journals (optional)
template_file = "~/.config/todoer/my_template.md"

# Custom template variables
[custom_variables]
ProjectName = "My Project"
Version = "1.0.0"
Author = "John Doe"
Debug = true
MaxTasks = 10
Tags = ["work", "personal", "urgent"]
```

### Environment Variables

```bash
export TODOER_ROOT_DIR="~/Documents/journals"
export TODOER_TEMPLATE_FILE="~/.config/todoer/my_template.md"
```

### CLI Usage

```bash
# Create a new daily journal (uses configuration)
todoer new

# Create with custom settings
todoer new --root-dir "./my-journals" --template-file "custom.md"

# Process existing journal files
todoer process source.md target.md --template-file "template.md"
```

### Templates

Templates customize the structure of new journal files. For working examples, see `testdata/shared_template.md`.

#### Template Variables

**Current Date Variables:**
- `{{.Date}}` - Current date in YYYY-MM-DD format (e.g., `2025-06-20`)
- `{{.DateShort}}` - Short date format (e.g., `06/20/25`)
- `{{.DateLong}}` - Long date format (e.g., `June 20, 2025`)
- `{{.Year}}` - Year (e.g., `2025`)
- `{{.Month}}` - Month number (e.g., `06`)
- `{{.MonthName}}` - Month name (e.g., `June`)
- `{{.Day}}` - Day number (e.g., `20`)
- `{{.DayName}}` - Day name (e.g., `Friday`)
- `{{.WeekNumber}}` - Week number of year (e.g., `25`)

**Previous Date Variables** (empty if no previous journal):
- `{{.PreviousDate}}` - Previous journal date in YYYY-MM-DD format
- `{{.PreviousDateShort}}` - Short format (e.g., `06/19/25`)
- `{{.PreviousDateLong}}` - Long format (e.g., `June 19, 2025`)
- `{{.PreviousYear}}` - Previous year
- `{{.PreviousMonth}}` - Previous month number  
- `{{.PreviousMonthName}}` - Previous month name
- `{{.PreviousDay}}` - Previous day number
- `{{.PreviousDayName}}` - Previous day name (e.g., `Thursday`)
- `{{.PreviousWeekNumber}}` - Previous week number

**Content Variables:**

- `{{.TODOS}}` - The uncompleted tasks section content

**Todo Statistics Variables:**

- `{{.TotalTodos}}` - Number of incomplete todos being carried over
- `{{.CompletedTodos}}` - Number of completed todos found in source journal
- `{{.TodoDates}}` - List of unique dates that todos came from (array of strings)
- `{{.OldestTodoDate}}` - Date of the oldest incomplete todo (YYYY-MM-DD format, empty if no todos)
- `{{.TodoDaysSpan}}` - Number of days spanned by todos (from oldest to current date)

**Custom Variables:**

- `{{.Custom.VariableName}}` - User-defined variables from configuration file
- Custom variables are defined in the `[custom_variables]` section of your config file
- Supported types: string, int, float64, bool, arrays of these types
- Custom variable names must be valid Go template identifiers (start with letter/underscore, contain only letters/digits/underscores)
- Cannot use reserved names that conflict with built-in template variables

#### Template Functions

Beyond the built-in template variables, todoer provides powerful template functions for advanced customization:

##### Date Arithmetic Functions

```go
{{addDays "2025-01-15" 5}}        // Returns: 2025-01-20
{{subDays "2025-01-15" 3}}        // Returns: 2025-01-12
{{addWeeks "2025-01-15" 2}}       // Returns: 2025-01-29
{{addMonths "2025-01-15" 1}}      // Returns: 2025-02-15
{{daysDiff "2025-01-15" "2025-01-20"}}  // Returns: 5
```

##### Date Formatting Functions

```go
{{formatDate .Date "Monday, January 02, 2006"}}  // Returns: Friday, January 15, 2025
{{weekday .Date}}                 // Returns: Friday
{{isWeekend .Date}}              // Returns: false (true for Sat/Sun)
```

##### String Manipulation Functions

```go
{{upper "hello world"}}          // Returns: HELLO WORLD
{{lower "HELLO WORLD"}}          // Returns: hello world
{{title "hello world"}}          // Returns: Hello World
{{trim "  spaced  "}}            // Returns: spaced
{{replace "old" "new" "old text"}}  // Returns: new text
{{contains "hello world" "world"}}  // Returns: true
{{hasPrefix "hello" "he"}}       // Returns: true
{{hasSuffix "world" "ld"}}       // Returns: true
{{split " " "hello world"}}      // Returns: ["hello", "world"]
{{join ", " .TodoDates}}         // Returns: 2025-01-15, 2025-01-16
{{repeat "abc" 3}}               // Returns: abcabcabc
{{len "hello"}}                  // Returns: 5
```

##### Utility Functions

```go
{{default "fallback" .EmptyValue}}  // Returns fallback if EmptyValue is empty
{{empty .SomeValue}}             // Returns true if SomeValue is empty/nil
{{notEmpty .SomeValue}}          // Returns true if SomeValue is not empty
{{seq 1 5}}                      // Returns: [1, 2, 3, 4, 5] (for range loops)
{{dict "key1" "value1" "key2" "value2"}}  // Creates a map
```

#### Advanced Template Example

```markdown
---
title: {{.Date}}
---

# {{formatDate .Date "Monday, January 02, 2006"}} Journal

{{$isWeekend := isWeekend .Date}}
{{$tomorrow := addDays .Date 1}}

**Today:** {{.DayName}} {{if $isWeekend}}🏖️ (Weekend){{else}}💼 (Workday){{end}}  
**Tomorrow:** {{formatDate $tomorrow "Monday, Jan 02"}} ({{weekday $tomorrow}})

## This Week Schedule
{{$monday := subDays .Date 6}}  {{/* Approximate start of week */}}
{{range seq 0 6}}
{{$day := addDays $monday .}}
- **{{formatDate $day "Mon 01/02"}}**: {{if isWeekend $day}}Weekend{{else}}Work{{end}}
{{end}}

## Todo Overview
{{if .TotalTodos}}
We have {{.TotalTodos}} {{if eq .TotalTodos 1}}todo{{else}}todos{{end}} 
{{if .TodoDaysSpan}}spanning {{.TodoDaysSpan}} days{{end}}.
{{else}}
🎉 No pending todos!
{{end}}

{{if and .PreviousDate .TotalTodos}}
**Carryover:** {{daysDiff .PreviousDate .Date}} days since {{.PreviousDate}}
{{end}}

## Tasks for {{title (lower .DayName)}}

{{.TODOS}}

---
*Generated on {{formatDate .Date "Jan 02, 2006"}} with todoer*
```

### Template Fallback

1. **Explicit template** (`--template-file` or config)
2. **XDG Config**: `$XDG_CONFIG_HOME/todoer/template.md`
3. **Embedded default**: Basic structure with `## Todos` section

If a template has a `## Todos` section without the `{{.TODOS}}` placeholder, uncompleted tasks are automatically inserted into that section.

**Example template using enhanced date variables:**

```markdown
---
title: {{.Date}}
created: {{.DateLong}}
week: {{.WeekNumber}}
{{if .PreviousDate}}from: {{.PreviousDateLong}}{{end}}
---

# {{.DayName}} Journal - Week {{.WeekNumber}}

## {{.DateLong}}

{{if .PreviousDate}}
### Todos (from {{.PreviousDayName}}, {{.PreviousDateShort}})
{{else}}
### Todos  
{{end}}

{{.TODOS}}

### Focus for {{.DayName}}
*Key priorities for today*

### Notes
*{{.MonthName}} {{.Day}}, {{.Year}} reflections*
```

**Example template using todo statistics:**

```markdown
---
date: {{.Date}}
---

# Daily Journal - {{.DateLong}} ({{.DayName}})

## Summary

Today is {{.DateLong}}, which is a {{.DayName}}.

{{if .PreviousDate}}Previous entry: {{.PreviousDateLong}} ({{.PreviousDayName}}){{end}}

## Todo Statistics

- **Total active todos**: {{.TotalTodos}}
- **Completed todos**: {{.CompletedTodos}}
{{if .OldestTodoDate}}- **Oldest todo date**: {{.OldestTodoDate}}{{end}}
{{if .TodoDaysSpan}}- **Days spanned by todos**: {{.TodoDaysSpan}}{{end}}
{{if .TodoDates}}- **Todo dates**: {{range $i, $date := .TodoDates}}{{if $i}}, {{end}}{{$date}}{{end}}{{end}}

## Today's Tasks

{{.TODOS}}

## Notes

Today's reflections...
```

**Example template using custom variables:**

```markdown
---
date: {{.Date}}
project: {{.Custom.ProjectName}}
version: {{.Custom.Version}}
---

# {{.Custom.ProjectName}} Daily Journal

**Date:** {{.DateLong}} ({{.DayName}})  
**Version:** {{.Custom.Version}}  
**Author:** {{.Custom.Author}}  
{{if .Custom.Debug}}**Debug Mode:** Enabled{{end}}

### Task Categories
{{range .Custom.Tags}}- {{.}}
{{end}}

### Statistics
- **Total active todos**: {{.TotalTodos}}
- **Completed todos**: {{.CompletedTodos}}

## Today's Tasks (Max: {{.Custom.MaxTasks}})

{{.TODOS}}

## Daily Notes

Reflections for {{.MonthName}} {{.Day}}, {{.Year}}
```

## Development Phases

The todoer project has been enhanced through multiple development phases:

### ✅ Phase 1: Enhanced Date Variables
- Added comprehensive date formatting variables (`DateShort`, `DateLong`, `DayName`, `WeekNumber`, etc.)
- Implemented previous date variants for referencing source journals
- Full backward compatibility maintained

### ✅ Phase 2: Todo Statistics
- Added todo counting and analysis (`TotalTodos`, `CompletedTodos`, etc.)
- Implemented date span tracking (`TodoDaysSpan`, `OldestTodoDate`)
- Enhanced template data with completion metrics

### ✅ Phase 3: Custom Variables via Config
- Added support for user-defined template variables through TOML configuration
- Custom variables available in templates via `.Custom.VariableName`
- Validation and error handling for custom variable configurations

### ✅ Phase 4: Template Functions
- **Date arithmetic**: `addDays`, `subDays`, `addWeeks`, `addMonths`, `daysDiff`
- **Date formatting**: `formatDate`, `weekday`, `isWeekend`
- **String manipulation**: `upper`, `lower`, `title`, `trim`, `replace`, `contains`, etc.
- **Utilities**: `default`, `empty`, `notEmpty`, `seq`, `dict`
- Robust error handling and graceful fallbacks

### Future Enhancements
- Additional template functions (mathematical operations, conditionals)
- Plugin system for custom processing logic
- Advanced configuration validation and schema
- Integration with external calendar/task systems

All phases include comprehensive test coverage and maintain full backward compatibility.

## Implementation Details

The tool uses a regex-based parser to analyze the journal format and maintains hierarchical structure. Tasks are considered "completed" only when both the task and all subtasks are marked as completed.

For detailed implementation examples and edge cases, see the comprehensive test suite in `testdata/`.
