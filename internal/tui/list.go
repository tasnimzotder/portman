package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/tasnimzotder/portman/internal/kill"
	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/scanner"
)

var sortOptions = []string{"port", "pid", "user", "conns", "uptime"}

type listView int

const (
	viewList listView = iota
	viewDetail
	viewPIDPorts
	viewKillConfirm
	viewKilling
	viewKillResult
)

type pidPortsMsg struct {
	listeners []model.Listener
	pid       int
	err       error
}

type ListModel struct {
	table     table.Model
	scanner   scanner.Scanner
	sortBy    string
	tcpOnly   bool
	udpOnly   bool
	listeners []model.Listener
	err       error
	loaded    bool

	// Drill-down state
	view     listView
	detail   *model.Listener
	detailOK bool

	// PID ports view
	pidPorts []model.Listener
	pidNum   int

	// Kill state
	killResult string
	spinner    spinner.Model

	// Double-q to quit
	lastQPress time.Time
}

func NewListModel(s scanner.Scanner, sortBy string, tcpOnly, udpOnly bool) ListModel {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorAccent)),
	)
	return ListModel{
		table:   newPortTable(20),
		scanner: s,
		sortBy:  sortBy,
		tcpOnly: tcpOnly,
		udpOnly: udpOnly,
		view:    viewList,
		spinner: sp,
	}
}

func NewListModelWithData(listeners []model.Listener, sortBy string) ListModel {
	output.SortListeners(listeners, sortBy)
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorAccent)),
	)
	m := ListModel{
		table:     newPortTable(20),
		sortBy:    sortBy,
		listeners: listeners,
		loaded:    true,
		view:      viewList,
		spinner:   sp,
	}
	m.table.SetRows(buildPortRows(listeners))
	return m
}

func (m ListModel) Init() tea.Cmd {
	if m.loaded {
		return nil
	}
	return m.fetchListeners
}

func (m ListModel) fetchListeners() tea.Msg {
	listeners, err := m.scanner.ListListeners()
	return listenersMsg{listeners: listeners, err: err}
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case listenersMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.listeners = filterByProtocol(msg.listeners, m.tcpOnly, m.udpOnly)
		output.SortListeners(m.listeners, m.sortBy)
		m.table.SetRows(buildPortRows(m.listeners))
		m.loaded = true
		return m, nil

	case portMsg:
		m.detail = msg.listener
		m.detailOK = true
		if msg.err != nil {
			m.err = msg.err
		}
		m.view = viewDetail
		return m, nil

	case pidPortsMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.pidPorts = msg.listeners
		m.pidNum = msg.pid
		m.view = viewPIDPorts
		return m, nil

	case killResultMsg:
		if msg.success {
			m.killResult = StyleSuccess.Render(msg.message)
		} else {
			m.killResult = StyleError.Render(msg.message)
		}
		m.view = viewKillResult
		return m, nil

	case spinner.TickMsg:
		if m.view == viewKilling {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	if m.view == viewList {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}

// tryQuit handles quit logic: ctrl+c quits immediately, q requires double-press.
func (m *ListModel) tryQuit(key string) (tea.Model, tea.Cmd, bool) {
	if key == "ctrl+c" {
		return m, tea.Quit, true
	}
	if key == "q" {
		now := time.Now()
		if !m.lastQPress.IsZero() && now.Sub(m.lastQPress) < 500*time.Millisecond {
			return m, tea.Quit, true
		}
		m.lastQPress = now
		return m, nil, true // consumed the key, but don't quit yet
	}
	return m, nil, false
}

func (m ListModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Check for quit in all views
	if result, cmd, handled := m.tryQuit(key); handled {
		return result, cmd
	}

	switch m.view {
	case viewDetail:
		switch key {
		case "esc", "backspace", "left":
			m.view = viewList
			m.detail = nil
			m.detailOK = false
			m.err = nil
			return m, nil
		case "r":
			if m.detail != nil {
				m.detailOK = false
				return m, m.fetchPort(m.detail.Port)
			}
		case "p":
			if m.detail != nil && m.detail.PID > 0 {
				return m, m.fetchPIDPorts(m.detail.PID)
			}
		case "k":
			if m.detail != nil && m.detail.PID > 0 {
				m.view = viewKillConfirm
				return m, nil
			}
		}
		return m, nil

	case viewPIDPorts:
		switch key {
		case "esc", "backspace", "left":
			m.view = viewDetail
			m.pidPorts = nil
			return m, nil
		}
		return m, nil

	case viewKillConfirm:
		switch key {
		case "y", "Y":
			m.view = viewKilling
			return m, tea.Batch(m.spinner.Tick, m.doKill())
		case "n", "N", "esc":
			m.view = viewDetail
			return m, nil
		}
		return m, nil

	case viewKilling:
		return m, nil

	case viewKillResult:
		// Any key → back to list and refresh
		m.view = viewList
		m.detail = nil
		m.detailOK = false
		m.killResult = ""
		if m.scanner != nil {
			return m, m.fetchListeners
		}
		return m, nil

	default: // viewList
		switch key {
		case "s":
			m.cycleSortBy()
			output.SortListeners(m.listeners, m.sortBy)
			m.table.SetRows(buildPortRows(m.listeners))
			return m, nil
		case "r":
			if m.scanner != nil {
				return m, m.fetchListeners
			}
		case "enter":
			return m, m.fetchSelectedPort()
		default:
			// Forward unhandled keys (arrows, page up/down, etc.) to the table
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m ListModel) fetchSelectedPort() tea.Cmd {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	port, err := strconv.Atoi(row[0])
	if err != nil || m.scanner == nil {
		return nil
	}
	return m.fetchPort(port)
}

func (m ListModel) fetchPort(port int) tea.Cmd {
	return func() tea.Msg {
		listener, err := m.scanner.GetPort(port)
		return portMsg{listener: listener, err: err}
	}
}

func (m ListModel) fetchPIDPorts(pid int) tea.Cmd {
	return func() tea.Msg {
		listeners, err := m.scanner.ListByPID(pid)
		return pidPortsMsg{listeners: listeners, pid: pid, err: err}
	}
}

func (m ListModel) doKill() tea.Cmd {
	if m.detail == nil {
		return nil
	}
	pid := m.detail.PID
	return func() tea.Msg {
		err := kill.Kill(pid, syscall.SIGTERM)
		if err != nil {
			if errors.Is(err, kill.ErrPermissionDenied) {
				return killResultMsg{success: false, message: "Permission denied. Try sudo."}
			}
			return killResultMsg{success: false, message: fmt.Sprintf("Error: %v", err)}
		}
		if kill.WaitForExit(pid, 3*time.Second) {
			return killResultMsg{success: true, message: "Process terminated."}
		}
		return killResultMsg{success: false, message: "Process didn't terminate."}
	}
}

func (m ListModel) View() tea.View {
	switch m.view {
	case viewDetail:
		return m.viewDetail()
	case viewPIDPorts:
		return m.viewPIDPorts()
	case viewKillConfirm:
		return m.viewKillConfirm()
	case viewKilling:
		return m.viewKilling()
	case viewKillResult:
		return m.viewKillResult()
	default:
		return m.viewList()
	}
}

func (m ListModel) viewList() tea.View {
	if m.err != nil {
		return fullscreenView(StyleError.Render(fmt.Sprintf("  Error: %v", m.err)) + "\n")
	}
	if !m.loaded {
		return fullscreenView(StyleDim.Render("  Scanning ports...") + "\n")
	}
	if len(m.listeners) == 0 {
		return fullscreenView(StyleDim.Render("  No listening ports found.") + "\n")
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(m.table.View())
	sb.WriteString("\n")
	sb.WriteString(helpBar(
		fmt.Sprintf("%d ports", len(m.listeners)),
		fmt.Sprintf("sort: %s", m.sortBy),
		"enter detail",
		"s sort",
		"r refresh",
		"qq quit",
	))
	sb.WriteString("\n")
	return fullscreenView(sb.String())
}

func (m ListModel) viewDetail() tea.View {
	if !m.detailOK {
		return fullscreenView(StyleDim.Render("  Loading...") + "\n")
	}
	if m.err != nil {
		return fullscreenView("\n" + StyleError.Render(fmt.Sprintf("  Error: %v", m.err)) +
			"\n\n" + helpBar("esc back", "qq quit") + "\n")
	}
	if m.detail == nil {
		return fullscreenView("\n" + StyleDim.Render("  Port not in use.") +
			"\n\n" + helpBar("esc back", "qq quit") + "\n")
	}

	content := "\n" + output.RenderDetail(m.detail)
	content += helpBar("r refresh", "p pid ports", "k kill", "esc back", "qq quit") + "\n"
	return fullscreenView(content)
}

func (m ListModel) viewPIDPorts() tea.View {
	if len(m.pidPorts) == 0 {
		return fullscreenView("\n" + StyleDim.Render(fmt.Sprintf("  No ports found for PID %d", m.pidNum)) +
			"\n\n" + helpBar("esc back", "qq quit") + "\n")
	}

	// Reuse the tree formatter from output package
	f := output.NewTableFormatter()
	content := "\n" + f.FormatTree(m.pidPorts, m.pidNum)
	content += "\n" + helpBar("esc back", "qq quit") + "\n"
	return fullscreenView(content)
}

func (m ListModel) viewKillConfirm() tea.View {
	if m.detail == nil {
		return fullscreenView("")
	}

	var sb strings.Builder
	l := m.detail

	sb.WriteString("\n")
	sb.WriteString(StyleWarning.Render(fmt.Sprintf("  Kill process on port %d?", l.Port)))
	sb.WriteString("\n\n")

	name := "unknown"
	if l.Process != nil && l.Process.Command != "" {
		name = l.Process.Command
	}
	sb.WriteString(fmt.Sprintf("    %s %s\n", StyleLabel.Render("Process:"), name))
	sb.WriteString(fmt.Sprintf("    %s %d\n", StyleLabel.Render("PID:    "), l.PID))
	sb.WriteString(fmt.Sprintf("    %s SIGTERM\n", StyleLabel.Render("Signal: ")))
	sb.WriteString("\n")
	sb.WriteString(StyleBold.Render("  Confirm? [y/n]"))
	sb.WriteString("\n")

	return fullscreenView(sb.String())
}

func (m ListModel) viewKilling() tea.View {
	pid := 0
	if m.detail != nil {
		pid = m.detail.PID
	}
	return fullscreenView(fmt.Sprintf("\n  %s Sending SIGTERM to PID %d...\n", m.spinner.View(), pid))
}

func (m ListModel) viewKillResult() tea.View {
	var sb strings.Builder
	sb.WriteString("\n  ")
	sb.WriteString(m.killResult)
	sb.WriteString("\n\n")
	sb.WriteString(helpBar("any key to continue"))
	sb.WriteString("\n")
	return fullscreenView(sb.String())
}

func (m *ListModel) cycleSortBy() {
	for i, opt := range sortOptions {
		if opt == m.sortBy {
			m.sortBy = sortOptions[(i+1)%len(sortOptions)]
			return
		}
	}
	m.sortBy = sortOptions[0]
}
