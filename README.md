# Keeping track of todos in a daily journal

In a journal, I keep a list of TODOS in the following format

```markdown
---
title: 2025-05-13
---

# Title

Any content here

## TODOS

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

## TODOS

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

## TODOS

- [[2025-05-12]]
  - [x] A completed todo #2025-06-17
- [[2025-05-10]]
  - [x] Completed #2025-06-17

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

## TODOS

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

## TODOS

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

## TODOS

- [[2025-05-11]]
  - [x] Unfinished 2 #2025-05-14
    - [x] Completed subtask #2025-05-13
    - [x] Uncompleted subtask #2025-05-14

## Section

Any content here

```

Only content in the `## TODOS` section should be touched. The rest of the file contents should be untouched.

## Usage

To use this utility, run:

```bash
go build
./todoer <source_file> <target_file>
```

For example:

```bash
./todoer journal.md new_journal.md
```

This will:

1. Parse the source journal file
2. Keep only completed tasks in the source file
3. Move uncompleted tasks to the target file
4. Add date tags to completed subtasks in the target file

## Implementation Details

The implementation uses a regex-based parser to analyze the journal format. The parser handles nested todo items with proper indentation and maintains the hierarchical structure.

Tasks are considered "completed" only if the task itself and all subtasks are marked as completed.
