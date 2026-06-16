# ADR-0001: Markdown-Native Carryover and Sync Strategy

## Status
Accepted

## Date
2026-03-14

## Context
Todoer currently supports `new`, `add`, `process`, and a minimal TUI for todo lifecycle updates.

As lifecycle features expand (for example adding todos for future dates), carryover behavior becomes more complex:

- A target day file may already exist before carryover is run.
- The immediately previous file may not contain the full active todo set.
- Users may update older files after newer files are created.

During design discussion, we evaluated approaches for stable todo identity and carryover reconciliation.

## Decision
We will keep Todoer fully markdown-native and avoid both:

1. Inline per-todo IDs in markdown content.
2. External identity stores (sidecar DB/JSON).

Instead, we adopt a markdown-only carryover model with frontmatter-based carryover markers.

### 1. Carryover metadata in frontmatter
Target journal files may include metadata indicating carryover sync has been applied.

Initial keys:

- `todoer_carryover_to`: target date (YYYY-MM-DD)
- `todoer_carryover_updated_at`: timestamp of last carryover sync

Additional source-horizon metadata may be added later if needed.

### 2. Idempotent sync behavior for existing files
`new`/future sync flows should be safe to run even when the target file already exists.

Expected behavior:

- If target file does not exist: create and carry over.
- If target file exists: merge missing carryovers into existing todos.
- Running repeatedly should not duplicate carryovers.

### 3. Markdown-only task matching
Carryover merge and deduplication use deterministic text-based matching derived from markdown content and hierarchy context.

This is accepted with known limits (see Consequences).

### 4. Unified user flow
Users should not need to choose between separate conceptual modes based on file existence.

The user-facing flow should remain simple: a single command path can both create and sync.

### 5. Evaluate markdown fingerprinting (`mdfp`) for post-sync change detection
We will evaluate incorporating `https://github.com/inful/mdfp` to detect whether source/target markdown documents changed after a carryover sync.

Scope of this evaluation:

- Use fingerprints only as an optimization and safety signal for sync decisions.
- Do not use fingerprints as authoritative task identity.
- Keep all persisted state markdown-native (frontmatter and/or deterministic recomputation).

Candidate usage:

- Compute and persist document fingerprints at sync time.
- On the next sync, compare prior and current fingerprints for relevant source windows.
- If changed, rerun merge with conservative behavior and update carryover metadata.

## Consequences
### Positive
- Preserves markdown readability and editability.
- No hidden external state that can drift from files.
- Fits tooling philosophy where markdown files are the source of truth.
- Supports idempotent carryover in existing files via frontmatter markers.
- Fingerprints can improve confidence when deciding whether incremental sync is still valid.

### Negative
- No perfect stable identity for todos across arbitrary rewrites.
- Renames/splits/merges of similar todo text can cause ambiguous matching.
- Matching quality depends on deterministic text/hierarchy heuristics.
- Fingerprinting introduces additional implementation complexity and canonicalization concerns.

## Alternatives Considered
### A) Inline stable todo IDs (rejected)
Example: HTML comment IDs on each todo line.

Rejected because it burdens source editing and introduces metadata noise users must move/delete correctly.

### B) External identity store (rejected)
Example: sidecar DB/JSON tracking task identities.

Rejected because it conflicts with markdown-only source-of-truth goals and creates drift/failure modes.

## Implementation Notes
- Frontmatter parsing/writing should preserve unrelated user metadata.
- Sync should be transactional for writes (atomic write path already exists).
- Conflict handling remains optimistic and markdown-based (no merge UI required for initial implementation).
- Fingerprint storage format should be explicit and versioned (for example `todoer_fingerprint_algo` + `todoer_fingerprint_value`).
- Fingerprints must be treated as hints; mismatch triggers re-merge, not hard failure.
- Canonicalization rules must be documented (for example newline normalization and frontmatter inclusion/exclusion).

## Follow-up Work
1. Implement carryover marker read/write in journal sync path.
2. Implement merge-into-existing behavior for target files.
3. Add tests for idempotent repeated sync and existing-file scenarios.
4. Extend `add --date`/date-targeted flows to call the same sync routine.
5. Timebox a spike to integrate `mdfp` behind an internal feature toggle.
6. Add tests for unchanged-vs-changed fingerprint paths and canonicalization edge cases.
7. Decide keep/adapt/drop `mdfp` based on correctness, complexity, and performance.
