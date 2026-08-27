# Architecture Cleanup — Summary

This document records the result of the multi-phase architecture
cleanup (Phases 0–6) executed against the original
architecture review.

## Final metrics vs baseline

| Metric | Baseline | Final | Change |
|--------|---------:|------:|-------:|
| `tokensave_health.quality_signal` | 7,548 | 8,065 | **+517 (+6.8%)** |
| `processJournal` parameter count | 9 | 3 | `-6` |
| `tui.go::View()` line count | 90 | 11 (delegator) | `-79` |
| Largest production file (cmd/todoer) | 274 (`tui.go`) | 505 (`journal.go`) | (was 491 before, +14 with `ProcessOptions` doc) |
| Largest test file | 2,057 (`main_test.go`) | 474 (`config_test.go`) | **-1,583 (-77%)** |
| Top untested-function risk | 76 (`mergeIntoExistingTarget`) | 43 (`shortHash`) | **-33** |
| cmd/todoer test coverage | 70% (28/40) | 85% (34/40) | **+15%** |
| Dead-code false positives | 11 | 4 (all intra-file, not real) | `-7` (real) |
| Total tests added across phases | — | ~50 new direct tests | — |

## Phases executed

| Phase | Outcome |
|------:|---------|
| 0  | Baseline: pre-existing modernize lint in `validation.go` fixed (commit `8c06428`). |
| 1a | `TestMergeIntoExistingTarget` + 2 guards — 11 cases (commit `6fe0122`). |
| 1b | `TestSharedPaths`, `TestGetGenerator`, `TestCmdPreview` — 21 cases (commit `c70e5a7`). |
| 1c | TUI IO helpers: `hashBytes`, `checkExternalChanges`, `flattenTodoItems`, `newTUIModel` — 11 cases (commit `a038c17`). |
| 2a | `tuiStyles`, `tuiTheme`, `tuiKeymap`, `tuiHelpText`, `tuiInitStatus` moved to `tui_styles.go` (commit `46c6866`). |
| 2b | `View()` decomposed into `viewHeader`, `viewItems`, `viewInputLine`, `viewStatus` (each ≤ 28 lines) + 10-case regression test (commit `2213e42`). |
| 2c | **DEFERRED** — sub-struct grouping of `tuiModel` fields. Low ROI vs churn; the model is 11 fields, the methods are already decomposed. |
| 3a | `ProcessOptions` struct + 3-case characterization test (commit `4c5a95d`). |
| 3b | 14 callers migrated to `processJournalWithOptions` (commit `eb046db`). |
| 3c | Legacy 9-parameter `processJournal` removed; single struct-based entry point (commit `4504fc4`). |
| 4a | `cmd/todoer/main_test.go` (2,057 lines, 55 tests) split into 8 per-topic files (commit `2fff487`). |
| 4b | `cmd/todoer/tui_test.go` (1,155 lines, 33 tests) split into 6 mode-aligned files (commit `ebc8ece`). |
| 4c | `pkg/core/journal_test.go` and `parser_test.go` test-helper duplication consolidated (commit `2815585`). |
| 5a | Dead-code audit: 7 of the original 11 items resolved implicitly; 4 remaining are false positives (intra-file calls not tracked by the index). Nothing to delete. |
| 5b | `tests/` directory kept as-is; rationale documented in `tests/README.md` (commit `1169725`). |
| 6  | This summary. |

## Stress points addressed

- **Stress 1: `processJournal` 9-parameter signature** → 3 parameters (struct + config + logger). ✅
- **Stress 2: `tui.go` god file** → styles extracted to `tui_styles.go`; View split into 4 small methods. 274 → 208 lines. ⚠ (lines: not fully under 150; the model struct itself is 11 fields, the rest is Run/Init/Update + 4 small view methods, each <30 lines)
- **Stress 3: massive test files** → main_test.go split into 8 files, tui_test.go into 6 files. Largest test file is now 474 lines. ✅
- **Stress 4: untested hotspots in cmd layer** → `mergeIntoExistingTarget`, `getGenerator`, `cmdPreview`, `sharedPaths`, `flattenTodoItems`, `hashBytes`, `checkExternalChanges` all now have direct tests. ✅
- **Stress 5: dead test helpers** → 7 of 11 items consolidated away; remaining 4 are false positives. ✅

## Findings documented but not actioned

- ~~**`ExtractTodosSectionWithHeader` regex limitation**: when the
  target's Todos section is empty and the next section header
  immediately follows the mandatory blank line, the next-section regex
  `\\n\\n## ` does not match and `afterTodos` is returned empty.~~ **Fixed**
  in commit `f367de0` ("fix(core): match any markdown heading in
  NextSectionRegex, preserve trailing sections") before this summary was
  written. The next-section regex was widened to `(?m)^\n## ` and a direct
  test added in `pkg/core/extract_todos_test.go`. The
  `TestMergeIntoExistingTarget/empty_target_todos_section_gets_filled_with_new_items`
  case was tightened to assert that `## Notes` and the body after it
  survive the merge. This finding is preserved here only to record the
  resolution; new readers should look at the commit history, not this doc.
- **Phase 2c** (sub-struct grouping of `tuiModel`): see DEFERRED note above.

## What did not change

- `pkg/core/types.go`, `pkg/core/parser.go`, `pkg/core/journal.go`,
  `pkg/core/sync.go`, `pkg/core/utils.go`, `pkg/core/file.go`,
  `pkg/core/template_functions_*.go`, `pkg/generator/generator.go` —
  the library surface is unchanged. The cleanup is contained to
  `cmd/todoer/` (CLI/TUI), test organisation, and one structural
  improvement in `cmd/todoer/journal.go` (the `ProcessOptions` struct).
- Public API: `pkg/core` and `pkg/generator` are stable. The fingerprint
  spike behind `TODOER_FINGERPRINT` is untouched.
- The `tests/` directory layout is unchanged; only a `tests/README.md`
  was added.
