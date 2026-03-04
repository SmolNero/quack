package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"quack/internal/sessions"
)

const refreshEvery = 4 * time.Second
const duckEvery = 220 * time.Millisecond

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

	confirmStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#B8CAC0")).
			Background(lipgloss.Color("#F5F9F6")).
			Foreground(lipgloss.Color("#50615A")).
			Padding(1, 2)

	duckBodyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#D64C4C"))
	duckWingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#B43C3C"))
	duckBeakStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E8B44D"))
	duckEyeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#1F1F1F"))
)

var duckFrames = [2]string{
	"      __\n" +
		"  ___(e )>\n" +
		" /   w   \\\n" +
		"(  wwww   )\n" +
		" \\_______/",
	"      __\n" +
		"  ___(e )>\n" +
		" /  wwww \\\n" +
		"(    w    )\n" +
		" \\_______/",
}

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

type tickMsg time.Time
type duckTickMsg time.Time

type Model struct {
	provider sessions.Provider
	table    table.Model
	help     help.Model

	sessions   []sessions.ActiveSession
	loading    bool
	confirming bool
	err        error
	status     string
	lastSync   time.Time
	width      int
	height     int
	duckFrame  int
}

func NewModel(provider sessions.Provider) Model {
	columns := []table.Column{
		{Title: "PID", Width: 7},
		{Title: "Session", Width: 18},
		{Title: "Where", Width: 30},
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

	return Model{provider: provider, table: t, help: h, loading: true, status: "Loading active sessions..."}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(refreshCmd(m.provider), tickCmd(), duckTickCmd())
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
			m.status = "Refreshing..."
			return m, refreshCmd(m.provider)
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

	case tickMsg:
		if !m.loading && !m.confirming {
			m.loading = true
			return m, tea.Batch(refreshCmd(m.provider), tickCmd())
		}
		return m, tickCmd()

	case duckTickMsg:
		m.duckFrame = (m.duckFrame + 1) % len(duckFrames)
		return m, duckTickCmd()
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
	places := locationCount(m.sessions)
	header := headerStyle.Render(fmt.Sprintf("quack  •  %d active  •  %d location%s", count, places, plural(places)))

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
	duck := m.duckView()
	confirm := ""
	if m.confirming {
		sel := m.selected()
		label := "selected session"
		if sel != nil {
			label = fmt.Sprintf("PID %d in %s", sel.PID, shortPath(sel.Directory))
		}
		confirm = confirmStyle.Render(fmt.Sprintf("Cancel %s? [y/N]", label))
	}

	reserve := lipgloss.Height(top) + lipgloss.Height(status) + lipgloss.Height(helpText)
	if confirm != "" {
		reserve += lipgloss.Height(confirm)
	}
	if duck != "" {
		reserve += lipgloss.Height(duck)
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

	sections := []string{top, body}
	if showDetail {
		sections = append(sections, detail)
	}
	sections = append(sections, status, helpText)
	if duck != "" {
		rightWidth := m.width - 4
		if rightWidth < 20 {
			rightWidth = 20
		}
		sections = append(sections, lipgloss.PlaceHorizontal(rightWidth, lipgloss.Right, duck))
	}
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
	ageW := 9
	sessionW := 18
	whereW := width - pidW - ageW - sessionW - 8
	if whereW < 20 {
		whereW = 20
	}

	cols := m.table.Columns()
	cols[0].Width = pidW
	cols[1].Width = sessionW
	cols[2].Width = whereW
	cols[3].Width = ageW
	m.table.SetColumns(cols)
}

func rowsFromSessions(active []sessions.ActiveSession) []table.Row {
	rows := make([]table.Row, 0, len(active))
	for _, item := range active {
		sid := item.SessionID
		if sid == "" {
			sid = "unknown"
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", item.PID),
			trimID(sid),
			shortPath(item.Directory),
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
		session = "unknown"
	}
	title := sel.Title
	if title == "" {
		title = "No title available"
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
		line("updated", updated),
		line("command", trimText(sel.Command, 64)),
	}, "\n"))
}

func refreshCmd(provider sessions.Provider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		active, err := provider.ListActive(ctx)
		return refreshMsg{sessions: active, err: err, at: time.Now()}
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

func duckTickCmd() tea.Cmd {
	return tea.Tick(duckEvery, func(t time.Time) tea.Msg {
		return duckTickMsg(t)
	})
}

func locationCount(items []sessions.ActiveSession) int {
	if len(items) == 0 {
		return 0
	}
	uniq := make(map[string]struct{}, len(items))
	for _, item := range items {
		uniq[item.Directory] = struct{}{}
	}
	return len(uniq)
}

func shortPath(path string) string {
	if path == "" {
		return "unknown"
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		path = strings.Replace(path, home, "~", 1)
	}
	return trimText(path, 42)
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

func trimID(id string) string {
	return trimText(id, 16)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func (m Model) duckView() string {
	if len(duckFrames) == 0 {
		return ""
	}
	frame := duckFrames[m.duckFrame%len(duckFrames)]
	return renderDuckFrame(frame)
}

func renderDuckFrame(frame string) string {
	lines := strings.Split(frame, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		var b strings.Builder
		for i := 0; i < len(line); i++ {
			ch := line[i]
			switch ch {
			case ' ':
				b.WriteByte(ch)
			case 'e':
				b.WriteString(duckEyeStyle.Render("o"))
			case '>':
				b.WriteString(duckBeakStyle.Render(string(ch)))
			case 'w':
				b.WriteString(duckWingStyle.Render("~"))
			default:
				b.WriteString(duckBodyStyle.Render(string(ch)))
			}
		}
		out = append(out, b.String())
	}

	return strings.Join(out, "\n")
}
