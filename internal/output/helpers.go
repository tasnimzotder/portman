package output

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

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

// FormatDuration formats seconds into a human-readable duration
func FormatDuration(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	return d.String()
}

// FormatBytes formats bytes into human-readable format
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
// Valid fields: port, pid, user, conns, uptime
func SortListeners(listeners []model.Listener, by string) {
	sort.Slice(listeners, func(i, j int) bool {
		switch strings.ToLower(by) {
		case "pid":
			return listeners[i].PID < listeners[j].PID
		case "user":
			ui, uj := "", ""
			if listeners[i].Process != nil {
				ui = listeners[i].Process.User
			}
			if listeners[j].Process != nil {
				uj = listeners[j].Process.User
			}
			return ui < uj
		case "conns":
			return listeners[i].ConnectionCount > listeners[j].ConnectionCount // Descending
		case "uptime":
			ui, uj := int64(0), int64(0)
			if listeners[i].Process != nil {
				ui = listeners[i].Process.UptimeSeconds
			}
			if listeners[j].Process != nil {
				uj = listeners[j].Process.UptimeSeconds
			}
			return ui > uj // Descending (longest uptime first)
		default: // "port"
			return listeners[i].Port < listeners[j].Port
		}
	})
}
