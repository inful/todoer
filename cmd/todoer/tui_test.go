// Package main wires the todoer CLI and TUI. Tests for the TUI are
// split across multiple mode-aligned files:
//
//   - tui_actions_test.go       key dispatch (normal, input, filter modes)
//   - tui_carryover_test.go     sticky carryover view, display-day semantics
//   - tui_filter_test.go        filter-mode behaviour
//   - tui_input_test.go         input-mode behaviour (Enter, Esc, backspace)
//   - tui_io_test.go            hashBytes, checkExternalChanges, flattenTodoItems
//   - tui_model_test.go         tuiModel state, refreshItems, isCarryoverView
//   - tui_normal_test.go        normal-mode navigation, toggling, deletion
//   - tui_scroll_test.go        viewport scrolling (offset, ensureSelectedVisible)
//   - tui_view_test.go          View() decomposition (header, items, input, status)
//
// This file exists only so `go test ./cmd/todoer` includes the
// package — the actual coverage is in the files above. Without it,
// the package would still build (main has no test files) but the
// test binary would not pick up any tui_*_test.go siblings.
//
// See cmd/todoer/tui_styles.go for the styles/theme/keymap that
// drive the View output the view tests assert against.
package main
