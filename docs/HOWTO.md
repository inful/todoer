# How-to Guides

## Configure storage and templates

### Configure journals root and template file

1. Create a configuration file at `$XDG_CONFIG_HOME/todoer/config.toml` (usually `~/.config/todoer/config.toml`).
2. Set the root directory and optional template file:

```toml
root_dir = "~/Documents/journals"

# Optional: template file to use for new journals
template_file = "~/.config/todoer/my_template.md"
```

3. Optionally override these values per run using CLI flags:

```bash
todoer --root-dir "./my-journals" --template-file "custom.md" new

todoer --template-file "template.md" process source.md target.md
```

4. Or via environment variables:

```bash
export TODOER_ROOT_DIR="~/Documents/journals"
export TODOER_TEMPLATE_FILE="~/.config/todoer/my_template.md"
```

Configuration precedence is:

1. CLI flags
2. Environment variables
3. Configuration file
4. Built-in defaults

## Create and process daily journals

### Create a new daily journal

To create todays journal file and carry over incomplete todos from the most recent previous journal:

```bash
# Uses configuration or defaults
todoer new

# Use an explicit root directory
todoer --root-dir "./journals" new
```

### Add a todo directly from the CLI

To add a new todo to today's journal:

```bash
todoer add review PR notes

# With an explicit journals root directory
todoer --root-dir "./journals" add write test plan
```

Behavior:

- If today's journal does not exist yet, todoer creates it first using
  the same transfer behavior as `new`.
- The new todo is appended as an unchecked item under today's date
  section.
- Multi-word todo text can be passed without quotes.

### Manage today's todos in a terminal UI

To open an interactive terminal UI for today's journal:

```bash
todoer tui

# With explicit root directory
todoer --root-dir "./journals" tui
```

Behavior:

- If today's journal does not exist yet, todoer creates it first using
  the same transfer behavior as `new`.
- You can move, toggle, add, and delete todos interactively.
- You can filter/search visible todos by text with `/` and clear with `c`.
- `s` saves changes, `r` reloads from disk, and both `q` and `ctrl+c` save (if dirty) and quit. A save that fails still quits, and prints the reason to stderr.
- `d` deletes: press it twice. The first press only arms the delete, so a slip on the key next to `j`/`k` cannot drop a task.
- Every save keeps the previous version of the journal in `ROOT/.backups/`.

Behavior:

- Creates a new file under `ROOT/YYYY/MM/YYYY-MM-DD.md`.
- Locates the most recent journal before today in the same root.
- Moves incomplete todos to the new file.
- Leaves completed todos in the previous file, tagged with the completion date.
- If today's journal already exists, `todoer new` does nothing: it
  reports the path and stops. Carryover is not re-applied, and nothing is
  merged. To carry over again, delete today's file first.
- The source journal is updated in place. Its previous version is kept
  in `ROOT/.backups/`, so it can be restored.
- If no previous journal exists, creates the file from the configured or embedded template.

### Process an existing journal file

To process one journal file into a new target file:

```bash
todoer process source.md target.md

# With explicit template and template date
todoer process source.md target.md \
  --template-file "template.md" \
  --template-date "2025-06-20"
```

Behavior:

1. Parses the source file and locates the todos section.
2. Moves incomplete tasks into the target file using the given template.
3. Keeps completed tasks in the source file with date tags.
4. The `process` command always overwrites the target. For the
   idempotent "merge into existing" behavior, use `new` (or
   `tui` / `add`), which share the same code path with
   `merge=true`.
5. The source file is updated in place. Its previous version, and the
   previous version of the target, are kept in `ROOT/.backups/`.

## Use custom templates

### Point todoer at a custom template file

1. Place your template at a known path, for example `~/.config/todoer/template.md`.
2. Reference it in configuration, environment, or CLI:

```toml
# config.toml
template_file = "~/.config/todoer/template.md"
```

```bash
# Override for a single run
todoer --template-file "./template.md" new
```

Template selection order:

1. Explicit template via `--template-file` or config.
2. `$XDG_CONFIG_HOME/todoer/template.md` if it exists.
3. The embedded default template.

### Insert todos into a template

To control where uncompleted todos are inserted, use the `{{.TODOS}}` placeholder inside the todos section:

```markdown
# Daily Journal - {{.Date}}

## Todos

{{.TODOS}}

## Notes

Notes here.
```

If a template has a todos section header (default `## Todos`) without the `{{.TODOS}}` placeholder, uncompleted tasks are inserted into that section automatically.

## Change how many backups are kept

Every write keeps the previous version of the file in `ROOT/.backups/`
(the copies rotate, so `.1` is the version before the last write). To
keep a different number of copies:

```toml
backup_count = 5
```

`0` keeps no backups at all. The same can be set with the
`TODOER_BACKUP_COUNT` environment variable.

## Change the todos section header

By default, todoer processes the `## Todos` section. To use a different header, for example `## Tasks`:

1. Configure the header in `config.toml`:

```toml
todos_header = "## Tasks"
```

2. Use the same header in your template and journals:

```markdown
# My Journal - {{.Date}}

## Tasks

{{.TODOS}}
```

Todoer will then process the `## Tasks` section instead of `## Todos`.

## Parser strictness inside the todos section

The parser is strict: inside the `## Todos` (or configured) section, only
the following line shapes are accepted:

- A day header: `- [[YYYY-MM-DD]]`
- A todo item: `- [ ]` (unchecked), or `- [x]` / `- [X]` / `- [-]` (all
  count as done), with optional sub-items at greater indentation.
- A bullet entry: `- text` (non-todo lines, associated with the nearest
  preceding todo item).
- A continuation line: indented text associated with the nearest
  preceding todo item.

Anything else inside the todos section is an error. If a free-form note,
a code block, a blockquote, or any other non-todo content is placed
between todo lines, `process` (and any other command that reparses the
journal) will fail with `unparseable line N: "<line>"`.

If you want a notes section, put it under a different `##` header. The
parser only processes the configured todos header and preserves all
other sections verbatim.

`todoer tui` is the exception: it opens read-only on a journal it cannot
parse (the header shows `read-only`) rather than refusing to start, because
a write would rewrite the section without that line.

## Use custom template variables

Todoer supports custom variables defined in the configuration file.

1. Define variables under `[custom_variables]` in `config.toml`:

```toml
[custom_variables]
ProjectName = "My Project"
Version = "1.0.0"
Author = "John Doe"
Debug = true
MaxTasks = 10
Tags = ["work", "personal", "urgent"]
```

2. Use these variables in templates via the `.Custom` field, for example:

```markdown
# {{.Custom.ProjectName}} - {{.Date}}

Maintainer: {{.Custom.Author}}
```

Notes:

- Supported types include string, integer, float, boolean, and arrays of these types.
- Variable names must be valid Go template identifiers and must not conflict with built-in names.

## Preview a template

Use the `preview` command to see how a template renders with a sample todos section and optional custom variables:

```bash
todoer --template-file "template.md" preview --date "2025-06-20"

# Provide todos content directly
todoer preview --todos-string "- [ ] Example task"

# Or a file that contains a todos section (a section, not a whole journal)
todoer preview --todos-file "my_todos.md"

# Provide custom variables as JSON
todoer preview --custom-vars '{"ProjectName":"My Project"}'
```

The command prints the rendered template to standard output.

## Use `--print-path` for scripting

The `--print-path` flag prints only the created or target file path to
standard output, which is useful in shell scripts and editor
integrations.

Available commands:

- `todoer new` - prints the created journal file path.
- `todoer process` - prints the target file path.

Examples:

```bash
# Create a new journal and open it in an editor
vim $(todoer --print-path new)

nvim $(todoer --print-path new)

code $(todoer --print-path new)

# Process a journal and view the output
cat $(todoer --print-path process source.md target.md)

# Store the path in a variable
NEW_JOURNAL=$(todoer --print-path new)
echo "Created: $NEW_JOURNAL"
```

When `--print-path` is set, informational messages and logs are
suppressed so that only the path is written to standard output.
