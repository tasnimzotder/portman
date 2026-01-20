package model

import "time"

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

type ScanResult struct {
	Listeners []Listener `json:"listeners"`
	ScanTime  time.Time  `json:"scanTime"`
	Platform  string     `json:"platform"`
	Hostname  string     `json:"hostname"`
}
