# Configuration Guide

The todoer application supports multiple configuration methods with the following priority order:

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Configuration file**
4. **Built-in defaults** (lowest priority)

## Configuration File

### Location

The configuration file is located at:

- `$XDG_CONFIG_HOME/todoer/config.toml` (if XDG_CONFIG_HOME is set)
- `$HOME/.config/todoer/config.toml` (default on Unix/Linux/macOS)

### Format

The configuration file uses TOML format:

```toml
# Root directory for journal files
root_dir = "/path/to/your/journals"

# Optional: Custom template file
template_file = "/path/to/your/template.md"

# Custom variables for templates
[custom_variables]
author = "Your Name"
project = "Project Name"
team = "Development Team"
tags = ["work", "personal"]
priority = 1
active = true
```

### Configuration Options

#### `root_dir` (string, required)

- **Description**: The root directory where journal files are stored
- **Default**: Current directory (`.`)
- **Example**: `"/home/user/journals"`
- **Environment Variable**: `TODOER_ROOT_DIR`

#### `template_file` (string, optional)

- **Description**: Path to a custom template file for generating new journals
- **Default**: Uses embedded default template
- **Example**: `"/home/user/.config/todoer/my_template.md"`
- **Environment Variable**: `TODOER_TEMPLATE_FILE`
- **Notes**:
  - If not specified, todoer will look for `$XDG_CONFIG_HOME/todoer/template.md`
  - Falls back to built-in template if no custom template is found

#### `custom_variables` (table, optional)

- **Description**: Custom variables that can be used in templates
- **Default**: Empty
- **Supported Types**:
  - `string`: Text values
  - `int`, `int32`, `int64`: Integer numbers
  - `float32`, `float64`: Decimal numbers
  - `bool`: True/false values
  - Arrays of the above types (e.g., `["item1", "item2"]`)

**Reserved Variable Names** (cannot be used):

- `Date`: Current date in YYYY-MM-DD format
- `DateLong`: Current date in long format
- `TODOS`: Todo items from source file
- `TotalTodos`: Count of total todo items
- `CompletedTodos`: Count of completed todo items
- `PreviousDate`: Date from previous journal

**Variable Name Rules**:

- Must start with a letter (a-z, A-Z) or underscore (_)
- Can contain letters, numbers, and underscores
- Cannot contain spaces, dashes, or special characters
- Examples: `author`, `_private`, `projectName`, `version2`

## Environment Variables

All configuration options can be overridden using environment variables:

```bash
# Set root directory
export TODOER_ROOT_DIR="/path/to/journals"

# Set custom template
export TODOER_TEMPLATE_FILE="/path/to/template.md"
```

## Template Variables

When creating custom templates, you can use these variables:

### Built-in Variables

- `{{.Date}}`: Current date (YYYY-MM-DD)
- `{{.DateLong}}`: Current date in long format
- `{{.TODOS}}`: Todo items extracted from source file
- `{{.TotalTodos}}`: Total number of todo items
- `{{.CompletedTodos}}`: Number of completed todo items
- `{{.PreviousDate}}`: Date from the previous journal (if available)

### Custom Variables

Access custom variables using the `Custom` namespace:

- `{{.Custom.author}}`: Author name from config
- `{{.Custom.project}}`: Project name from config
- `{{.Custom.tags}}`: Array of tags from config

### Template Functions

Todoer provides additional template functions:

#### Date Functions

- `{{formatDate .Date "Monday, January 02, 2006"}}`: Format date with custom layout
- `{{addDays .Date 7}}`: Add days to a date
- `{{subDays .Date 7}}`: Subtract days from a date
- `{{weekday .Date}}`: Get day of week
- `{{isWeekend .Date}}`: Check if date is weekend

#### String Functions

- `{{upper "text"}}`: Convert to uppercase
- `{{lower "text"}}`: Convert to lowercase
- `{{title "text"}}`: Convert to title case
- `{{trim " text "}}`: Remove whitespace
- `{{replace "text" "old" "new"}}`: Replace text

#### Utility Functions

- `{{shuffle "line1\nline2\nline3"}}`: Randomly shuffle lines
- `{{default "fallback" .Custom.optional}}`: Use fallback if value is empty

## Example Configuration

### Complete Configuration File

```toml
# ~/.config/todoer/config.toml
root_dir = "/home/user/Documents/journals"
template_file = "/home/user/.config/todoer/my_template.md"

[custom_variables]
author = "John Doe"
project = "Daily Planning System"
team = "Development"
priorities = ["urgent", "high", "medium", "low"]
version = 2
beta = true
```

### Custom Template Example

```markdown
---
title: {{.Date}}
author: {{.Custom.author}}
project: {{.Custom.project}}
---

# {{.Custom.project}} - {{formatDate .Date "Monday, January 02, 2006"}}

**Team**: {{.Custom.team}}  
**Version**: {{.Custom.version}}{{if .Custom.beta}} (Beta){{end}}

## Tasks for {{weekday .Date}}

{{.TODOS}}

## Priority Levels
{{range .Custom.priorities}}
- {{.}}
{{end}}

## Notes

*Generated on {{.DateLong}}*
```

## Validation

Todoer validates configuration to ensure:

1. **Required fields**: `root_dir` must be specified
2. **Path security**: No directory traversal attempts
3. **File accessibility**: Template files must exist and be readable
4. **Variable names**: Must follow valid identifier rules
5. **Variable types**: Only supported types are allowed
6. **Reserved names**: Cannot override built-in variables

## Troubleshooting

### Common Issues

1. **Config file not found**: Ensure the file is in the correct location with proper permissions
2. **Template file not found**: Check the path and file permissions
3. **Invalid variable names**: Use only letters, numbers, and underscores
4. **Permission errors**: Ensure todoer has read access to config and template files

### Debug Mode

Enable debug logging to see configuration details:

```bash
todoer --debug process source.md target.md
```

This will show:

- Configuration file location
- Template source being used
- Custom variables loaded
