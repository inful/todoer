# Library Usage

This project can be used both as a CLI tool and as a Go library for processing TODO journal files.

The library API is built around the **generator** package and its
options-based constructors. This document shows how to embed todoer in
your own Go programs.

> Note: Older constructors such as `NewGenerator` and
> `NewGeneratorFromFile` have been removed. New code should always use
> the options-based API described below.

## Basic usage

Process in-memory journal content using a template string and a
template date:

```go
package main

import (
    "fmt"
    "log"

    "git.luguber.info/inful/todoer/pkg/generator"
)

func main() {
    templateContent := `# My TODOs - {{.Date}}

## Todos

{{.TODOS}}
`

    journalContent := `## Todos

- [[2025-03-01]]
    - [ ] Uncompleted task
    - [x] Completed task
`

    // Create a generator with template content and template date
    gen, err := generator.NewGeneratorWithOptions(templateContent, "2025-03-01")
    if err != nil {
        log.Fatal(err)
    }

    // Process journal content
    res, err := gen.Process(journalContent)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Modified original:\n" + res.ModifiedOriginal)
    fmt.Println("New file:\n" + res.NewFile)
}
```

## Generator from template file

Create a generator from a template file and process a journal file
directly:

```go
package main

import (
    "fmt"
    "log"

    "git.luguber.info/inful/todoer/pkg/generator"
)

func main() {
    // Create generator from a template file
    gen, err := generator.NewGeneratorFromFileWithOptions(
        "/path/to/template.md",
        "2025-03-01",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Process a file directly
    res, err := gen.ProcessFile("/path/to/journal.md")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Modified original:\n" + res.ModifiedOriginal)
    fmt.Println("New file:\n" + res.NewFile)
}
```

## Using options

The generator constructors accept functional options for additional
configuration. This allows you to specify a previous date, custom
variables, and more without breaking changes.

### With Previous Date

```go
gen, err := generator.NewGeneratorWithOptions(
    templateContent,
    "2025-03-01",
    generator.WithPreviousDate("2025-02-28"),
)
```

### With Custom Variables

```go
customVars := map[string]interface{}{
    "ProjectName": "My Project",
    "Owner":       "Jane Doe",
}

gen, err := generator.NewGeneratorWithOptions(
    templateContent,
    "2025-03-01",
    generator.WithCustomVariables(customVars),
)
```

In templates, custom variables are available under `.Custom`, for
example `{{.Custom.ProjectName}}`.

### Reconfiguring an Existing Generator

You can derive a new generator from an existing one by applying
additional options:

```go
gen1, err := generator.NewGeneratorWithOptions(templateContent, "2025-03-01")
if err != nil {
    log.Fatal(err)
}

// Create a new generator with extra configuration
gen2, err := gen1.WithOptions(
    generator.WithPreviousDate("2025-02-28"),
)
if err != nil {
    log.Fatal(err)
}
```

## API Reference (Library)

All types and functions below are in the
`git.luguber.info/inful/todoer/pkg/generator` package.

### Constructors

#### `func NewGeneratorWithOptions(templateContent, templateDate string, opts ...Option) (*Generator, error)`

Creates a new `Generator` with the provided template content and date
for template rendering. The template is validated at creation time, and
an error is returned if it is invalid.

Optional configuration is provided via `Option` values, such as
`WithPreviousDate` and `WithCustomVariables`.

#### `func NewGeneratorFromFileWithOptions(templateFile, templateDate string, opts ...Option) (*Generator, error)`

Like `NewGeneratorWithOptions`, but reads the template content from a
file.

### Options

#### `func WithPreviousDate(previousDate string) Option`

Sets an explicit previous date used to populate the `Previous*`
template variables when rendering.

#### `func WithCustomVariables(vars map[string]interface{}) Option`

Provides custom template variables that are exposed under the `.Custom`
field in templates.

Custom variable names must be valid Go template identifiers and must
not conflict with built-in variable names.

#### `func WithFrontmatterDateKey(key string) Option`

Overrides the key used to extract the date from frontmatter when
processing files.

#### `func WithTodosHeader(header string) Option`

Overrides the header that marks the todos section (default:
`"## Todos"`).

### Processing Methods

#### `func (g *Generator) Process(originalContent string) (*ProcessResult, error)`

Processes the journal content and returns both the modified original
and the newly generated file content.

#### `func (g *Generator) ProcessFile(filename string) (*ProcessResult, error)`

Reads a journal file from disk, processes it, and returns the results.

### Reconfiguration

#### `func (g *Generator) WithOptions(opts ...Option) (*Generator, error)`

Returns a new `Generator` based on `g` with additional options applied.
The original generator is not modified.

### Results

#### `ProcessResult`

```go
type ProcessResult struct {
    ModifiedOriginal io.Reader
    NewFile         io.Reader
}
```

`ModifiedOriginal` is an `io.Reader` for the original journal content
with completed tasks marked with date tags. `NewFile` is an
`io.Reader` for the new file content with uncompleted tasks formatted
using the template.

## Journal Format Requirements

The journal content must follow this general structure (see
`README.md` and `testdata/` for more examples):

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

Only the todos section (by default `## Todos`, configurable via
`WithTodosHeader`) is processed. Other content is preserved as-is.

## Template Format

Templates use Go's `text/template` syntax with variables and functions
documented in `README.md` under the **Templates** section.

Common variables include:

- `{{.Date}}` – The template date specified when creating the
  generator
- `{{.TODOS}}` – The uncompleted tasks content
- `{{.TotalTodos}}` – Number of incomplete todos being carried over
- `{{.Custom.VariableName}}` – Custom variables passed via
  `WithCustomVariables`

Example template:

```markdown
# My TODO List - {{.Date}}

## Outstanding Tasks

{{.TODOS}}

## Generated

This file was generated on {{.Date}}.
```

## Running Examples

You can see additional generator usage examples in this document. The
primary API surface is the options-based constructor
`NewGeneratorWithOptions`, optional `Option` values, and the
`Process`/`ProcessFile` methods.
