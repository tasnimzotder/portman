package output

import (
	"fmt"
	"strings"

	"github.com/tasnimzotder/portman/internal/model"
)

type TableFormatter struct {
	NoHeader bool
	SortBy   string
}

func NewTableFormatter() *TableFormatter {
	return &TableFormatter{}
}

func (f *TableFormatter) Format(listeners []model.Listener) string {
	if len(listeners) == 0 {
		return "No listening ports found."
	}

	var sb strings.Builder

	if !f.NoHeader {
		sb.WriteString(fmt.Sprintf(
			"%-8s %-8s %-8s %-10s %-24s %-7s %s\n",
			"PORT", "PROTO", "PID", "USER", "COMMAND", "CONNS", "UPTIME",
		))
	}

	for _, l := range listeners {
		pid := "-"
		user := "-"
		command := "-"
		uptime := "-"

		if l.PID > 0 {
			pid = fmt.Sprintf("%d", l.PID)
		}

		if l.Process != nil {
			if l.Process.User != "" {
				user = l.Process.User
			}

			if l.Process.Command != "" {
				command = truncate(l.Process.Command, 24)
			} else if l.Process.Name != "" {
				command = truncate(l.Process.Name, 24)
			}

			if l.Process.UptimeSeconds > 0 {
				uptime = formatDuration(l.Process.UptimeSeconds)
			}
		}

		sb.WriteString(fmt.Sprintf(
			"%-8d %-8s %-8s %-10s %-24s %-7d %s\n",
			l.Port,
			l.Protocol,
			pid,
			truncate(user, 10),
			command,
			l.ConnectionCount,
			uptime,
		))
	}

	return sb.String()
}

func (f *TableFormatter) FormatDetail(l *model.Listener) string {
	if l == nil {
		return "Port not in use."
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Port %d\n", l.Port))
	sb.WriteString("═══════════════════════════════════════════════════════════════\n\n")

	sb.WriteString("Process\n")
	if l.Process != nil {
		sb.WriteString(fmt.Sprintf("  PID:         %d\n", l.Process.PID))
		sb.WriteString(fmt.Sprintf("  Command:     %s\n", l.Process.Command))
		if len(l.Process.Cmdline) > 0 {
			sb.WriteString(fmt.Sprintf("  Full:        %s\n", strings.Join(l.Process.Cmdline, " ")))
		}
		sb.WriteString(fmt.Sprintf("  User:        %s (uid: %d)\n", l.Process.User, l.Process.UID))
		if !l.Process.StartTime.IsZero() {
			sb.WriteString(fmt.Sprintf("  Started:     %s (%s ago)\n",
				l.Process.StartTime.Format("2006-01-02 15:04:05"),
				formatDuration(l.Process.UptimeSeconds)))
		} else if l.Process.UptimeSeconds > 0 {
			sb.WriteString(fmt.Sprintf("  Uptime:      %s\n", formatDuration(l.Process.UptimeSeconds)))
		}
	} else {
		sb.WriteString("  (permission denied or process info unavailable)\n")
	}

	sb.WriteString("\n")

	sb.WriteString("Listening\n")
	addr := l.Address
	if addr == "0.0.0.0" || addr == "::" {
		addr = fmt.Sprintf("%s (all interfaces)", l.Address)
	}
	sb.WriteString(fmt.Sprintf("  Address:     %s:%d\n", addr, l.Port))
	sb.WriteString(fmt.Sprintf("  Protocol:    %s\n", strings.ToUpper(l.Protocol)))

	if len(l.Connections) > 0 {
		sb.WriteString(fmt.Sprintf("\nConnections (%d established)\n", len(l.Connections)))
		sb.WriteString(fmt.Sprintf("  %-42s %-14s %s\n", "REMOTE ADDRESS", "STATE", "DURATION"))
		for _, c := range l.Connections {
			remoteAddr := fmt.Sprintf("%s:%d", c.RemoteAddr, c.RemotePort)
			duration := "-"
			if c.DurationSeconds > 0 {
				duration = formatDuration(c.DurationSeconds)
			}
			sb.WriteString(fmt.Sprintf("  %-42s %-14s %s\n", remoteAddr, c.State, duration))
		}
	}

	if l.Stats != nil {
		sb.WriteString("\nProcess Stats\n")
		sb.WriteString(fmt.Sprintf("  Memory:      %s (RSS)\n", formatBytes(l.Stats.MemoryRSS)))
		sb.WriteString(fmt.Sprintf("  CPU:         %.1f%%\n", l.Stats.CPUPercent))
		sb.WriteString(fmt.Sprintf("  FDs:         %d open\n", l.Stats.FDCount))
		sb.WriteString(fmt.Sprintf("  Threads:     %d\n", l.Stats.ThreadCount))
	}

	return sb.String()
}
