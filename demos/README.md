# Demo Templates and Examples

This directory contains example templates and source files demonstrating various features of todoer.

## Template Examples

### `demo_custom_template.md`

Demonstrates how to create templates with custom variables and advanced template features.

### `demo_shuffle_template.md`

Shows how to use the shuffle functionality to randomize lists in your daily journals.

### `demo_stats_template.md`

Example template that includes todo statistics and completion metrics.

### `demo_template_functions.md`

Comprehensive example showcasing various template functions including date arithmetic, string manipulation, and utility functions.

### `enhanced_template_demo.md`

Advanced template example with complex layouts and multiple features combined.

## Source Examples

### `demo_source.md`

Example source journal file showing the expected format for todo processing.

### `demo_todos.md`

Example todos section on its own, for `preview --todos-file`.

## Usage

You can use these templates as starting points for your own journal templates:

```bash
# Create a new journal using a demo template
./todoer new --template-file demos/demo_custom_template.md

# Process a demo source file
./todoer process demos/demo_source.md output.md

# Render a template against a sample todos section
./todoer --template-file demos/demo_stats_template.md preview --todos-file demos/demo_todos.md
```

`demo_source.md` is a whole journal; `demo_todos.md` is only a todos
section, which is what `preview --todos-file` expects.

For more information on template syntax and features, see the main documentation.
