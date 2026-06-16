# ADR-0001 Implementation Plan

## Scope
Implement the accepted ADR-0001 strategy:

- Markdown-native carryover markers in frontmatter.
- Idempotent merge-into-existing behavior for target journals.
- Deterministic markdown-only matching for carryover dedupe.
- Unified flow for create/sync behavior.
- Timeboxed `mdfp` spike behind a feature toggle.

## Current Baseline
Relevant code paths today:

- `cmd/todoer/journal.go`: `cmdNewWithOptions`, `processJournal` (source/target writes and backup behavior).
- `cmd/todoer/tui.go`: runs `cmdNewWithOptions(..., preserveSourceBackup=false)` before loading today.
- `pkg/generator/generator.go`: transforms one source file into modified source + new target content.
- `pkg/core/file.go`, `pkg/core/journal.go`: todos extraction, split/tag logic, journal serialization.

Current tests already cover basic process behavior and TUI lifecycle, but ADR-level sync/idempotency and metadata behavior are not fully locked down.

## Workstreams

### 1) Metadata Model and Frontmatter I/O
Goal: add carryover metadata without breaking existing user frontmatter.

Tasks:

1. Define metadata keys and versioning:
   - `todoer_carryover_to`
   - `todoer_carryover_updated_at`
   - optional fingerprint keys for later spike (`todoer_fingerprint_algo`, `todoer_fingerprint_value`)
2. Implement frontmatter read/update helpers that preserve unrelated keys and order as much as practical.
3. Add tests for:
   - files with no frontmatter,
   - files with existing frontmatter,
   - idempotent updates,
   - malformed frontmatter fallback behavior.

Likely files:

- `pkg/core/file.go`
- `pkg/core/file_test.go`

### 2) Sync Engine for Existing Target Files
Goal: when target exists, merge carryovers into existing todos without duplication.

Tasks:

1. Introduce a dedicated sync function that takes:
   - source todos (carryover candidates),
   - target todos (already present),
   - matching strategy context (day hierarchy + normalized text).
2. Implement deterministic matching and dedupe:
   - normalize checkbox text,
   - include parent/day context in key derivation,
   - keep target item if already present.
3. Preserve behavior for completed items in source (do not lose completed history in source update).
4. Ensure repeated sync runs are idempotent.

Likely files:

- `pkg/core/journal.go` (or new `pkg/core/sync.go`)
- `pkg/core/journal_test.go` (or new sync-focused tests)
- `pkg/generator/generator.go`

### 3) Command Flow Unification
Goal: one command path handles both create-and-sync and sync-into-existing consistently.

Tasks:

1. Refactor `processJournal` so it can operate in two modes:
   - create target when absent,
   - merge when target already exists.
2. Keep source write transactional and explicit:
   - only write source when its content changed,
   - avoid unnecessary `.bak` artifacts when no source mutation is needed.
3. Ensure `cmdNewWithOptions` and TUI startup use the same sync routine and resulting target content is what TUI loads.

Likely files:

- `cmd/todoer/journal.go`
- `cmd/todoer/tui.go`
- `cmd/todoer/main_test.go`
- `cmd/todoer/tui_test.go`

### 4) Fingerprint Spike (`mdfp`) Behind Feature Toggle
Goal: evaluate fingerprinting as a safety/optimization hint, not identity.

Tasks:

1. Add internal feature toggle (config/env) for fingerprint-enabled sync checks.
2. Add canonicalization function for stable fingerprint input (newline normalization and frontmatter inclusion policy).
3. Persist fingerprint metadata only when toggle enabled.
4. On mismatch, force conservative re-merge; never hard-fail due to fingerprint mismatch.
5. Timebox spike and capture decision criteria: correctness, complexity, performance.

Likely files:

- `cmd/todoer/config.go`
- `pkg/core/file.go` (canonicalization helpers)
- `cmd/todoer/journal.go`
- new tests for toggle on/off and changed/unchanged paths

## Test Plan (Lockdown)

### Unit

1. Metadata round-trip tests for frontmatter updates.
2. Merge dedupe tests with:
   - identical task text,
   - same text in different day sections,
   - nested subtasks,
   - bullet lines attached to items.
3. Idempotency tests: run sync twice, output unchanged on second run.
4. Completed/uncompleted invariants:
   - completed source todos preserved in source history,
   - uncompleted moved to target,
   - no silent loss.

### Command-level

1. `processJournal` with existing target file should merge missing carryovers only.
2. `processJournal` should not create `.bak` when source remains unchanged.
3. `cmdNewWithOptions` should honor backup mode while still syncing correctly.

### TUI

1. Startup after carryover should display carryover todos in the loaded model.
2. Save/reload should keep carryover items stable.

### End-to-end

1. Add fixture-driven scenarios in `tests/testdata` for:
   - existing target with partial overlap,
   - repeated run idempotency,
   - changed older source then re-sync.

## Delivery Sequence

1. Phase A: metadata helpers + tests.
2. Phase B: sync merge engine + unit tests.
3. Phase C: CLI flow integration (`process`, `new`, `tui`) + command tests.
4. Phase D: `mdfp` spike under toggle + evaluation note.

## Acceptance Criteria

1. Re-running sync against an existing target does not duplicate carryovers.
2. Existing target todos remain intact after sync.
3. Source completed history is preserved and no task class is silently dropped.
4. TUI startup reflects synchronized target content immediately.
5. `.bak` behavior is intentional and covered by tests.
6. Feature-toggle-off path behaves identically to current markdown-only logic.
7. All relevant tests pass in `go test ./cmd/todoer ./pkg/... ./tests/...`.

## Risks and Mitigations

1. Ambiguous text matching can produce false matches.
   - Mitigation: include hierarchy context and conservative non-match behavior.
2. Frontmatter rewriting can disturb user formatting.
   - Mitigation: narrow, minimally invasive update function and snapshot tests.
3. Fingerprint canonicalization drift.
   - Mitigation: explicit versioned algorithm key + dedicated canonicalization tests.
