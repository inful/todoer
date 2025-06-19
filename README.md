# Keeping track of todos in a daily journal

In a journal, I keep a list of TODOS in the following format

```markdown
---
title: 2025-05-13
---

# Title

Any content here

## Todos

- [[2025-05-12]]
  - [ ] An unfinished todo
  - [x] A completed todo
- [[2025-05-11]]
  - [ ] Unfinished
    - [ ] Unfinished subtask
  - [ ] Unfinished 2
    - [x] Completed subtask
    - [ ] Uncompleted subtask
- [[2025-05-10]]
  - [x] Completed

## Section

Any content here

```

I would like to create a utility in go that moves uncompleted tasks
to a new journal file in the following way:

```markdown
---
title: 2025-05-14
---

# Title

Any content here

## Todos

- [[2025-05-12]]
  - [ ] An unfinished todo
- [[2025-05-11]]
  - [ ] Unfinished
    - [ ] Unfinished subtask
  - [ ] Unfinished 2
    - [x] Completed subtask #2025-05-13
    - [ ] Uncompleted subtask

## Section

Any content here

```

and the original file should keep the following

```markdown
---
title: 2025-05-13
---

# Title

Any conttent here

## Todos

- [[2025-05-12]]
  - [x] A completed todo #2025-05-13
- [[2025-05-10]]
  - [x] Completed #2025-05-13

## Section

Any content here

```

If I complete some tasks in the new file like this:

```markdown
---
title: 2025-05-14
---

# Title

Any content here

## Todos

- [[2025-05-12]]
  - [ ] An unfinished todo
- [[2025-05-11]]
  - [ ] Unfinished
    - [ ] Unfinished subtask
  - [x] Unfinished 2
    - [x] Completed subtask #2025-05-13
    - [x] Uncompleted subtask

## Section

Any content here

```

The next run should produce the following in the new file:

```markdown
---
title: 2025-05-15
---

# Title

Any content here

## Todos

- [[2025-05-12]]
  - [ ] An unfinished todo
- [[2025-05-11]]
  - [ ] Unfinished
    - [ ] Unfinished subtask

## Section

Any content here

```

And the old file should contain the following:

```markdown
---
title: 2025-05-14
---

# Title

Any content here

## Todos

- [[2025-05-11]]
  - [x] Unfinished 2 #2025-05-14
    - [x] Completed subtask #2025-05-13
    - [x] Uncompleted subtask #2025-05-14

## Section

Any content here

```

Only content in the `## Todos` section should be touched. The rest of the file contents should be untouched.

## Usage

To use this utility, run:

```bash
go build
./todoer <source_file> <target_file> [template_file]
```

### Basic Usage

```bash
./todoer journal.md new_journal.md
```

This will:

1. Parse the source journal file
2. Keep only completed tasks in the source file
3. Move uncompleted tasks to the target file
4. Add date tags to completed subtasks in the target file

### Template Usage

```bash
./todoer journal.md new_journal.md template.md
```

When a template file is provided, the target file will be created using the template structure instead of copying the original file structure.

#### Template Variables

Templates can use the following variables:

- `{{date}}` - Current date in YYYY-MM-DD format
- `{{TODOS}}` - The uncompleted tasks section content

#### Example Template

```markdown
---
title: {{date}}
tags: journal, daily
---

# Daily Journal - {{date}}

## Todos

{{TODOS}}

## Notes

*Add your notes for {{date}} here.*

## Reflection

*What went well today?*

## Tomorrow's Focus

*What are the key priorities for tomorrow?*
```

#### Template with Automatic TODOS Insertion

If your template doesn't use the `{{TODOS}}` placeholder but has a `## Todos` section, the uncompleted tasks will be automatically inserted into that section:

```markdown
---
title: {{date}}
---

# Journal Entry

## Todos

## Notes

Today's notes...
```

The tool will automatically insert the uncompleted tasks between the `## Todos` header and the next section.

## Implementation Details

The implementation uses a regex-based parser to analyze the journal format. The parser handles nested todo items with proper indentation and maintains the hierarchical structure.

Tasks are considered "completed" only if the task itself and all subtasks are marked as completed.
