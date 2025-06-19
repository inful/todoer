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

- `{{.Date}}` - Current date in YYYY-MM-DD format  
- `{{.TODOS}}` - The uncompleted tasks section content
- `{{.PreviousDate}}` - Date of the previous journal that todos came from (YYYY-MM-DD format, empty if no previous journal)

#### Template Fallback

1. **Explicit template** (`--template-file` or config)
2. **XDG Config**: `$XDG_CONFIG_HOME/todoer/template.md`
3. **Embedded default**: Basic structure with `## Todos` section

If a template has a `## Todos` section without the `{{.TODOS}}` placeholder, uncompleted tasks are automatically inserted into that section.

**Example template using PreviousDate:**

```markdown
---
title: {{.Date}}
{{if .PreviousDate}}previous: {{.PreviousDate}}{{end}}
---

# Daily Journal - {{.Date}}

{{if .PreviousDate}}## Todos (from {{.PreviousDate}}){{else}}## Todos{{end}}

{{.TODOS}}

## Notes

*Today's notes go here*
```

## Implementation Details

The tool uses a regex-based parser to analyze the journal format and maintains hierarchical structure. Tasks are considered "completed" only when both the task and all subtasks are marked as completed.

For detailed implementation examples and edge cases, see the comprehensive test suite in `testdata/`.
