package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/SmolNero/quack/internal/sessions"
<<<<<<< HEAD
	"github.com/SmolNero/quack/internal/usage"
=======
>>>>>>> bb6579af04679718e00f1b5cb47321370f2e0353
)

const (
	refreshEvery      = 4 * time.Second
	usageRefreshEvery = time.Minute
)

var (
	pageStyle = lipgloss.NewStyle().Padding(0, 2)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4E5D57")).
			Background(lipgloss.Color("#E8EFEA")).
			Padding(0, 1)

	cardStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#C7D6CE")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6A7A73"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#A65C5C"))

	detailKeyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#73827B"))
	detailValStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4A5852"))
	usageCardStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#C7D6CE")).
			Padding(0, 1)
	usageLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50615A")).Bold(true)
	usageGoodStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#60976F"))
	usageWarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4944C"))
	usageLowStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#D66A70"))
	usageEmptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#D8DEDA"))

	confirmStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#B8CAC0")).
			Background(lipgloss.Color("#F5F9F6")).
			Foreground(lipgloss.Color("#50615A")).
			Padding(1, 2)
)

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Refresh key.Binding
	Cancel  key.Binding
	Quit    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Refresh, k.Cancel, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Refresh, k.Cancel, k.Quit}}
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "move")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "move")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Cancel:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "cancel session")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

type refreshMsg struct {
	sessions []sessions.ActiveSession
	err      error
	at       time.Time
}

type cancelMsg struct {
	err error
	pid int
}

type usageMsg struct {
	weekly usage.Weekly
	err    error
	at     time.Time
}

type tickMsg time.Time
type usageTickMsg time.Time

type Model struct {
	provider      sessions.Provider
	usageProvider usage.Provider
	table         table.Model
	help          help.Model

	sessions      []sessions.ActiveSession
	weekly        usage.Weekly
	loading       bool
	usageLoading  bool
	confirming    bool
	err           error
	usageErr      error
	status        string
	lastSync      time.Time
	usageLastSync time.Time
	width         int
	height        int
}

func NewModel(provider sessions.Provider, usageProvider usage.Provider) Model {
	columns := []table.Column{
		{Title: "PID", Width: 7},
		{Title: "Session name", Width: 36},
		{Title: "RAM (RSS)", Width: 10},
		{Title: "CPU", Width: 7},
		{Title: "Cache", Width: 10},
		{Title: "Age", Width: 9},
	}

	t := table.New(table.WithColumns(columns), table.WithRows(nil), table.WithFocused(true), table.WithHeight(9))

	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#D0DDD6")).
		BorderBottom(true).
		Bold(false).
		Foreground(lipgloss.Color("#65756E"))
	styles.Selected = styles.Selected.
		Foreground(lipgloss.Color("#42514A")).
		Background(lipgloss.Color("#DDE9E3")).
		Bold(false)
	t.SetStyles(styles)

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(lipgloss.Color("#6A7A73"))
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#8B9A93"))

	return Model{
		provider:      provider,
		usageProvider: usageProvider,
		table:         t,
		help:          h,
		loading:       true,
		usageLoading:  true,
		status:        "Loading active sessions...",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(refreshCmd(m.provider), usageRefreshCmd(m.usageProvider), tickCmd(), usageTickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.confirming {
		switch typed := msg.(type) {
		case tea.KeyMsg:
			switch typed.String() {
			case "y", "Y", "enter":
				sel := m.selected()
				if sel == nil {
					m.confirming = false
					return m, nil
				}
				m.confirming = false
				m.status = fmt.Sprintf("Cancelling PID %d...", sel.PID)
				return m, cancelCmd(m.provider, *sel)
			case "n", "N", "esc":
				m.confirming = false
				m.status = "Cancel aborted."
				return m, nil
			default:
				return m, nil
			}
		}
	}

	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.resizeTable()

	case tea.KeyMsg:
		switch typed.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			m.usageLoading = true
			m.usageErr = nil
			m.status = "Refreshing..."
			return m, tea.Batch(refreshCmd(m.provider), usageRefreshCmd(m.usageProvider))
		case "c":
			if len(m.sessions) > 0 {
				m.confirming = true
			}
		}

	case refreshMsg:
		selectedPID := 0
		if sel := m.selected(); sel != nil {
			selectedPID = sel.PID
		}

		m.loading = false
		m.lastSync = typed.at
		m.err = typed.err
		if typed.err != nil {
			m.status = "Refresh failed."
			break
		}
		m.sessions = typed.sessions
		m.table.SetRows(rowsFromSessions(typed.sessions))
		m.restoreCursor(selectedPID)
		if len(typed.sessions) == 0 {
			m.status = "No active OpenCode sessions."
		} else {
			m.status = fmt.Sprintf("%d active session%s.", len(typed.sessions), plural(len(typed.sessions)))
		}

	case cancelMsg:
		if typed.err != nil {
			m.err = typed.err
			m.status = "Cancel failed."
			break
		}
		m.err = nil
		m.status = fmt.Sprintf("PID %d cancelled.", typed.pid)
		m.loading = true
		return m, refreshCmd(m.provider)

	case usageMsg:
		m.usageLoading = false
		m.usageErr = typed.err
		if typed.err == nil {
			m.weekly = typed.weekly
			m.usageLastSync = typed.at
		}

	case tickMsg:
		if !m.loading && !m.confirming {
			m.loading = true
			return m, tea.Batch(refreshCmd(m.provider), tickCmd())
		}
		return m, tickCmd()

	case usageTickMsg:
		m.usageLoading = true
		m.usageErr = nil
		return m, tea.Batch(usageRefreshCmd(m.usageProvider), usageTickCmd())
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	count := len(m.sessions)
	header := headerStyle.Render(fmt.Sprintf("quack  •  %d active  •  %s RAM", count, formatBytes(totalMemory(m.sessions))))

	stamp := "never"
	if !m.lastSync.IsZero() {
		stamp = m.lastSync.Format("15:04:05")
	}

	top := lipgloss.JoinHorizontal(lipgloss.Top, header, "  ", statusStyle.Render("last refresh "+stamp))

	status := statusStyle.Render(m.status)
	if m.err != nil {
		status = errorStyle.Render(m.err.Error())
	}

	helpText := m.help.View(keys)
	detail := m.detailView()
	weekly := m.usageView()
	confirm := ""
	if m.confirming {
		sel := m.selected()
		label := "selected session"
		if sel != nil {
			label = fmt.Sprintf("%q (PID %d)", trimText(sel.Title, 36), sel.PID)
		}
		confirm = confirmStyle.Render(fmt.Sprintf("Cancel %s? [y/N]", label))
	}

	reserve := lipgloss.Height(top) + lipgloss.Height(weekly) + lipgloss.Height(status) + lipgloss.Height(helpText)
	if confirm != "" {
		reserve += lipgloss.Height(confirm)
	}

	showDetail := true
	tableHeight := m.height - reserve - lipgloss.Height(detail)
	if tableHeight < 6 {
		showDetail = false
		tableHeight = m.height - reserve
	}
	if tableHeight < 3 {
		tableHeight = 3
	}
	m.table.SetHeight(tableHeight)
	m.resizeTable()

	body := m.table.View()
	if len(m.sessions) == 0 && !m.loading {
		body = cardStyle.Render("No running OpenCode process found.")
	}

	sections := []string{top, weekly, body}
	if showDetail {
		sections = append(sections, detail)
	}
	sections = append(sections, status, helpText)
	if confirm != "" {
		sections = append(sections, confirm)
	}

	view := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return pageStyle.Width(m.width).Render(view)
}

func (m *Model) resizeTable() {
	width := m.width - 6
	if width < 40 {
		width = 40
	}

	pidW := 7
	memoryW := 10
	cpuW := 7
	cacheW := 10
	ageW := 9
	sessionW := width - pidW - memoryW - cpuW - cacheW - ageW - 12
	if sessionW < 20 {
		sessionW = 20
	}

	cols := m.table.Columns()
	cols[0].Width = pidW
	cols[1].Width = sessionW
	cols[2].Width = memoryW
	cols[3].Width = cpuW
	cols[4].Width = cacheW
	cols[5].Width = ageW
	m.table.SetColumns(cols)
}

func rowsFromSessions(active []sessions.ActiveSession) []table.Row {
	rows := make([]table.Row, 0, len(active))
	for _, item := range active {
		title := item.Title
		if title == "" {
			title = "Unidentified session"
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", item.PID),
			title,
			formatBytes(item.Memory),
			fmt.Sprintf("%.1f%%", item.CPU),
			formatTokens(item.CacheRead + item.CacheWrite),
			time.Since(item.StartedAt).Round(time.Second).String(),
		})
	}

	return rows
}

func (m Model) selected() *sessions.ActiveSession {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.sessions) {
		return nil
	}
	return &m.sessions[idx]
}

func (m *Model) restoreCursor(selectedPID int) {
	if len(m.sessions) == 0 {
		m.table.SetCursor(0)
		return
	}

	if selectedPID > 0 {
		for i, item := range m.sessions {
			if item.PID == selectedPID {
				m.table.SetCursor(i)
				return
			}
		}
	}

	idx := m.table.Cursor()
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.sessions) {
		idx = len(m.sessions) - 1
	}
	m.table.SetCursor(idx)
}

func (m Model) detailView() string {
	sel := m.selected()
	if sel == nil {
		return cardStyle.Render("Select a row to see details.")
	}

	session := sel.SessionID
	if session == "" {
		session = "not available"
	}
	title := sel.Title
	if title == "" {
		title = "Unidentified session"
	}

	updated := "unknown"
	if !sel.UpdatedAt.IsZero() {
		updated = sel.UpdatedAt.Format("2006-01-02 15:04:05")
	}

	line := func(k, v string) string {
		return detailKeyStyle.Render(k+":") + " " + detailValStyle.Render(v)
	}

	return cardStyle.Render(strings.Join([]string{
		line("title", title),
		line("session", session),
		line("directory", sel.Directory),
		line("resources", fmt.Sprintf("%s RSS | %.1f%% CPU", formatBytes(sel.Memory), sel.CPU)),
		line("tokens", fmt.Sprintf("%s input | %s output | %s reasoning", formatTokens(sel.Input), formatTokens(sel.Output), formatTokens(sel.Reasoning))),
		line("cache", fmt.Sprintf("%s read | %s write", formatTokens(sel.CacheRead), formatTokens(sel.CacheWrite))),
		line("updated", updated),
		line("command", trimText(sel.Command, 64)),
	}, "\n"))
}

func (m Model) usageView() string {
	width := m.width - 10
	if width < 24 {
		width = 24
	}

	if m.usageLastSync.IsZero() {
		message := "Weekly usage: loading..."
		if m.usageErr != nil {
			message = trimText("Weekly usage unavailable: "+m.usageErr.Error(), width)
		}
		return usageCardStyle.Width(width).Render(message)
	}

	remaining := m.weekly.RemainingPercent()
	percent := int(remaining + 0.5)
	levelStyle := usageGoodStyle
	if remaining <= 20 {
		levelStyle = usageLowStyle
	} else if remaining <= 50 {
		levelStyle = usageWarnStyle
	}

	left := usageLabelStyle.Render("Weekly usage") + "  " + levelStyle.Render(fmt.Sprintf("%d%% remaining", percent))
	reset := "reset time unavailable"
	if !m.weekly.ResetAt.IsZero() {
		reset = "resets " + m.weekly.ResetAt.In(time.Local).Format("Mon Jan 2, 3:04 PM")
	}
	checked := "checked " + m.usageLastSync.In(time.Local).Format("3:04 PM")
	if m.usageErr != nil {
		checked = "update failed"
	} else if m.usageLoading {
		checked = "updating"
	}
	right := statusStyle.Render(reset + "  •  " + checked)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	summary := left + "\n" + right
	if gap >= 2 {
		summary = left + strings.Repeat(" ", gap) + right
	}

	barWidth := width
	if barWidth > 64 {
		barWidth = 64
	}
	filled := int((remaining/100)*float64(barWidth) + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	bar := levelStyle.Render(strings.Repeat("█", filled)) + usageEmptyStyle.Render(strings.Repeat("░", barWidth-filled))

	return usageCardStyle.Width(width).Render(summary + "\n" + bar)
}

func refreshCmd(provider sessions.Provider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		active, err := provider.ListActive(ctx)
		return refreshMsg{sessions: active, err: err, at: time.Now()}
	}
}

func usageRefreshCmd(provider usage.Provider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()

		weekly, err := provider.Weekly(ctx)
		return usageMsg{weekly: weekly, err: err, at: time.Now()}
	}
}

func cancelCmd(provider sessions.Provider, active sessions.ActiveSession) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		err := provider.Cancel(ctx, active)
		return cancelMsg{pid: active.PID, err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshEvery, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func usageTickCmd() tea.Cmd {
	return tea.Tick(usageRefreshEvery, func(t time.Time) tea.Msg {
		return usageTickMsg(t)
	})
}

func totalMemory(items []sessions.ActiveSession) int64 {
	var total int64
	for _, item := range items {
		total += item.Memory
	}
	return total
}

func trimText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit < 5 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func formatBytes(value int64) string {
	const unit = int64(1024)
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}

	div, exp := unit, 0
	for n := value / unit; n >= unit && exp < 3; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(value)/float64(div), "KMGT"[exp])
}

func formatTokens(value int64) string {
	switch {
	case value >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(value)/1_000_000_000)
	case value >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	case value >= 1_000:
		return fmt.Sprintf("%.1fK", float64(value)/1_000)
	default:
		return fmt.Sprintf("%d", value)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
