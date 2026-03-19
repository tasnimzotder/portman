package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/scanner"
	"github.com/tasnimzotder/portman/internal/style"
)

type WatchModel struct {
	scanner   scanner.Scanner
	interval  time.Duration
	sortBy    string
	tcpOnly   bool
	udpOnly   bool
	port      int // 0 = all ports, >0 = single port
	listeners []model.Listener
	previous  map[int]model.Listener
	added     map[int]bool
	removed   map[int]bool
	listener  *model.Listener // single-port mode
	prevSnap  *portSnapshot
	prevConns map[int]int // port → previous connection count (for deltas)
	err       error
	loaded    bool
	lastScan  time.Time
}

type portSnapshot struct {
	PID             int
	ConnectionCount int
	MemoryRSS       int64
	CPUPercent      float64
	FDCount         int
	ThreadCount     int
}

func NewWatchModel(s scanner.Scanner, interval time.Duration, sortBy string, tcpOnly, udpOnly bool) WatchModel {
	return WatchModel{
		scanner:   s,
		interval:  interval,
		sortBy:    sortBy,
		tcpOnly:   tcpOnly,
		udpOnly:   udpOnly,
		port:      0,
		added:     make(map[int]bool),
		removed:   make(map[int]bool),
		prevConns: make(map[int]int),
	}
}

func NewWatchPortModel(s scanner.Scanner, port int, interval time.Duration) WatchModel {
	return WatchModel{
		scanner:   s,
		interval:  interval,
		port:      port,
		added:     make(map[int]bool),
		removed:   make(map[int]bool),
		prevConns: make(map[int]int),
	}
}

func (m WatchModel) Init() tea.Cmd {
	return tea.Batch(m.fetch(), tick(m.interval))
}

func (m WatchModel) fetch() tea.Cmd {
	if m.port > 0 {
		return func() tea.Msg {
			listener, err := m.scanner.GetPort(m.port)
			return portMsg{listener: listener, err: err}
		}
	}
	return func() tea.Msg {
		listeners, err := m.scanner.ListListeners()
		return listenersMsg{listeners: listeners, err: err}
	}
}

func (m WatchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case listenersMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		filtered := filterByProtocol(msg.listeners, m.tcpOnly, m.udpOnly)
		output.SortListeners(filtered, m.sortBy)
		m.updateDiff(filtered)
		m.listeners = filtered
		m.loaded = true
		m.lastScan = time.Now()
		return m, nil

	case portMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.listener = msg.listener
		m.loaded = true
		m.lastScan = time.Now()
		return m, nil

	case tickMsg:
		return m, tea.Batch(m.fetch(), tick(m.interval))

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *WatchModel) updateDiff(current []model.Listener) {
	currentMap := make(map[int]model.Listener, len(current))
	for _, l := range current {
		currentMap[l.Port] = l
	}

	m.added = make(map[int]bool)
	m.removed = make(map[int]bool)

	if m.previous != nil {
		for port := range currentMap {
			if _, exists := m.previous[port]; !exists {
				m.added[port] = true
			}
		}
		for port := range m.previous {
			if _, exists := currentMap[port]; !exists {
				m.removed[port] = true
			}
		}
	}

	// Track connection count deltas
	newConns := make(map[int]int, len(current))
	for _, l := range current {
		newConns[l.Port] = l.ConnectionCount
	}
	m.prevConns, m.previous = newConns, currentMap
}

func (m WatchModel) View() tea.View {
	if m.err != nil {
		return fullscreenView(StyleError.Render(fmt.Sprintf("Error: %v", m.err)) + "\n")
	}
	if !m.loaded {
		return fullscreenView(StyleDim.Render("  Scanning...") + "\n")
	}

	if m.port > 0 {
		return m.viewSinglePort()
	}
	return m.viewAllPorts()
}

func (m WatchModel) viewAllPorts() tea.View {
	var sb strings.Builder

	// Header with timestamp
	sb.WriteString("\n")
	sb.WriteString(StyleHeader.Render("  portman watch"))
	timestamp := ""
	if !m.lastScan.IsZero() {
		timestamp = m.lastScan.Format("15:04:05")
	}
	sb.WriteString(StyleDim.Render(fmt.Sprintf("  %d ports · every %s · %s", len(m.listeners), m.interval, timestamp)))
	sb.WriteString("\n")
	sb.WriteString(StyleSubtle.Render("  " + strings.Repeat("─", 76)))
	sb.WriteString("\n")

	// Column headers
	sb.WriteString(fmt.Sprintf("  %-8s %-7s %-8s %-10s %-24s %6s  %s\n",
		StyleBold.Render("PORT"),
		StyleBold.Render("PROTO"),
		StyleBold.Render("PID"),
		StyleBold.Render("USER"),
		StyleBold.Render("COMMAND"),
		StyleBold.Render("CONNS"),
		StyleBold.Render("UPTIME"),
	))

	if len(m.listeners) == 0 {
		sb.WriteString(StyleDim.Render("  No listening ports found."))
		sb.WriteString("\n")
	}

	for _, l := range m.listeners {
		row := m.formatWatchRow(l)
		if m.added[l.Port] {
			// Status indicator: new port
			sb.WriteString(StyleSuccess.Render("+ ") + StyleSuccess.Render(row[2:])) // replace leading spaces
		} else {
			sb.WriteString(row)
		}
		sb.WriteString("\n")
	}

	// Show removed ports
	for port := range m.removed {
		sb.WriteString(StyleError.Render(fmt.Sprintf("- port %d removed", port)))
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("\n")
	footer := StyleDim.Render(fmt.Sprintf("  %d ports", len(m.listeners)))
	if n := len(m.added); n > 0 {
		footer += " " + StyleSuccess.Render(fmt.Sprintf("+%d", n))
	}
	if n := len(m.removed); n > 0 {
		footer += " " + StyleError.Render(fmt.Sprintf("-%d", n))
	}
	footer += StyleDim.Render(" · q quit")
	sb.WriteString(footer)
	sb.WriteString("\n")

	return fullscreenView(sb.String())
}

func (m WatchModel) formatWatchRow(l model.Listener) string {
	pid := "-"
	user := "-"
	command := "-"
	uptime := "-"

	if l.PID > 0 {
		pid = strconv.Itoa(l.PID)
	}
	if l.Process != nil {
		if l.Process.User != "" {
			user = l.Process.User
		}
		if l.Process.Command != "" {
			command = l.Process.Command
		} else if l.Process.Name != "" {
			command = l.Process.Name
		}
		if l.Process.UptimeSeconds > 0 {
			uptime = output.FormatDuration(l.Process.UptimeSeconds)
		}
	}

	// Connection count with delta arrow
	connsStr := fmt.Sprintf("%6d", l.ConnectionCount)
	if prev, ok := m.prevConns[l.Port]; ok {
		delta := l.ConnectionCount - prev
		if delta > 0 {
			connsStr = StyleSuccess.Render(fmt.Sprintf("%4d +%d", l.ConnectionCount, delta))
		} else if delta < 0 {
			connsStr = StyleError.Render(fmt.Sprintf("%4d %d", l.ConnectionCount, delta))
		}
	}

	// Connection count badge coloring
	if l.ConnectionCount >= 10 {
		connsStr = StyleWarning.Render(connsStr)
	}

	// Protocol coloring
	proto := l.Protocol
	if proto == "udp" {
		proto = StyleWarning.Render("udp")
	}

	return fmt.Sprintf("  %-8d %-7s %-8s %-10s %-24s %s  %s",
		l.Port, proto, pid, user, command, connsStr, uptime)
}

func (m *WatchModel) viewSinglePort() tea.View {
	var sb strings.Builder

	// Header with timestamp
	sb.WriteString("\n")
	sb.WriteString(StyleHeader.Render("  portman watch"))
	timestamp := ""
	if !m.lastScan.IsZero() {
		timestamp = m.lastScan.Format("15:04:05")
	}
	sb.WriteString(StyleDim.Render(fmt.Sprintf("  port %d · every %s · %s", m.port, m.interval, timestamp)))
	sb.WriteString("\n")
	sb.WriteString(StyleSubtle.Render("  " + strings.Repeat("─", 50)))
	sb.WriteString("\n")

	if m.listener == nil {
		sb.WriteString("\n")
		sb.WriteString(StyleDim.Render(fmt.Sprintf("  Port %d is not in use.", m.port)))
		sb.WriteString("\n\n")
		sb.WriteString(StyleDim.Render("  q quit"))
		sb.WriteString("\n")
		m.prevSnap = nil
		return fullscreenView(sb.String())
	}

	l := m.listener

	var currentSnap portSnapshot
	currentSnap.PID = l.PID
	currentSnap.ConnectionCount = l.ConnectionCount
	if l.Stats != nil {
		currentSnap.MemoryRSS = l.Stats.MemoryRSS
		currentSnap.CPUPercent = l.Stats.CPUPercent
		currentSnap.FDCount = l.Stats.FDCount
		currentSnap.ThreadCount = l.Stats.ThreadCount
	}

	// Process section
	if l.Process != nil {
		p := l.Process
		sb.WriteString("\n")
		sb.WriteString(StyleSection.Render("  Process"))
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "    %s %d\n", StyleLabel.Render("PID:       "), p.PID)
		fmt.Fprintf(&sb, "    %s %s\n", StyleLabel.Render("Command:   "), p.Command)
		fmt.Fprintf(&sb, "    %s %s\n", StyleLabel.Render("User:      "), p.User)
		if p.UptimeSeconds > 0 {
			if !p.StartTime.IsZero() {
				fmt.Fprintf(&sb, "    %s %s (%s)\n", StyleLabel.Render("Started:   "),
					p.StartTime.Format("2006-01-02 15:04:05"),
					output.FormatDuration(p.UptimeSeconds))
			} else {
				fmt.Fprintf(&sb, "    %s %s\n", StyleLabel.Render("Uptime:    "),
					output.FormatDuration(p.UptimeSeconds))
			}
		}
	} else if l.PID > 0 {
		sb.WriteString("\n")
		sb.WriteString(StyleSection.Render("  Process"))
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "    %s %d\n", StyleLabel.Render("PID:       "), l.PID)
	}

	// Listening section
	sb.WriteString("\n")
	sb.WriteString(StyleSection.Render("  Listening"))
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "    %s %s:%d\n", StyleLabel.Render("Address:   "), l.Address, l.Port)
	fmt.Fprintf(&sb, "    %s %s\n", StyleLabel.Render("Protocol:  "), l.Protocol)

	// Connections with diff indicator
	connLabel := fmt.Sprintf("Connections (%d)", len(l.Connections))
	if m.prevSnap != nil && l.ConnectionCount != m.prevSnap.ConnectionCount {
		diff := l.ConnectionCount - m.prevSnap.ConnectionCount
		sign := "+"
		if diff < 0 {
			sign = ""
		}
		connLabel += " " + StyleWarning.Render(fmt.Sprintf("[%s%d]", sign, diff))
	}
	if len(l.Connections) > 0 {
		sb.WriteString("\n")
		sb.WriteString(StyleSection.Render("  " + connLabel))
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "    %-24s %-14s %s\n",
			StyleBold.Render("REMOTE"),
			StyleBold.Render("STATE"),
			StyleBold.Render("DURATION"),
		)
		for _, c := range l.Connections {
			remote := fmt.Sprintf("%s:%d", c.RemoteAddr, c.RemotePort)
			dur := "-"
			if c.DurationSeconds > 0 {
				dur = output.FormatDuration(c.DurationSeconds)
			}
			fmt.Fprintf(&sb, "    %-24s %-14s %s\n", remote, style.ConnState(c.State), dur)
		}
	}

	// Stats with change highlighting
	if l.Stats != nil {
		st := l.Stats
		sb.WriteString("\n")
		sb.WriteString(StyleSection.Render("  Stats"))
		sb.WriteString("\n")

		memStr := output.FormatBytes(st.MemoryRSS)
		cpuStr := fmt.Sprintf("%.1f%%", st.CPUPercent)
		fdStr := strconv.Itoa(st.FDCount)
		threadStr := strconv.Itoa(st.ThreadCount)

		if m.prevSnap != nil {
			if st.MemoryRSS != m.prevSnap.MemoryRSS {
				memStr = StyleWarning.Render(memStr)
			}
			if st.CPUPercent != m.prevSnap.CPUPercent {
				cpuStr = StyleWarning.Render(cpuStr)
			}
			if st.FDCount != m.prevSnap.FDCount {
				fdStr = StyleWarning.Render(fdStr)
			}
			if st.ThreadCount != m.prevSnap.ThreadCount {
				threadStr = StyleWarning.Render(threadStr)
			}
		}

		fmt.Fprintf(&sb, "    %s %s\n", StyleLabel.Render("Memory:    "), memStr)
		fmt.Fprintf(&sb, "    %s %s\n", StyleLabel.Render("CPU:       "), cpuStr)
		fmt.Fprintf(&sb, "    %s %s\n", StyleLabel.Render("FDs:       "), fdStr)
		fmt.Fprintf(&sb, "    %s %s\n", StyleLabel.Render("Threads:   "), threadStr)
	}

	sb.WriteString("\n")
	sb.WriteString(StyleDim.Render("  q quit"))
	sb.WriteString("\n")

	m.prevSnap = &currentSnap

	return fullscreenView(sb.String())
}
