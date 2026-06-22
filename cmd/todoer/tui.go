package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inful/todoer/pkg/core"
)

type tuiCmd struct {
	Backup bool `help:"Preserve a .bak copy of the source journal when creating today's journal for the first time"`
}

type tuiItem struct {
	item  *core.TodoItem
	depth int
}

type tuiTickMsg struct{}

type tuiModel struct {
	journalPath string
	config      *Config
	today       string

	beforeTodos string
	afterTodos  string
	journal     *core.TodoJournal
	todayDay    *core.DaySection
	displayDay  *core.DaySection // the day currently shown in the view; sticky after first set

	items    []tuiItem
	selected int

	dirty           bool
	fileHash        string
	externalChanged bool
	status          string

	inputMode bool
	inputText string

	filterMode  bool
	filterQuery string
}

func (cmd *tuiCmd) Run(cli *cliOptions, config *Config, baseLogger *Logger) error {
	rootDir, templateFile := sharedPaths(cli, config)
	logger := baseLogger.WithMode(ModeQuiet)

	if err := cmdNewWithOptions(rootDir, templateFile, false, cmd.Backup, config, logger); err != nil {
		return fmt.Errorf("failed to prepare today's journal for tui: %w", err)
	}

	today := time.Now().Format(core.DateFormat)
	journalPath := buildJournalPath(rootDir, today)

	model, err := newTUIModel(journalPath, today, config)
	if err != nil {
		return err
	}

	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func newTUIModel(journalPath, today string, config *Config) (tuiModel, error) {
	m := tuiModel{
		journalPath: journalPath,
		config:      config,
		today:       today,
		status:      tuiInitStatus(),
	}

	if err := m.reloadFromDisk(); err != nil {
		return tuiModel{}, err
	}

	return m, nil
}

func (m tuiModel) Init() tea.Cmd {
	return tuiTickCmd()
}

func tuiTickCmd() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
		return tuiTickMsg{}
	})
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tuiTickMsg:
		m.checkExternalChanges()
		return m, tuiTickCmd()
	case tea.KeyMsg:
		if m.inputMode {
			return m.updateInputMode(msg)
		}
		if m.filterMode {
			return m.updateFilterMode(msg)
		}
		return m.updateNormalMode(msg)
	}

	return m, nil
}

func (m tuiModel) View() string {
	var b strings.Builder
	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	b.WriteString(m.viewItems())
	b.WriteString("\n")
	b.WriteString(m.viewInputLine())
	b.WriteString("\n")
	b.WriteString(m.viewStatus())
	return b.String()
}

// viewHeader renders the title bar, file path, state, and filter line.
// It uses tuiTheme styles; the test surface is the textual content
// (State:, Filter:, File:), which TestView_StateMarkers asserts on.
func (m tuiModel) viewHeader() string {
	stateText := "clean"
	stateStyle := tuiTheme.stateClean
	if m.dirty {
		stateText = "dirty"
		stateStyle = tuiTheme.stateDirty
	}
	if m.externalChanged {
		stateText += ", external-changed"
		stateStyle = tuiTheme.stateChanged
	}

	var b strings.Builder
	b.WriteString(tuiTheme.header.Render("todoer tui"))
	b.WriteString("\n")
	b.WriteString(tuiTheme.filePath.Render("File: "+m.journalPath) + "\n")
	b.WriteString("State: " + stateStyle.Render(stateText) + "\n")
	b.WriteString("Filter: " + tuiTheme.filePath.Render(m.filterQuery) + "\n")
	return b.String()
}

// viewItems renders the todo list, applying cursor and selection styles.
// Returns the empty-state placeholder if no items match the filter.
func (m tuiModel) viewItems() string {
	filtered := m.filteredItems()
	if len(filtered) == 0 {
		return tuiTheme.empty.Render(m.emptyStateMessage()) + "\n"
	}

	var b strings.Builder
	for i, entry := range filtered {
		cursor := " "
		if i == m.selected {
			cursor = ">"
		}

		check := "[ ]"
		if entry.item.Completed {
			check = "[x]"
		}

		indent := strings.Repeat("  ", entry.depth)
		line := fmt.Sprintf("%s %s%s %s", cursor, indent, check, entry.item.Text)
		if i == m.selected {
			b.WriteString(tuiTheme.selected.Render(line) + "\n")
		} else {
			b.WriteString(tuiTheme.item.Render(line) + "\n")
		}
	}
	return b.String()
}

// viewInputLine renders the bottom-of-view prompt: add-todo prompt
// when in input mode, filter prompt when in filter mode, and the
// static help line in normal mode. The help line is driven by
// tuiKeymap so adding a key is a one-place edit.
func (m tuiModel) viewInputLine() string {
	switch {
	case m.inputMode:
		return tuiTheme.inputLabel.Render("Add todo: ") + m.inputText + "\n"
	case m.filterMode:
		return tuiTheme.inputLabel.Render("Filter todos: ") + m.filterQuery + "\n"
	default:
		return tuiTheme.keyHelp.Render(tuiHelpText()) + "\n"
	}
}

// viewStatus renders the status line. The style is chosen from the
// status text: "failed" / "error" use the error style, "external" /
// "blocked" use the warn style, anything else uses the ok style.
func (m tuiModel) viewStatus() string {
	statusStyle := tuiTheme.statusOK
	lowerStatus := strings.ToLower(m.status)
	switch {
	case strings.Contains(lowerStatus, "failed") || strings.Contains(lowerStatus, "error"):
		statusStyle = tuiTheme.statusErr
	case strings.Contains(lowerStatus, "external") || strings.Contains(lowerStatus, "blocked"):
		statusStyle = tuiTheme.statusWarn
	}
	return "Status: " + statusStyle.Render(m.status) + "\n"
}
