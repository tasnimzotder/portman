//go:build darwin

package scanner

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/tasnimzotder/portman/internal/model"
)

type DarwinScanner struct {
	opts Options
}

func NewDarwinScanner(opts Options) *DarwinScanner {
	return &DarwinScanner{opts: opts}
}

func (s *DarwinScanner) ListListeners() ([]model.Listener, error) {
	args := []string{"-i", "-n", "-P"}
	if s.opts.IncludeTCP && !s.opts.IncludeUDP {
		args = append(args, "-sTCP:LISTEN")
	}

	cmd := exec.Command("lsof", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lsof failed: %w", err)
	}

	return s.parseLsofOutput(string(output))
}

func (s *DarwinScanner) GetPort(port int) (*model.Listener, error) {
	cmd := exec.Command(
		"lsof",
		"-i",
		fmt.Sprintf(":%d", port),
		"-n",
		"-P",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, nil // Port not in use
	}

	listener, connections := s.parsePortDetail(string(output), port)
	if listener == nil {
		return nil, nil
	}

	listener.Connections = connections
	listener.ConnectionCount = len(connections)
	listener.Stats = s.getProcessStats(listener.PID)

	return listener, nil
}

// getProcessStats collects memory, CPU, FD count, and thread count for a process.
func (s *DarwinScanner) getProcessStats(pid int) *model.ProcessStats {
	stats := &model.ProcessStats{}
	pidStr := strconv.Itoa(pid)

	// Memory (RSS in KB) and CPU using ps
	cmd := exec.Command("ps", "-o", "rss=,%cpu=", "-p", pidStr)
	output, err := cmd.Output()
	if err == nil {
		fields := strings.Fields(string(output))
		if len(fields) >= 2 {
			if rss, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
				stats.MemoryRSS = rss * 1024 // KB to bytes
			}
			if cpu, err := strconv.ParseFloat(fields[1], 64); err == nil {
				stats.CPUPercent = cpu
			}
		}
	}

	// FD count using lsof -p
	cmd = exec.Command("lsof", "-p", pidStr)
	output, _ = cmd.Output()
	lines := strings.Split(string(output), "\n")
	if len(lines) > 2 {
		stats.FDCount = len(lines) - 2 // Subtract header and trailing newline
	}

	// Thread count using ps -M
	cmd = exec.Command("ps", "-M", "-p", pidStr)
	output, _ = cmd.Output()
	lines = strings.Split(string(output), "\n")
	if len(lines) > 2 {
		stats.ThreadCount = len(lines) - 2 // Subtract header and trailing newline
	}

	return stats
}

// parsePortDetail parses lsof output for a specific port, returning the listener and its connections.
func (s *DarwinScanner) parsePortDetail(output string, targetPort int) (*model.Listener, []model.Connection) {
	var listener *model.Listener
	var connections []model.Connection

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "COMMAND") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		command := fields[0]
		pid, _ := strconv.Atoi(fields[1])
		user := fields[2]
		protocol := strings.ToLower(fields[7])
		name := fields[8]

		state := ""
		if len(fields) > 9 {
			state = strings.Trim(fields[9], "()")
		}

		if state == "LISTEN" {
			addr, port := parseAddressPort(name)
			if port == targetPort && listener == nil {
				uptime := getProcessUptime(pid)
				listener = &model.Listener{
					Port:     port,
					Protocol: protocol,
					Address:  addr,
					PID:      pid,
					Process: &model.Process{
						PID:           pid,
						Name:          command,
						Command:       command,
						User:          user,
						UptimeSeconds: uptime,
					},
				}
			}
		} else if state == "ESTABLISHED" && strings.Contains(name, "->") {
			// Parse: "10.0.0.1:22000->192.168.1.50:45678"
			parts := strings.Split(name, "->")
			if len(parts) == 2 {
				localAddr, localPort := parseAddressPort(parts[0])
				remoteAddr, remotePort := parseAddressPort(parts[1])

				if localPort == targetPort {
					connections = append(connections, model.Connection{
						LocalAddr:  localAddr,
						LocalPort:  localPort,
						RemoteAddr: remoteAddr,
						RemotePort: remotePort,
						State:      state,
					})
				}
			}
		}
	}

	return listener, connections
}

func (s *DarwinScanner) FindByPattern(pattern string) ([]model.Listener, error) {
	listeners, err := s.ListListeners()
	if err != nil {
		return nil, err
	}

	var matches []model.Listener
	patternLower := strings.ToLower(pattern)

	// Check if pattern is a port number
	patternPort, isPort := strconv.Atoi(pattern)

	for _, l := range listeners {
		// Match by port number
		if isPort == nil && l.Port == patternPort {
			matches = append(matches, l)
			continue
		}

		// Match by PID
		if isPort == nil && l.PID == patternPort {
			matches = append(matches, l)
			continue
		}

		if l.Process == nil {
			continue
		}

		name := strings.ToLower(l.Process.Name)
		cmd := strings.ToLower(l.Process.Command)
		user := strings.ToLower(l.Process.User)

		if strings.Contains(name, patternLower) ||
			strings.Contains(cmd, patternLower) ||
			strings.Contains(user, patternLower) {
			matches = append(matches, l)
		}
	}

	return matches, nil
}

// lsofEntry represents a parsed line from lsof output.
type lsofEntry struct {
	command  string
	pid      int
	user     string
	protocol string
	port     int
	address  string
	state    string // "LISTEN", "ESTABLISHED", etc.
}

// Example
// COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
// nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
// nginx    5678   root    7u  IPv4   0x456       0t0  TCP 10.0.0.1:80->192.168.1.1:54321 (ESTABLISHED)
func (s *DarwinScanner) parseLsofOutput(output string) ([]model.Listener, error) {
	// First pass: parse all entries
	var entries []lsofEntry
	connCount := make(map[int]int) // port -> established connection count

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "COMMAND") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		command := fields[0]
		pid, _ := strconv.Atoi(fields[1])
		user := fields[2]
		protocol := strings.ToLower(fields[7])
		name := fields[8]

		state := ""
		if len(fields) > 9 {
			state = strings.Trim(fields[9], "()")
		}

		// Parse port from name field
		// For ESTABLISHED: "10.0.0.1:80->192.168.1.1:54321"
		// For LISTEN: "*:80"
		var port int
		var addr string

		if strings.Contains(name, "->") {
			// ESTABLISHED connection: extract local port
			localPart := strings.Split(name, "->")[0]
			addr, port = parseAddressPort(localPart)
		} else {
			addr, port = parseAddressPort(name)
		}

		if port == 0 {
			continue
		}

		// Count established connections
		if state == "ESTABLISHED" {
			connCount[port]++
			continue // Don't add to entries, just count
		}

		if state != "LISTEN" {
			continue
		}

		entries = append(entries, lsofEntry{
			command:  command,
			pid:      pid,
			user:     user,
			protocol: protocol,
			port:     port,
			address:  addr,
			state:    state,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Second pass: build listeners with connection counts and uptime
	var listeners []model.Listener
	seen := make(map[int]bool)

	for _, e := range entries {
		if seen[e.port] {
			continue
		}
		seen[e.port] = true

		uptime := getProcessUptime(e.pid)

		listeners = append(listeners, model.Listener{
			Port:            e.port,
			Protocol:        e.protocol,
			Address:         e.address,
			PID:             e.pid,
			ConnectionCount: connCount[e.port],
			Process: &model.Process{
				PID:           e.pid,
				Name:          e.command,
				Command:       e.command,
				User:          e.user,
				UptimeSeconds: uptime,
			},
		})
	}

	return listeners, nil
}

// parseAddressPort extracts address and port from lsof NAME field.
// Examples:
//
//	"*:80" -> ("0.0.0.0", 80)
//	"127.0.0.1:3000" -> ("127.0.0.1", 3000)
//	"[fe80::1]:22000" -> ("fe80::1", 22000)
func parseAddressPort(name string) (string, int) {
	// Handle IPv6 format: [address]:port
	if strings.HasPrefix(name, "[") {
		closeBracket := strings.LastIndex(name, "]")
		if closeBracket == -1 {
			return "", 0
		}
		addr := name[1:closeBracket] // Remove brackets
		portStr := strings.TrimPrefix(name[closeBracket+1:], ":")
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return "", 0
		}
		return addr, port
	}

	// Handle IPv4 format: address:port or *:port
	idx := strings.LastIndex(name, ":")
	if idx == -1 {
		return "", 0
	}

	addr := name[:idx]
	if addr == "*" {
		addr = "0.0.0.0"
	}

	port, err := strconv.Atoi(name[idx+1:])
	if err != nil {
		return "", 0
	}

	return addr, port
}

// getProcessUptime returns the uptime in seconds for a given PID.
// Uses `ps -o etime=` which returns elapsed time in format:
// - "57:42" (minutes:seconds)
// - "22:57:42" (hours:minutes:seconds)
// - "01-22:57:42" (days-hours:minutes:seconds)
func getProcessUptime(pid int) int64 {
	cmd := exec.Command("ps", "-o", "etime=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	return parseElapsedTime(strings.TrimSpace(string(output)))
}

// parseElapsedTime converts ps etime format to seconds.
func parseElapsedTime(etime string) int64 {
	var days, hours, minutes, seconds int64

	// Check for days (format: "DD-HH:MM:SS")
	if strings.Contains(etime, "-") {
		parts := strings.SplitN(etime, "-", 2)
		days, _ = strconv.ParseInt(parts[0], 10, 64)
		etime = parts[1]
	}

	// Split remaining by ":"
	parts := strings.Split(etime, ":")
	switch len(parts) {
	case 3: // HH:MM:SS
		hours, _ = strconv.ParseInt(parts[0], 10, 64)
		minutes, _ = strconv.ParseInt(parts[1], 10, 64)
		seconds, _ = strconv.ParseInt(parts[2], 10, 64)
	case 2: // MM:SS
		minutes, _ = strconv.ParseInt(parts[0], 10, 64)
		seconds, _ = strconv.ParseInt(parts[1], 10, 64)
	}

	return days*86400 + hours*3600 + minutes*60 + seconds
}
