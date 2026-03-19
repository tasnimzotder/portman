package output

import (
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/tasnimzotder/portman/internal/model"
)

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// FormatDuration formats seconds into a concise human-readable duration.
func FormatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		m := seconds / 60
		s := seconds % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm%ds", m, s)
	}
	if seconds < 86400 {
		h := seconds / 3600
		m := (seconds % 3600) / 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	}
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	if h == 0 {
		return fmt.Sprintf("%dd", d)
	}
	return fmt.Sprintf("%dd%dh", d, h)
}

// FormatBytes formats bytes into human-readable format.
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func getPlatform() string {
	return runtime.GOOS
}

// SortListeners sorts listeners by the specified field.
func SortListeners(listeners []model.Listener, by string) {
	sort.Slice(listeners, func(i, j int) bool {
		switch strings.ToLower(by) {
		case "pid":
			return listeners[i].PID < listeners[j].PID
		case "user":
			return listeners[i].ProcessUser() < listeners[j].ProcessUser()
		case "conns":
			return listeners[i].ConnectionCount > listeners[j].ConnectionCount
		case "uptime":
			ui, uj := int64(0), int64(0)
			if listeners[i].Process != nil {
				ui = listeners[i].Process.UptimeSeconds
			}
			if listeners[j].Process != nil {
				uj = listeners[j].Process.UptimeSeconds
			}
			return ui > uj
		default: // "port"
			return listeners[i].Port < listeners[j].Port
		}
	})
}

// ProcessGroup represents a set of listeners owned by the same process.
type ProcessGroup struct {
	PID       int
	Name      string
	User      string
	Listeners []model.Listener
}

// GroupByProcess groups listeners by their PID, preserving encounter order.
func GroupByProcess(listeners []model.Listener) []ProcessGroup {
	pidOrder := make([]int, 0)
	groups := make(map[int]*ProcessGroup)

	for _, l := range listeners {
		pid := l.PID
		if g, ok := groups[pid]; ok {
			g.Listeners = append(g.Listeners, l)
		} else {
			groups[pid] = &ProcessGroup{
				PID:       pid,
				Name:      l.ProcessName(),
				User:      l.ProcessUser(),
				Listeners: []model.Listener{l},
			}
			pidOrder = append(pidOrder, pid)
		}
	}

	result := make([]ProcessGroup, 0, len(groups))
	for _, pid := range pidOrder {
		result = append(result, *groups[pid])
	}
	return result
}
