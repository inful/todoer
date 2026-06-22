package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// tuiStyles groups the lipgloss.Style values used by the TUI view.
// Keeping them in one struct lets the View method reference a single
// theme value (tuiTheme) and makes colour changes a one-place edit.
type tuiStyles struct {
	header       lipgloss.Style
	filePath     lipgloss.Style
	stateClean   lipgloss.Style
	stateDirty   lipgloss.Style
	stateChanged lipgloss.Style
	selected     lipgloss.Style
	item         lipgloss.Style
	keyHelp      lipgloss.Style
	statusOK     lipgloss.Style
	statusWarn   lipgloss.Style
	statusErr    lipgloss.Style
	inputLabel   lipgloss.Style
	empty        lipgloss.Style
}

// tuiTheme is the single source of truth for TUI colours. The View
// method references tuiTheme directly; tests and helpers must not
// hard-code styles of their own.
var tuiTheme = tuiStyles{
	header: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62")).
		Padding(0, 1),
	filePath:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	stateClean:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
	stateDirty:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
	stateChanged: lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
	selected: lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("57")).
		Bold(true),
	item:       lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	keyHelp:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	statusOK:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	statusWarn: lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
	statusErr:  lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
	inputLabel: lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true),
	empty:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true),
}

// tuiKeymap is the single source of truth for the TUI's normal-mode
// keybindings. The same table drives both the bottom-of-view help
// line and the initial status message, so adding a key only requires
// updating this slice — drift between the three previously-hardcoded
// strings is no longer possible.
var tuiKeymap = []struct {
	keys  string
	label string
}{
	{"j/k", "move"},
	{"space", "toggle"},
	{"/", "filter"},
	{"c", "clear filter"},
	{"a", "add"},
	{"d", "delete"},
	{"s", "save"},
	{"r", "reload"},
	{"q", "quit"},
}

// tuiHelpText renders the bottom-of-view help line and is also the
// base of the initial status message.
func tuiHelpText() string {
	var b strings.Builder
	b.WriteString("Keys: ")
	for i, k := range tuiKeymap {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(k.keys)
		b.WriteString(" ")
		b.WriteString(k.label)
	}
	return b.String()
}

// tuiInitStatus is the message shown in the model status field on
// startup. It embeds the same key list as tuiHelpText so the
// initial render is consistent with the help line.
func tuiInitStatus() string {
	return "Ready. " + strings.TrimPrefix(tuiHelpText(), "Keys: ")
}
