package model

import "time"

// Connection states returned by lsof.
const (
	StateListen      = "LISTEN"
	StateEstablished = "ESTABLISHED"
	StateCloseWait   = "CLOSE_WAIT"
	StateTimeWait    = "TIME_WAIT"
)

// Network protocols.
const (
	ProtoTCP = "tcp"
	ProtoUDP = "udp"
)

type Process struct {
	PID           int       `json:"pid"`
	Name          string    `json:"name"`
	Command       string    `json:"command"`
	Cmdline       []string  `json:"cmdline"`
	User          string    `json:"user"`
	UID           int       `json:"uid"`
	StartTime     time.Time `json:"startTime"`
	UptimeSeconds int64     `json:"uptimeSeconds"`
}

// DisplayName returns the best human-readable name for this process.
// Prefers Command over Name, falls back to "unknown".
func (p *Process) DisplayName() string {
	if p == nil {
		return "unknown"
	}
	if p.Command != "" {
		return p.Command
	}
	if p.Name != "" {
		return p.Name
	}
	return "unknown"
}

// DisplayUser returns the process owner, or "-" if unavailable.
func (p *Process) DisplayUser() string {
	if p == nil || p.User == "" {
		return "-"
	}
	return p.User
}

type Connection struct {
	LocalAddr       string `json:"localAddr"`
	LocalPort       int    `json:"localPort"`
	RemoteAddr      string `json:"remoteAddr"`
	RemotePort      int    `json:"remotePort"`
	State           string `json:"state"`
	DurationSeconds int64  `json:"durationSeconds,omitempty"`
}

type ProcessStats struct {
	MemoryRSS   int64   `json:"memoryRSS"`
	CPUPercent  float64 `json:"cpuPercent"`
	FDCount     int     `json:"fdCount"`
	ThreadCount int     `json:"threadCount"`
}

type Listener struct {
	Port            int           `json:"port"`
	Protocol        string        `json:"protocol"`
	Address         string        `json:"address"`
	PID             int           `json:"pid"`
	Process         *Process      `json:"process,omitempty"`
	Connections     []Connection  `json:"connections,omitempty"`
	ConnectionCount int           `json:"connectionCount"`
	Stats           *ProcessStats `json:"stats,omitempty"`
}

// ProcessName returns the display name of the owning process.
func (l *Listener) ProcessName() string {
	return l.Process.DisplayName()
}

// ProcessUser returns the user of the owning process.
func (l *Listener) ProcessUser() string {
	return l.Process.DisplayUser()
}

// FilterByProtocol returns listeners matching the given protocol flags.
// If neither flag is set, all listeners are returned.
func FilterByProtocol(listeners []Listener, tcpOnly, udpOnly bool) []Listener {
	if !tcpOnly && !udpOnly {
		return listeners
	}
	filtered := make([]Listener, 0, len(listeners))
	for _, l := range listeners {
		if tcpOnly && l.Protocol != ProtoTCP {
			continue
		}
		if udpOnly && l.Protocol != ProtoUDP {
			continue
		}
		filtered = append(filtered, l)
	}
	return filtered
}

type ScanResult struct {
	Listeners []Listener `json:"listeners"`
	ScanTime  time.Time  `json:"scanTime"`
	Platform  string     `json:"platform"`
	Hostname  string     `json:"hostname"`
}
