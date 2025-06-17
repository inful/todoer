# Library Usage

This package can be used both as a CLI tool and as a Go library for processing TODO journal files.

## Library Interface

### Basic Usage

```go
package main

import (
    "fmt"
    "io"
    "log"
)

func main() {
    // Create a generator with template content and template date
    generator, err := NewGenerator(templateContent, "2025-03-01")
    if err != nil {
        log.Fatal(err)
    }

    // Process journal content
    result, err := generator.Process(journalContent)
    if err != nil {
        log.Fatal(err)
    }

    // Get the results as io.Reader instances
    modifiedOriginal := result.ModifiedOriginal  // Contains completed tasks with date tags
    newFile := result.NewFile                    // Contains uncompleted tasks formatted with template
}
```

### Generator from File

```go
// Create generator from a template file
generator, err := NewGeneratorFromFile("/path/to/template.md", "2025-03-01")
if err != nil {
    log.Fatal(err)
}

// Process a file directly
result, err := generator.ProcessFile("/path/to/journal.md")
if err != nil {
    log.Fatal(err)
}
```

### API Reference

#### `NewGenerator(templateContent, templateDate string) (*Generator, error)`
Creates a new Generator with the provided template content and date for template rendering.

#### `NewGeneratorFromFile(templateFile, templateDate string) (*Generator, error)`
Creates a new Generator by reading the template from a file.

#### `(*Generator) Process(originalContent string) (*ProcessResult, error)`
Processes the journal content and returns readers for both the modified original and new file.

#### `(*Generator) ProcessFile(filename string) (*ProcessResult, error)`
Processes a journal file and returns readers for both outputs.

#### `ProcessResult`
Contains:
- `ModifiedOriginal io.Reader` - Original journal with completed tasks marked with date tags
- `NewFile io.Reader` - New file with uncompleted tasks formatted using the template

### Journal Format Requirements

The journal content must follow this format:

```markdown
---
title: "YYYY-MM-DD"
---

# Journal Title

## TODOS

- [[YYYY-MM-DD]]
  - [ ] Uncompleted task
  - [x] Completed task
  - [ ] Another task
    - [ ] Subtask
    - [x] Completed subtask

## Other sections...
```

### Template Format

Templates use Go template syntax with these variables:
- `{{.Date}}` - The template date specified when creating the generator
- `{{.TODOS}}` - The uncompleted tasks content

Example template:
```markdown
# My TODO List - {{.Date}}

## Outstanding Tasks

{{.TODOS}}

## Generated

This file was generated on {{.Date}}.
```

### Running Examples

```bash
./todoer --examples
```

This will run example usage of the library interface to demonstrate functionality.
