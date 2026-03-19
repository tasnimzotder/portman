//go:build darwin

package scanner

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/tasnimzotder/portman/internal/model"
)

const cmdTimeout = 10 * time.Second

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

	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "lsof", args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("lsof timed out after %s", cmdTimeout)
		}
		return nil, fmt.Errorf("lsof failed: %w", err)
	}

	return s.parseLsofOutput(string(output))
}

func (s *DarwinScanner) GetPort(port int) (*model.Listener, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "lsof",
		"-i", fmt.Sprintf(":%d", port),
		"-n", "-P",
	)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("lsof timed out after %s", cmdTimeout)
		}
		return nil, nil
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

func (s *DarwinScanner) ListByPID(pid int) ([]model.Listener, error) {
	listeners, err := s.ListListeners()
	if err != nil {
		return nil, err
	}

	var matches []model.Listener
	for _, l := range listeners {
		if l.PID == pid {
			matches = append(matches, l)
		}
	}
	return matches, nil
}

func (s *DarwinScanner) getProcessStats(pid int) *model.ProcessStats {
	stats := &model.ProcessStats{}
	pidStr := strconv.Itoa(pid)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ps", "-o", "rss=,%cpu=", "-p", pidStr)
	output, err := cmd.Output()
	if err == nil {
		fields := strings.Fields(string(output))
		if len(fields) >= 2 {
			if rss, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
				stats.MemoryRSS = rss * 1024
			}
			if cpu, err := strconv.ParseFloat(fields[1], 64); err == nil {
				stats.CPUPercent = cpu
			}
		}
	}

	cmd = exec.CommandContext(ctx, "lsof", "-p", pidStr)
	if fdOutput, err := cmd.Output(); err == nil {
		lines := strings.Split(string(fdOutput), "\n")
		if len(lines) > 2 {
			stats.FDCount = len(lines) - 2
		}
	}

	cmd = exec.CommandContext(ctx, "ps", "-M", "-p", pidStr)
	if thOutput, err := cmd.Output(); err == nil {
		lines := strings.Split(string(thOutput), "\n")
		if len(lines) > 2 {
			stats.ThreadCount = len(lines) - 2
		}
	}

	return stats
}

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

		command := decodeLsofEscapes(fields[0])
		pid, _ := strconv.Atoi(fields[1])
		user := fields[2]
		protocol := strings.ToLower(fields[7])
		name := fields[8]

		state := ""
		if len(fields) > 9 {
			state = strings.Trim(fields[9], "()")
		}

		if state == model.StateListen {
			addr, port := parseAddressPort(name)
			if port == targetPort && listener == nil {
				uptime := getProcessUptime(pid)
				var startTime time.Time
				if uptime > 0 {
					startTime = time.Now().Add(-time.Duration(uptime) * time.Second)
				}
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
						StartTime:     startTime,
					},
				}
			}
		} else if state == model.StateEstablished && strings.Contains(name, "->") {
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
	patternNum, isNum := strconv.Atoi(pattern)

	for _, l := range listeners {
		if isNum == nil && l.Port == patternNum {
			matches = append(matches, l)
			continue
		}
		if isNum == nil && l.PID == patternNum {
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

type lsofEntry struct {
	command  string
	pid      int
	user     string
	protocol string
	port     int
	address  string
	state    string
}

func (s *DarwinScanner) parseLsofOutput(output string) ([]model.Listener, error) {
	entries, connCount, err := parseLsofLines(output)
	if err != nil {
		return nil, err
	}
	return deduplicateEntries(entries, connCount), nil
}

// parseLsofLines parses raw lsof output into entries and connection counts.
func parseLsofLines(output string) ([]lsofEntry, map[int]int, error) {
	var entries []lsofEntry
	connCount := make(map[int]int)

	sc := bufio.NewScanner(strings.NewReader(output))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "COMMAND") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		command := decodeLsofEscapes(fields[0])
		pid, _ := strconv.Atoi(fields[1])
		user := fields[2]
		protocol := strings.ToLower(fields[7])
		name := fields[8]

		state := ""
		if len(fields) > 9 {
			state = strings.Trim(fields[9], "()")
		}

		addr, port := parseLsofName(name)
		if port == 0 {
			continue
		}

		if state == model.StateEstablished {
			connCount[port]++
			continue
		}
		if state != model.StateListen {
			continue
		}

		entries = append(entries, lsofEntry{
			command: command, pid: pid, user: user,
			protocol: protocol, port: port, address: addr, state: state,
		})
	}

	return entries, connCount, sc.Err()
}

// parseLsofName extracts address and port from the NAME field,
// handling both "addr:port" and "addr:port->remote:port" formats.
func parseLsofName(name string) (string, int) {
	if strings.Contains(name, "->") {
		return parseAddressPort(strings.Split(name, "->")[0])
	}
	return parseAddressPort(name)
}

// deduplicateEntries builds model.Listener slice from entries,
// deduplicating by port and enriching with uptime.
func deduplicateEntries(entries []lsofEntry, connCount map[int]int) []model.Listener {
	var listeners []model.Listener
	seen := make(map[int]bool)

	for _, e := range entries {
		if seen[e.port] {
			continue
		}
		seen[e.port] = true

		uptime := getProcessUptime(e.pid)
		var startTime time.Time
		if uptime > 0 {
			startTime = time.Now().Add(-time.Duration(uptime) * time.Second)
		}

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
				StartTime:     startTime,
			},
		})
	}

	return listeners
}

// decodeLsofEscapes decodes \xNN hex escape sequences that lsof uses for
// special characters (e.g. spaces become \x20).
func decodeLsofEscapes(s string) string {
	if !strings.Contains(s, `\x`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if i+3 < len(s) && s[i] == '\\' && s[i+1] == 'x' {
			hi := unhex(s[i+2])
			lo := unhex(s[i+3])
			if hi >= 0 && lo >= 0 {
				b.WriteByte(byte(hi<<4 | lo))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	}
	return -1
}

func parseAddressPort(name string) (string, int) {
	if strings.HasPrefix(name, "[") {
		closeBracket := strings.LastIndex(name, "]")
		if closeBracket == -1 {
			return "", 0
		}
		addr := name[1:closeBracket]
		portStr := strings.TrimPrefix(name[closeBracket+1:], ":")
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return "", 0
		}
		return addr, port
	}

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

func getProcessUptime(pid int) int64 {
	cmd := exec.Command("ps", "-o", "etime=", "-p", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	return parseElapsedTime(strings.TrimSpace(string(output)))
}

func parseElapsedTime(etime string) int64 {
	var days, hours, minutes, seconds int64

	if strings.Contains(etime, "-") {
		parts := strings.SplitN(etime, "-", 2)
		days, _ = strconv.ParseInt(parts[0], 10, 64)
		etime = parts[1]
	}

	parts := strings.Split(etime, ":")
	switch len(parts) {
	case 3:
		hours, _ = strconv.ParseInt(parts[0], 10, 64)
		minutes, _ = strconv.ParseInt(parts[1], 10, 64)
		seconds, _ = strconv.ParseInt(parts[2], 10, 64)
	case 2:
		minutes, _ = strconv.ParseInt(parts[0], 10, 64)
		seconds, _ = strconv.ParseInt(parts[1], 10, 64)
	}

	return days*86400 + hours*3600 + minutes*60 + seconds
}
