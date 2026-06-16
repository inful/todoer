package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

var tuiTheme = tuiStyles{
	header: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62")).
		Padding(0, 1),
	filePath: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	stateClean: lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true),
	stateDirty: lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true),
	stateChanged: lipgloss.NewStyle().
		Foreground(lipgloss.Color("203")).
		Bold(true),
	selected: lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("57")).
		Bold(true),
	item:     lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	keyHelp:  lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	statusOK: lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	statusWarn: lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true),
	statusErr: lipgloss.NewStyle().
		Foreground(lipgloss.Color("203")).
		Bold(true),
	inputLabel: lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true),
	empty: lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true),
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
		status:      "Ready. j/k move, space toggle, / filter, a add, d delete, s save, r reload, q quit",
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

func (m tuiModel) updateInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputMode = false
		m.inputText = ""
		m.status = "Add cancelled"
	case "enter":
		text := strings.TrimSpace(m.inputText)
		if text == "" {
			m.status = "Cannot add empty todo"
			return m, nil
		}
		if m.todayDay == nil {
			m.todayDay = findOrCreateDaySection(m.journal, m.today)
		}
		m.todayDay.Items = append(m.todayDay.Items, &core.TodoItem{
			Completed:   false,
			Text:        text,
			SubItems:    []*core.TodoItem{},
			BulletLines: []string{},
		})
		m.refreshItems()
		if len(m.items) > 0 {
			m.selected = len(m.items) - 1
		}
		m.dirty = true
		m.inputMode = false
		m.inputText = ""
		m.status = "Todo added"
	case "backspace":
		if len(m.inputText) > 0 {
			m.inputText = m.inputText[:len(m.inputText)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			m.inputText += string(msg.Runes)
		}
	}

	return m, nil
}

func (m tuiModel) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredItems()

	switch msg.String() {
	case "ctrl+c", "q":
		if msg.String() == "q" && m.dirty {
			if err := m.saveToDisk(); err != nil {
				m.status = fmt.Sprintf("Save failed: %v", err)
				return m, nil
			}
			m.status = "Saved"
		}
		return m, tea.Quit
	case "j", "down":
		if m.selected < len(filtered)-1 {
			m.selected++
		}
	case "k", "up":
		if m.selected > 0 {
			m.selected--
		}
	case "/":
		m.filterMode = true
		m.status = "Type filter text, Enter to apply, Esc to cancel"
	case " ", "x":
		if len(filtered) == 0 {
			m.status = "No todo selected"
			return m, nil
		}
		if m.isReadOnlyView() {
			m.status = "Cannot edit carryover items"
			return m, nil
		}
		entry := filtered[m.selected]
		entry.item.Completed = !entry.item.Completed
		m.dirty = true
		m.status = "Todo toggled"
	case "d":
		if len(filtered) == 0 {
			m.status = "No todo selected"
			return m, nil
		}
		if m.isReadOnlyView() {
			m.status = "Cannot edit carryover items"
			return m, nil
		}
		target := filtered[m.selected].item
		m.journal.Days = removeItemFromDays(m.journal.Days, target)
		m.todayDay = findDaySection(m.journal, m.today)
		m.refreshItems()
		filtered = m.filteredItems()
		if m.selected >= len(filtered) && len(filtered) > 0 {
			m.selected = len(filtered) - 1
		}
		if len(filtered) == 0 {
			m.selected = 0
		}
		m.dirty = true
		m.status = "Todo deleted"
	case "a":
		m.inputMode = true
		m.inputText = ""
		m.status = "Type todo text, Enter to add, Esc to cancel"
	case "s":
		if err := m.saveToDisk(); err != nil {
			m.status = fmt.Sprintf("Save failed: %v", err)
		} else {
			m.status = "Saved"
		}
	case "r":
		if err := m.reloadFromDisk(); err != nil {
			m.status = fmt.Sprintf("Reload failed: %v", err)
		} else {
			m.status = "Reloaded from disk"
		}
	case "c":
		m.filterQuery = ""
		m.selected = 0
		m.status = "Filter cleared"
	}

	return m, nil
}

func (m tuiModel) updateFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterMode = false
		m.status = "Filter cancelled"
	case "enter":
		m.filterMode = false
		m.selected = 0
		if m.filterQuery == "" {
			m.status = "Filter cleared"
		} else {
			m.status = fmt.Sprintf("Filter applied: %q", m.filterQuery)
		}
	case "backspace":
		if len(m.filterQuery) > 0 {
			m.filterQuery = m.filterQuery[:len(m.filterQuery)-1]
		}
		m.selected = 0
	default:
		if len(msg.Runes) > 0 {
			m.filterQuery += string(msg.Runes)
			m.selected = 0
		}
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
		b.WriteString(tuiTheme.empty.Render("(No todos in today's section)") + "\n")
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
		b.WriteString(tuiTheme.keyHelp.Render("Keys: j/k move | space toggle | / filter | c clear filter | a add | d delete | s save | r reload | q quit") + "\n")
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

func (m *tuiModel) refreshItems() {
	m.items = make([]tuiItem, 0)
	displayDay := m.displayDaySection()
	if displayDay == nil {
		return
	}
	flattenTodoItems(displayDay.Items, 0, &m.items)
	filtered := m.filteredItems()
	if m.selected >= len(filtered) {
		m.selected = max(0, len(filtered)-1)
	}
}

func (m *tuiModel) displayDaySection() *core.DaySection {
	if m.todayDay != nil {
		return m.todayDay
	}

	for i := len(m.journal.Days) - 1; i >= 0; i-- {
		day := m.journal.Days[i]
		if day != nil && len(day.Items) > 0 {
			return day
		}
	}

	return nil
}

// isReadOnlyView reports whether the currently displayed day is not today's
// section. When the model falls back to showing a carryover day, edits to those
// items would silently mutate a different journal; the carryover view is
// read-only and the user is asked to run `new` to bring items into today.
func (m *tuiModel) isReadOnlyView() bool {
	return m.displayDaySection() != m.todayDay
}

func (m tuiModel) filteredItems() []tuiItem {
	if strings.TrimSpace(m.filterQuery) == "" {
		return m.items
	}

	needle := strings.ToLower(strings.TrimSpace(m.filterQuery))
	filtered := make([]tuiItem, 0, len(m.items))
	for _, entry := range m.items {
		if strings.Contains(strings.ToLower(entry.item.Text), needle) {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

func flattenTodoItems(items []*core.TodoItem, depth int, out *[]tuiItem) {
	for _, item := range items {
		if item == nil {
			continue
		}
		*out = append(*out, tuiItem{item: item, depth: depth})
		flattenTodoItems(item.SubItems, depth+1, out)
	}
}

func (m *tuiModel) reloadFromDisk() error {
	content, err := os.ReadFile(m.journalPath)
	if err != nil {
		return fmt.Errorf("failed to read journal file %s: %w", m.journalPath, err)
	}

	beforeTodos, todosSection, afterTodos, err := core.ExtractTodosSectionWithHeader(string(content), m.config.TodosHeader)
	if err != nil {
		return fmt.Errorf("failed to locate todos section in %s: %w", m.journalPath, err)
	}

	journal, err := core.ParseTodosSection(todosSection)
	if err != nil {
		return fmt.Errorf("failed to parse todos section in %s: %w", m.journalPath, err)
	}

	m.beforeTodos = beforeTodos
	m.afterTodos = afterTodos
	m.journal = journal
	m.todayDay = findDaySection(journal, m.today)
	m.refreshItems()
	if m.selected >= len(m.items) {
		m.selected = max(0, len(m.items)-1)
	}
	m.dirty = false
	m.externalChanged = false
	m.fileHash = hashBytes(content)

	return nil
}

func (m *tuiModel) saveToDisk() error {
	if !m.dirty {
		return nil
	}

	currentContent, err := os.ReadFile(m.journalPath)
	if err != nil {
		return fmt.Errorf("failed to read journal for conflict check: %w", err)
	}

	currentHash := hashBytes(currentContent)
	if currentHash != m.fileHash {
		m.externalChanged = true
		return fmt.Errorf("file changed externally, reload before saving")
	}

	newTodos := core.JournalToString(m.journal)
	newContent := []byte(m.beforeTodos + newTodos + m.afterTodos)

	if err := safeWriteFile(m.journalPath, newContent, FilePermissions); err != nil {
		return fmt.Errorf("failed to write updated journal file %s: %w", m.journalPath, err)
	}

	m.fileHash = hashBytes(newContent)
	m.dirty = false
	m.externalChanged = false

	return nil
}

func (m *tuiModel) checkExternalChanges() {
	content, err := os.ReadFile(m.journalPath)
	if err != nil {
		return
	}

	newHash := hashBytes(content)
	if newHash == m.fileHash {
		return
	}

	if m.dirty {
		m.externalChanged = true
		m.status = "External change detected. Save blocked; press r to reload."
		return
	}

	if err := m.reloadFromDisk(); err == nil {
		m.status = "External change detected and reloaded"
	}
}

func findDaySection(journal *core.TodoJournal, date string) *core.DaySection {
	if journal == nil {
		return nil
	}
	for _, day := range journal.Days {
		if day != nil && day.Date == date {
			return day
		}
	}
	return nil
}

func removeItemFromDays(days []*core.DaySection, target *core.TodoItem) []*core.DaySection {
	for _, day := range days {
		if day == nil {
			continue
		}
		day.Items, _ = removeItemRecursive(day.Items, target)
	}
	return days
}

func removeItemRecursive(items []*core.TodoItem, target *core.TodoItem) ([]*core.TodoItem, bool) {
	for i, item := range items {
		if item == nil {
			continue
		}
		if item == target {
			return append(items[:i], items[i+1:]...), true
		}

		updatedSub, removed := removeItemRecursive(item.SubItems, target)
		if removed {
			item.SubItems = updatedSub
			return items, true
		}
	}
	return items, false
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}
