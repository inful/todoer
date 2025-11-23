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
todoer new --root-dir "./my-journals" --template-file "custom.md"

todoer process source.md target.md --template-file "template.md"
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
todoer new --root-dir "./journals"
```

Behavior:

- Creates a new file under `ROOT/YYYY/MM/YYYY-MM-DD.md`.
- Locates the most recent journal before today in the same root.
- Moves incomplete todos to the new file.
- Leaves completed todos in the previous file, tagged with the completion date.
- Creates a backup of the previous file before modifying it.
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
4. Creates a backup of the source file before modifications.

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
todoer new --template-file "./template.md"
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
todoer preview --template-file "template.md" --date "2025-06-20"

# Provide a specific todos section file
todoer preview --todos-file "todos_sample.md"

# Provide todos content directly
todoer preview --todos-string "- [ ] Example task"

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
vim $(todoer new --print-path)

nvim $(todoer new --print-path)

code $(todoer new --print-path)

# Process a journal and view the output
cat $(todoer process source.md target.md --print-path)

# Store the path in a variable
NEW_JOURNAL=$(todoer new --print-path)
echo "Created: $NEW_JOURNAL"
```

When `--print-path` is set, informational messages and logs are
suppressed so that only the path is written to standard output.
