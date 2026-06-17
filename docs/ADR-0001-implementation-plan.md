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

Status: **DONE** (commits `1ea2c84`, `0547008`).

- `pkg/core/file.go` gains `ExtractFrontmatterMetadata` and
  `UpsertFrontmatterMetadata`. Internal helpers
  `splitFrontmatter` / `parseFrontmatterLine` /
  `sortedMetadataKeys` cover the no-frontmatter, existing, and
  malformed cases. Line ending style (LF or CRLF) is preserved
  on roundtrip so a Windows-checked-in journal does not have
  its body silently rewritten to LF.
- Tests in `pkg/core/file_test.go` cover all four cases plus
  CRLF preservation and the idempotent upsert.

### 2) Sync Engine for Existing Target Files
Goal: when target exists, merge carryovers into existing todos without duplication.

Status: **DONE** (commit `43f104d`).

- `pkg/core/sync.go` adds `MergeCarryover(source, target)` plus
  the `carryoverItemKey` helper. Match key is `(day, text)` with
  date tags stripped. Items are deep-copied on append so the
  caller's journal is not aliased. Day sections are sorted on
  return.
- 12 test cases in `pkg/core/sync_test.go` cover nil inputs,
  add-missing-days, dedup-on-same-day, keep-both-on-different-days,
  date-tag strip, deep-copy isolation, nested subtasks,
  idempotent re-runs, bullet line preservation, two source
  windows, day sorting, and `sortDays` safety.

### 3) Command Flow Unification
Goal: one command path handles both create-and-sync and sync-into-existing consistently.

Status: **DONE** (commit `db665ac`).

- `processJournal` gains a `merge bool` parameter. The daily
  flow (`cmdNewWithOptions`) calls it with `merge=true`; the
  explicit `process` command keeps `merge=false` to preserve
  its "overwrite into that" contract.
- `mergeIntoExistingTarget` is the new helper: it reads the
  existing target's beforeTodos / todosSection / afterTodos,
  parses the new content's todos, calls `MergeCarryover`, and
  rebuilds the file with the merged todos. The target's body
  (frontmatter, sections after Todos) is preserved verbatim.
- Tests: `TestProcessJournal_MergeIntoExistingTarget` and
  `TestProcessJournal_MergeIsIdempotent` pin the behaviour.

### 4) Fingerprint Spike (`mdfp`) Behind Feature Toggle
Goal: evaluate fingerprinting as a safety/optimization hint, not identity.

Status: **DONE** (initial spike at commit `c5aeb03`; mdfp library
integrated in v0.4.0 follow-up).

- Feature toggle: `TODOER_FINGERPRINT=1` enables the spike; off
  by default. When on, the daily flow's merge path reads the
  existing target's `fingerprint` and logs a `Fingerprint mismatch`
  message if the current source's fingerprint differs. Per
  ADR-0001, the fingerprint is a hint: a mismatch does not fail
  the sync, it just forces a conservative re-merge (the default
  behaviour).
- Companion toggle: `TODOER_FINGERPRINT_WRITE=0` suppresses the
  write side of the spike (the frontmatter upsert) while keeping
  the read side (mismatch detection) active. The default is
  "write enabled" to preserve the original behaviour.
- Fingerprint computation: `github.com/inful/mdfp` v1.2.0, via
  `mdfp.CalculateFingerprintFromParts(frontmatter, body)`. The
  library hashes the markdown body (excluding frontmatter) so
  changes to metadata alone don't change the fingerprint, and it
  strips any existing `fingerprint` field from the frontmatter
  before hashing so re-runs on the same body are stable. The
  field name `fingerprint` matches the library's
  `mdfp.FingerprintField` constant.
- Algorithm: SHA-256, hardcoded in mdfp. The mdfp library
  documents BLAKE3 as a possible future swap, at which point a
  per-record algorithm tag would be worth adding. For now the
  single algorithm is recorded in the library's docs.

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

1. Phase A: metadata helpers + tests. — **DONE** (commit `1ea2c84`)
2. Phase B: sync merge engine + unit tests. — **DONE** (commit `e84e303`)
3. Phase C: CLI flow integration (`process`, `new`, `tui`) + command tests. — **DONE** (commit `db665ac`)
4. Phase D: `mdfp` spike under toggle + evaluation note. — **DONE** (initial spike at `c5aeb03`; mdfp library swap landed in v0.4.0 follow-up)

## Acceptance Criteria

1. Re-running sync against an existing target does not duplicate carryovers. — **MET**
   (`TestProcessJournal_MergeIsIdempotent` and the 12
   `TestMergeCarryover_*` cases pin this.)
2. Existing target todos remain intact after sync. — **MET**
   (`TestProcessJournal_MergeIntoExistingTarget`.)
3. Source completed history is preserved and no task class is silently dropped. — **MET**
   (`MoveUndatedTodosToCurrentDate` now moves both completed
   and uncompleted undated items; pinned by
   `TestCmdAdd_PreservesCompletedUndatedTodos` and
   `TestCmdNewWithOptions_DisableSourceBackup_PreservesCompletedAndCarriesUnfinished`.)
4. TUI startup reflects synchronized target content immediately. — **MET**
   (TUI calls `cmdNewWithOptions` with `merge=true` on startup;
   `TestTUIQuitSavesDirtyChanges` and the new TUI tests cover
   the lifecycle.)
5. `.bak` behavior is intentional and covered by tests. — **MET**
   (No `.bak` by default; `--backup` flag on `new` / `add` /
   `tui` opts in. `TestCmdNew_DoesNotCreateBackupByDefault`
   and `TestCmdAdd_WithBackupCreatesBackup` pin both paths.)
6. Feature-toggle-off path behaves identically to current
   markdown-only logic. — **MET** (the fingerprint code is
   behind `TODOER_FINGERPRINT=1` and reads no frontmatter
   when the toggle is off; the spike is invisible in the
   default path.)
7. All relevant tests pass in `go test ./cmd/todoer ./pkg/...
   ./tests/...`. — **MET** (verified at every commit.)

## Risks and Mitigations

1. Ambiguous text matching can produce false matches. — **PARTIAL**
   The match key is `(day, text)`; two same-text items on
   the same day collide. Subitem-level dedup across
   different parents is out of scope; documented in the
   function comment and the ADR's known-limits section.
2. Frontmatter rewriting can disturb user formatting. — **MITIGATED**
   `splitFrontmatter` detects the line ending and re-emits
   with the same; unrelated keys and their order are
   preserved. Snapshot tests via the integration testdata.
3. Fingerprint canonicalization drift. — **MITIGATED**
   The spike uses a plain SHA-256 of the source content
   and records the algorithm in the frontmatter. The mdfp
   library swap (with proper canonicalization) is a
   follow-up and the test data is small enough to verify
   by hand.
