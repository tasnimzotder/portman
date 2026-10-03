package tui

import (
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/style"
)

// tickMsg is sent on each refresh interval.
type tickMsg time.Time

// listenersMsg carries scan results.
type listenersMsg struct {
	listeners []model.Listener
	err       error
}

// portMsg carries a single port scan result.
type portMsg struct {
	listener *model.Listener
	err      error
}

// IsTerminal requires both input and output to be actual terminals.
func IsTerminal() bool {
	return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}

// tick returns a Cmd that sends a tickMsg after the given interval.
func tick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fullscreenView creates a View with AltScreen enabled.
func fullscreenView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// portTableColumns defines the standard columns for port listing tables.
var portTableColumns = []table.Column{
	{Title: "PORT", Width: 8},
	{Title: "PROTO", Width: 7},
	{Title: "PID", Width: 8},
	{Title: "USER", Width: 10},
	{Title: "COMMAND", Width: 24},
	{Title: "CONNS", Width: 7},
	{Title: "UPTIME", Width: 12},
	{Title: "ADDRESS", Width: 24},
}

// tableWidth computes the total width from portTableColumns plus cell padding.
func tableWidth() int {
	w := 0
	for _, col := range portTableColumns {
		w += col.Width + 2
	}
	return w
}

// newPortTable creates a styled table for port listings.
func newPortTable(height int) table.Model {
	if height <= 0 {
		height = 20
	}

	t := table.New(
		table.WithColumns(portTableColumns),
		table.WithFocused(true),
		table.WithHeight(height),
		table.WithWidth(tableWidth()),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(style.ColorSubtle).
		BorderBottom(true).
		Bold(true).
		Foreground(style.ColorDim)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(style.ColorAccent).
		Bold(false)
	t.SetStyles(s)

	return t
}

// buildPortRows converts a slice of listeners into table rows.
func buildPortRows(listeners []model.Listener) []table.Row {
	rows := make([]table.Row, 0, len(listeners))
	for _, l := range listeners {
		pid := "-"
		if l.PID > 0 {
			pid = strconv.Itoa(l.PID)
		}

		command := l.ProcessName()
		if command == "unknown" {
			command = "-"
		}

		user := l.ProcessUser()

		uptime := "-"
		if l.Process != nil && l.Process.UptimeSeconds > 0 {
			uptime = output.FormatDuration(l.Process.UptimeSeconds)
		}

		rows = append(rows, table.Row{
			strconv.Itoa(l.Port),
			l.Protocol,
			pid,
			user,
			command,
			strconv.Itoa(l.ConnectionCount),
			uptime,
			l.Address,
		})
	}
	return rows
}

// filterByProtocol delegates to model.FilterByProtocol.
func filterByProtocol(listeners []model.Listener, tcpOnly, udpOnly bool) []model.Listener {
	return model.FilterByProtocol(listeners, tcpOnly, udpOnly)
}

// helpBar delegates to style.HelpBar.
func helpBar(items ...string) string {
	return style.HelpBar(items...)
}

// joinHelp joins help items with a separator.
func joinHelp(items ...string) string {
	return strings.Join(items, " · ")
}
