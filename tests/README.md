# `tests/` directory

This directory contains the project's "external" test suite —
tests that exercise the public library API and the compiled
binary rather than the internals of any single package.

## Why this directory exists

Go's convention is to colocate tests with the code they test
(`pkg/core/parser_test.go` lives next to `pkg/core/parser.go`).
That is the right pattern for unit tests of internals, and
todoer follows it for most tests.

The `tests/` directory is the project's home for the categories
of tests that don't fit that pattern:

| File | Category |
|------|----------|
| `library_test.go`     | Exercises the public `pkg/core` and `pkg/generator` APIs as a black box (mirrors what an external consumer would do). |
| `integration_test.go` | Round-trip tests using the `tests/testdata/` golden files (input → expected output). |
| `cmd_test.go`         | Spawns the compiled `todoer` binary via `os/exec` to verify the CLI surface end-to-end. |
| `cmd_extended_test.go`| Additional CLI tests covering config files, env vars, and the kong flag parser. |
| `options_test.go`     | Options-API coverage for the generator's functional-options. |
| `simple_cli_test.go`  | Small smoke tests for the binary (one per behaviour). |
| `main_test.go`        | Shared setup; not a test entry point on its own. |

## Decision (Phase 5b, 2026-06-22)

This structure was reviewed during the architecture cleanup
(plan Phase 5b) and the decision is to **keep it as-is**.

Reasons:

- The `tests/testdata/` golden-file convention is a recognised
  Go pattern (cf. `text/template`).
- `package main` is correct here: these tests are
  "black-box from the repo root" — they only see what an external
  consumer of the compiled binary or the public library would.
- The tests/ package has no internal types; the `package main`
  is a marker, not a coupling.
- Moving these into `pkg/*_test.go` would mix white-box and
  black-box tests in the same files and dilute the signal.

The two adjacent refactor phases addressed the perceived cost
of the structure:

- Phase 4a split the god `cmd/todoer/main_test.go` into 8
  per-topic files so the colocation-test pattern works without
  any single file exceeding 600 lines.
- Phase 4c consolidated the duplicated test-helper constructors
  in `pkg/core/journal_test.go` and `pkg/core/parser_test.go`.

## Test-data

`tests/testdata/` contains 8 fixture directories, each with an
`input.md`, `expected_output.md`, and `expected_input_after.md`.
These are consumed by `integration_test.go` and `library_test.go`
to verify round-trip behaviour.
