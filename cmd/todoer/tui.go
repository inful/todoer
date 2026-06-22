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

	b.WriteString(tuiTheme.header.Render("todoer tui"))
	b.WriteString("\n")
	b.WriteString(tuiTheme.filePath.Render("File: "+m.journalPath) + "\n")
	b.WriteString("State: " + stateStyle.Render(stateText) + "\n")
	b.WriteString("Filter: " + tuiTheme.filePath.Render(m.filterQuery) + "\n\n")

	filtered := m.filteredItems()
	if len(filtered) == 0 {
		b.WriteString(tuiTheme.empty.Render(m.emptyStateMessage()) + "\n")
	} else {
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
	}

	b.WriteString("\n")
	if m.inputMode {
		b.WriteString(tuiTheme.inputLabel.Render("Add todo: ") + m.inputText + "\n")
	} else if m.filterMode {
		b.WriteString(tuiTheme.inputLabel.Render("Filter todos: ") + m.filterQuery + "\n")
	} else {
		b.WriteString(tuiTheme.keyHelp.Render(tuiHelpText()) + "\n")
	}

	statusStyle := tuiTheme.statusOK
	lowerStatus := strings.ToLower(m.status)
	if strings.Contains(lowerStatus, "failed") || strings.Contains(lowerStatus, "error") {
		statusStyle = tuiTheme.statusErr
	} else if strings.Contains(lowerStatus, "external") || strings.Contains(lowerStatus, "blocked") {
		statusStyle = tuiTheme.statusWarn
	}
	b.WriteString("Status: " + statusStyle.Render(m.status) + "\n")

	return b.String()
}
