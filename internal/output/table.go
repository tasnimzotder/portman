package output

import (
	"fmt"
	"strings"

	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/style"
)

type TableFormatter struct {
	NoHeader bool
	Grouped  bool
	SortBy   string
}

func NewTableFormatter() *TableFormatter {
	return &TableFormatter{}
}

func (f *TableFormatter) Format(listeners []model.Listener) string {
	if len(listeners) == 0 {
		return style.Dim.Render("No listening ports found.") + "\n"
	}
	if f.Grouped {
		return f.formatGrouped(listeners)
	}
	return f.formatTable(listeners)
}

func (f *TableFormatter) formatTable(listeners []model.Listener) string {
	var sb strings.Builder

	if !f.NoHeader {
		header := fmt.Sprintf("  %-7s %-7s %-8s %-10s %-24s %6s  %s",
			"PORT", "PROTO", "PID", "USER", "COMMAND", "CONNS", "UPTIME")
		sb.WriteString(style.Dim.Bold(true).Render(header))
		sb.WriteString("\n")
		sb.WriteString(style.Separator(76))
		sb.WriteString("\n")
	}

	for _, l := range listeners {
		sb.WriteString(formatRow(l))
		sb.WriteString("\n")
	}

	sb.WriteString(style.Dim.Render(fmt.Sprintf("\n  %d listening ports", len(listeners))))
	sb.WriteString("\n")

	return sb.String()
}

func formatRow(l model.Listener) string {
	pid := "-"
	if l.PID > 0 {
		pid = fmt.Sprintf("%d", l.PID)
	}

	command := truncate(l.ProcessName(), 24)
	user := truncate(l.ProcessUser(), 10)

	uptime := "-"
	if l.Process != nil && l.Process.UptimeSeconds > 0 {
		uptime = FormatDuration(l.Process.UptimeSeconds)
	}

	connsStr := fmt.Sprintf("%6d", l.ConnectionCount)
	if l.ConnectionCount >= 10 {
		connsStr = style.Bold.Render(connsStr)
	} else if l.ConnectionCount >= 5 {
		connsStr = style.Accent.Render(connsStr)
	}

	return fmt.Sprintf("  %s %s %-8s %-10s %-24s %s  %s",
		style.Port.Render(fmt.Sprintf("%-7d", l.Port)),
		style.Proto(l.Protocol),
		pid, user, command, connsStr, uptime)
}

func (f *TableFormatter) formatGrouped(listeners []model.Listener) string {
	var sb strings.Builder
	groups := GroupByProcess(listeners)

	for i, g := range groups {
		sb.WriteString(fmt.Sprintf("  %s%s\n",
			style.Bold.Render(truncate(g.Name, 32)),
			style.Dim.Render(fmt.Sprintf("  pid %d · %s", g.PID, g.User)),
		))

		for _, l := range g.Listeners {
			uptime := "-"
			if l.Process != nil && l.Process.UptimeSeconds > 0 {
				uptime = FormatDuration(l.Process.UptimeSeconds)
			}
			fmt.Fprintf(&sb, "    %s  %-5s  %s  %s\n",
				style.Accent.Render(fmt.Sprintf(":%d", l.Port)),
				l.Protocol,
				style.Dim.Render(fmt.Sprintf("%d conns", l.ConnectionCount)),
				style.Dim.Render(uptime),
			)
		}

		if i < len(groups)-1 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString(style.Dim.Render(fmt.Sprintf("\n  %d ports across %d processes", len(listeners), len(groups))))
	sb.WriteString("\n")

	return sb.String()
}

// FormatDetail renders a detailed view of a single listener.
func (f *TableFormatter) FormatDetail(l *model.Listener) string {
	if l == nil {
		return style.Dim.Render("Port not in use.") + "\n"
	}
	return RenderDetail(l)
}

// FormatTree renders a tree view of all ports owned by a PID.
func (f *TableFormatter) FormatTree(listeners []model.Listener, pid int) string {
	if len(listeners) == 0 {
		return style.Dim.Render(fmt.Sprintf("No ports found for PID %d", pid)) + "\n"
	}

	var sb strings.Builder

	name := listeners[0].ProcessName()
	user := listeners[0].ProcessUser()
	uptime := ""
	if listeners[0].Process != nil && listeners[0].Process.UptimeSeconds > 0 {
		uptime = " · " + FormatDuration(listeners[0].Process.UptimeSeconds)
	}

	fmt.Fprintf(&sb, "  %s %s\n\n",
		style.Bold.Render(name),
		style.Dim.Render(fmt.Sprintf("PID %d · %s%s", pid, user, uptime)),
	)

	for i, l := range listeners {
		connector := "├──"
		if i == len(listeners)-1 {
			connector = "└──"
		}
		conns := ""
		if l.ConnectionCount > 0 {
			conns = style.Dim.Render(fmt.Sprintf("  %d conns", l.ConnectionCount))
		}
		fmt.Fprintf(&sb, "  %s %s  %s%s\n",
			style.Dim.Render(connector),
			style.Accent.Render(fmt.Sprintf(":%d", l.Port)),
			l.Protocol, conns,
		)
	}

	sb.WriteString(style.Dim.Render(fmt.Sprintf("\n  %d ports", len(listeners))))
	sb.WriteString("\n")

	return sb.String()
}

// RenderDetail builds the detail view string for a listener.
// Shared between static output and TUI models.
func RenderDetail(l *model.Listener) string {
	var sb strings.Builder

	sb.WriteString(style.Accent.Render(fmt.Sprintf("  Port %d", l.Port)))
	sb.WriteString("\n")
	sb.WriteString(style.Separator(50))
	sb.WriteString("\n\n")

	// Process section
	sb.WriteString(style.Section.Render("  Process"))
	sb.WriteString("\n")
	if l.Process != nil {
		p := l.Process
		sb.WriteString(style.KV("PID", fmt.Sprintf("%d", p.PID)))
		sb.WriteString(style.KV("Command", p.DisplayName()))
		if len(p.Cmdline) > 0 {
			sb.WriteString(style.KV("Full", strings.Join(p.Cmdline, " ")))
		}
		sb.WriteString(style.KV("User", fmt.Sprintf("%s (uid: %d)", p.User, p.UID)))
		if !p.StartTime.IsZero() {
			sb.WriteString(style.KV("Started", fmt.Sprintf("%s (%s ago)",
				p.StartTime.Format("2006-01-02 15:04:05"),
				FormatDuration(p.UptimeSeconds))))
		} else if p.UptimeSeconds > 0 {
			sb.WriteString(style.KV("Uptime", FormatDuration(p.UptimeSeconds)))
		}
	} else {
		sb.WriteString(style.Dim.Render("    (permission denied or process info unavailable)"))
		sb.WriteString("\n")
	}

	// Listening section
	sb.WriteString("\n")
	sb.WriteString(style.Section.Render("  Listening"))
	sb.WriteString("\n")
	addr := l.Address
	if addr == "0.0.0.0" || addr == "::" {
		addr = fmt.Sprintf("%s (all interfaces)", l.Address)
	}
	sb.WriteString(style.KV("Address", fmt.Sprintf("%s:%d", addr, l.Port)))
	sb.WriteString(style.KV("Protocol", strings.ToUpper(l.Protocol)))

	// Connections section
	if len(l.Connections) > 0 {
		sb.WriteString("\n")
		sb.WriteString(style.Section.Render(fmt.Sprintf("  Connections (%d established)", len(l.Connections))))
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "    %-36s %-14s %s\n",
			style.Bold.Render("REMOTE ADDRESS"),
			style.Bold.Render("STATE"),
			style.Bold.Render("DURATION"),
		)
		for _, c := range l.Connections {
			dur := "-"
			if c.DurationSeconds > 0 {
				dur = FormatDuration(c.DurationSeconds)
			}
			fmt.Fprintf(&sb, "    %-36s %-14s %s\n",
				fmt.Sprintf("%s:%d", c.RemoteAddr, c.RemotePort),
				style.ConnState(c.State),
				dur,
			)
		}
	}

	// Stats section
	if l.Stats != nil {
		st := l.Stats
		sb.WriteString("\n")
		sb.WriteString(style.Section.Render("  Stats"))
		sb.WriteString("\n")
		sb.WriteString(style.KV("Memory", fmt.Sprintf("%s (RSS)", FormatBytes(st.MemoryRSS))))
		sb.WriteString(style.KV("CPU", fmt.Sprintf("%.1f%%", st.CPUPercent)))
		sb.WriteString(style.KV("FDs", fmt.Sprintf("%d open", st.FDCount)))
		sb.WriteString(style.KV("Threads", fmt.Sprintf("%d", st.ThreadCount)))
	}

	sb.WriteString("\n")
	return sb.String()
}
