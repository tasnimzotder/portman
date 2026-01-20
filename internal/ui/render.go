package ui

import (
	"fmt"
	"strings"

	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/output"
)

// render displays the current state with highlighting (flicker-free)
func (s *WatchState) render() {
	// Only clear screen on first render, then just move cursor to top
	if s.isFirstRender {
		ClearAndReset()
		s.isFirstRender = false
	} else {
		MoveToTop()
	}

	// Header
	PrintLine("%s%sportman --watch%s  ", Bold, Cyan, Reset)
	fmt.Printf("%sRefresh: %s%s  ", Dim, s.config.Interval, Reset)
	fmt.Printf("%sPress 'q' to quit%s\n", Dim, Reset)
	PrintLine("%s\n", strings.Repeat("─", 70))

	// Column headers
	PrintLine("%s%-8s %-8s %-8s %-10s %-8s %-12s %s%s\n",
		Bold,
		"PORT", "PROTO", "PID", "USER", "CONNS", "UPTIME", "PROCESS",
		Reset)

	if len(s.previous) == 0 {
		PrintLine("\n")
		PrintLine("%sNo listening ports found.%s\n", Dim, Reset)
		// Clear remaining lines
		for range 20 {
			PrintLine("\n")
		}
		return
	}

	// Convert map to slice for ordered display
	listeners := make([]model.Listener, 0, len(s.previous))
	for _, l := range s.previous {
		listeners = append(listeners, l)
	}
	output.SortListeners(listeners, s.config.SortBy)

	// Render each row
	for _, l := range listeners {
		s.renderRow(l)
	}

	// Show removed ports briefly
	if len(s.removed) > 0 {
		PrintLine("\n")
		for port := range s.removed {
			PrintLine("%s  ● Port %d removed%s\n", Red, port, Reset)
		}
	}

	// Footer
	PrintLine("\n")
	PrintLine("%s%d ports%s", Dim, len(listeners), Reset)
	if len(s.added) > 0 {
		fmt.Printf("  %s+%d new%s", Green, len(s.added), Reset)
	}
	if len(s.removed) > 0 {
		fmt.Printf("  %s-%d removed%s", Red, len(s.removed), Reset)
	}
	fmt.Println()

	// Clear any leftover lines from previous renders
	for range 10 {
		PrintLine("\n")
	}
}

// renderRow renders a single listener row with appropriate highlighting
func (s *WatchState) renderRow(l model.Listener) {
	processName := ""
	user := ""
	uptime := ""

	if l.Process != nil {
		processName = l.Process.Name
		if len(processName) > 20 {
			processName = processName[:17] + "..."
		}
		user = l.Process.User
		if len(user) > 10 {
			user = user[:7] + "..."
		}
		if l.Process.UptimeSeconds > 0 {
			uptime = output.FormatDuration(l.Process.UptimeSeconds)
		}
	}

	// Choose color based on state
	color := ""
	if s.added[l.Port] {
		color = Green
	}

	row := fmt.Sprintf("%-8d %-8s %-8d %-10s %-8d %-12s %s",
		l.Port,
		l.Protocol,
		l.PID,
		user,
		l.ConnectionCount,
		uptime,
		processName,
	)

	if color != "" {
		PrintLine("%s%s%s\n", color, row, Reset)
	} else {
		PrintLine("%s\n", row)
	}
}
