# Contributing to todoer

Thanks for your interest in contributing. This document covers the day-to-day workflow: how to set up a development environment, how to run the tests, and the conventions the project follows so your changes land clean.

## Project at a glance

- **Language:** Go (see `go.mod` for the version; currently 1.25)
- **CLI framework:** [kong](https://github.com/alecthomas/kong)
- **TUI:** [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss)
- **License:** GPL-3.0
- **Module path:** `github.com/inful/todoer`

## Development setup

```bash
git clone https://github.com/inful/todoer
cd todoer

# Run the test suite
go test ./...

# Run the test suite with race detection
go test -race ./...

# Run the linter (config in .golangci.yml)
golangci-lint run

# Build the binary
go build -o todoer ./cmd/todoer
```

The dev container (`.devcontainer/`) is set up for VS Code; if you use it, the toolchain is pre-installed.

## Layout

```
cmd/todoer/     CLI entry point, kong commands, TUI (bubbletea model)
pkg/core/       Domain logic: parser, journal, file I/O, templates
pkg/generator/  Public library API (functional options)
tests/          External / black-box tests, golden fixtures in tests/testdata/
docs/           User-facing documentation and ADRs
demos/          Example templates and source files
```

The `tests/` directory exists by deliberate choice (see `tests/README.md`): it houses tests that exercise the public API or the compiled binary as a black box. Internal/colocation tests live next to the code they test (`*_test.go` siblings).

## Workflow

1. **Open an issue first** for non-trivial changes. The maintainer may have context that changes the approach.
2. **Fork and create a branch.** Branch names are not enforced; `feat/<slug>`, `fix/<slug>`, or `chore/<slug>` are fine.
3. **Write tests first.** The project follows TDD: pin the desired behaviour with a failing test, then implement. The full suite has 80%+ coverage per package; please don't drop coverage for new code.
4. **Match the style.** Run `golangci-lint run` before pushing. The repo's lint config (`.golangci.yml`) enforces `errcheck`, `govet`, `ineffassign`, `modernize`, `staticcheck`, `revive`, `unused`, `testableexamples`, `usetesting`, `contextcheck`. New code must pass cleanly.
5. **Commit with Conventional Commits.** `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`, `style:` — the changelog is generated from these. One logical change per commit.
6. **Push and open a PR.** The CI runs `go build ./...` and `go test ./...` on every push to a PR. All green is the bar.

## Coding conventions

These are the conventions the project already uses; please follow them so reviewers can focus on substance, not style.

- **Error wrapping:** always wrap with `fmt.Errorf("%w", ...)` so callers can use `errors.Is`/`errors.As`. Use sentinel errors for caller-actionable cases (see `cmd/todoer/validation.go` for the canonical list).
- **No panics in production code.** Errors are returned, not raised. The only `os.Exit` is in `main`'s `fatalError`.
- **Idempotency matters.** `todoer new` is meant to be safe to run repeatedly. Changes to the daily-flow code path must preserve that.
- **Atomic writes** go through `cmd/todoer/fs.go::safeWriteFile`. Don't write to disk any other way in the CLI.
- **Public API stability:** `pkg/core` and `pkg/generator` are the project's stable surface. Changes to exported names need an ADR (`docs/ADR-NNNN-*.md`).
- **No new top-level deps without discussion.** The dependency footprint is intentionally small. If you think a new module is required, open an issue first.
- **Comments explain *why*, not *what*.** The existing code leans on docstrings to encode intent ("Calendar dates, not elapsed durations — see daysBetween for why..."). Match that register.
- **Don't introduce global mutable state.** Compiled regexes and immutable style values are fine; everything else should be passed in.

## ADRs

Architectural decisions live in `docs/ADR-NNNN-<slug>.md`. If your change is more than a small refactor, write a short ADR that captures the decision, alternatives considered, and consequences. See `docs/ADR-0001-markdown-native-carryover-and-sync.md` for the format.

## Tests

- Colocate unit tests with the code (`foo.go` → `foo_test.go`).
- External / black-box tests go in `tests/`. Golden fixtures for round-trip behaviour live in `tests/testdata/<scenario>/` with `input.md`, `expected_output.md`, and `expected_input_after.md`.
- Use `t.Setenv` and `t.TempDir`; do not mutate process-wide state.
- The race detector must pass: `go test -race ./...`.

## Security reports

Please see [SECURITY.md](SECURITY.md). Do **not** file public issues for security bugs.

## License

By contributing, you agree that your contributions will be licensed under the project's GPL-3.0 license.