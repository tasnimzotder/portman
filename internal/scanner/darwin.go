//go:build darwin

package scanner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tasnimzotder/portman/internal/model"
)

const cmdTimeout = 10 * time.Second

type DarwinScanner struct{ opts Options }

func NewDarwinScanner(opts Options) *DarwinScanner { return &DarwinScanner{opts: opts} }

// An exit status of 1 with no output or diagnostics means no matching sockets.
// Any other failure is an error, never evidence that a port is free.
func runLsof(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "lsof", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return "", fmt.Errorf("lsof: %w", ctx.Err())
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 && len(out) == 0 && stderr.Len() == 0 {
			return "", nil
		}
		return "", fmt.Errorf("lsof failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if stderr.Len() != 0 {
		return "", fmt.Errorf("lsof: %s", strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func (s *DarwinScanner) scan(ctx context.Context, port int) ([]socketRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()
	selector := ""
	if s.opts.IncludeTCP && !s.opts.IncludeUDP {
		selector = "TCP"
	}
	if s.opts.IncludeUDP && !s.opts.IncludeTCP {
		selector = "UDP"
	}
	if !s.opts.IncludeTCP && !s.opts.IncludeUDP {
		return nil, nil
	}
	if port > 0 {
		selector += fmt.Sprintf(":%d", port)
	}
	args := []string{"-n", "-P", "-F0pcuLfPtnT", "-i" + selector}
	out, err := runLsof(ctx, args...)
	if err != nil {
		return nil, err
	}
	records, err := parseLsofFields(out)
	if err != nil {
		return nil, err
	}
	filtered := records[:0]
	for _, r := range records {
		if !s.opts.IncludeIPv6 && r.family == "IPv6" {
			continue
		}
		if (r.protocol == model.ProtoTCP && s.opts.IncludeTCP) || (r.protocol == model.ProtoUDP && s.opts.IncludeUDP) {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

func (s *DarwinScanner) ListListeners() ([]model.Listener, error) {
	records, err := s.scan(context.Background(), 0)
	if err != nil {
		return nil, err
	}
	listeners := buildListeners(records, true)
	for i := range listeners {
		listeners[i].Connections = nil
	}
	return listeners, nil
}

func (s *DarwinScanner) GetPort(port int) (*model.Listener, error) {
	return s.GetPortContext(context.Background(), port)
}

// GetPortContext allows wait deadlines and cancellation to stop lsof itself.
func (s *DarwinScanner) GetPortContext(ctx context.Context, port int) (*model.Listener, error) {
	return s.getPort(ctx, port, true)
}

// GetPortStatusContext checks occupancy without choosing an owner for an action.
func (s *DarwinScanner) GetPortStatusContext(ctx context.Context, port int) (*model.Listener, error) {
	return s.getPort(ctx, port, false)
}

func (s *DarwinScanner) getPort(ctx context.Context, port int, strict bool) (*model.Listener, error) {
	records, err := s.scan(ctx, port)
	if err != nil {
		return nil, err
	}
	listeners := buildListeners(records, false)
	var chosen *model.Listener
	for i := range listeners {
		if listeners[i].Port != port {
			continue
		}
		if strict && chosen != nil && (chosen.PID != listeners[i].PID || chosen.Protocol != listeners[i].Protocol) {
			return nil, fmt.Errorf("port %d has multiple owners or protocols; use a port range or pid lookup to inspect each binding", port)
		}
		if chosen == nil {
			chosen = &listeners[i]
		}
	}
	if strict && chosen != nil && s.opts.FetchStats {
		uptime := batchGetUptimes([]int{chosen.PID})[chosen.PID]
		chosen.Process.UptimeSeconds = uptime
		if uptime > 0 {
			chosen.Process.StartTime = time.Now().Add(-time.Duration(uptime) * time.Second)
		}
		chosen.Stats = s.getProcessStats(chosen.PID)
	}
	return chosen, nil
}

func (s *DarwinScanner) ListByPID(pid int) ([]model.Listener, error) {
	listeners, err := s.ListListeners()
	if err != nil {
		return nil, err
	}
	matches := make([]model.Listener, 0)
	for _, l := range listeners {
		if l.PID == pid {
			matches = append(matches, l)
		}
	}
	return matches, nil
}
func (s *DarwinScanner) FindByPattern(pattern string) ([]model.Listener, error) {
	listeners, err := s.ListListeners()
	if err != nil {
		return nil, err
	}
	matches := make([]model.Listener, 0)
	pattern = strings.ToLower(pattern)
	number, numericErr := strconv.Atoi(pattern)
	for _, l := range listeners {
		if numericErr == nil && (l.Port == number || l.PID == number) || strings.Contains(strings.ToLower(l.ProcessName()), pattern) || strings.Contains(strings.ToLower(l.ProcessUser()), pattern) {
			matches = append(matches, l)
		}
	}
	return matches, nil
}

// socketRecord is one file record from lsof's NUL-delimited field output.
type socketRecord struct {
	pid, uid                                     int
	command, user, family, protocol, name, state string
}

func parseLsofFields(out string) ([]socketRecord, error) {
	if out != "" && !strings.Contains(out, "\x00") {
		return nil, fmt.Errorf("invalid lsof field output")
	}
	var records []socketRecord
	var process, file socketRecord
	inFile := false
	flush := func() {
		if inFile && file.name != "" {
			records = append(records, file)
		}
		inFile = false
	}
	for _, raw := range strings.Split(out, "\x00") {
		field := strings.TrimLeft(raw, "\n")
		if field == "" {
			continue
		}
		value := field[1:]
		switch field[0] {
		case 'p':
			flush()
			pid, err := strconv.Atoi(value)
			if err != nil || pid <= 0 {
				return nil, fmt.Errorf("invalid lsof PID %q", value)
			}
			process = socketRecord{pid: pid}
		case 'c':
			process.command = decodeLsofEscapes(value)
		case 'u':
			process.uid, _ = strconv.Atoi(value)
		case 'L':
			process.user = decodeLsofEscapes(value)
		case 'f':
			flush()
			file = process
			inFile = true
		case 't':
			file.family = value
		case 'P':
			file.protocol = strings.ToLower(value)
		case 'n':
			file.name = value
		case 'T':
			if strings.HasPrefix(value, "ST=") {
				file.state = strings.TrimPrefix(value, "ST=")
			}
		}
	}
	flush()
	return records, nil
}

func isBound(r socketRecord) bool {
	return r.state == model.StateListen && r.protocol == model.ProtoTCP || r.protocol == model.ProtoUDP
}
func buildListeners(records []socketRecord, enrich bool) []model.Listener {
	listeners := make([]model.Listener, 0)
	seen := make(map[model.BindingKey]bool)
	pids := make(map[int]bool)
	type connectionKey struct {
		pid, port int
		protocol  string
	}
	connections := make(map[connectionKey][]socketRecord)
	for _, r := range records {
		if r.state != model.StateEstablished {
			continue
		}
		endpoints := strings.SplitN(r.name, "->", 2)
		if len(endpoints) != 2 {
			continue
		}
		_, port := parseAddressPort(endpoints[0])
		key := connectionKey{r.pid, port, r.protocol}
		connections[key] = append(connections[key], r)
	}
	for _, r := range records {
		if !isBound(r) {
			continue
		}
		addr, port := parseAddressPort(strings.SplitN(r.name, "->", 2)[0])
		if port < 1 || port > 65535 {
			continue
		}
		if r.family == "IPv6" && r.name == fmt.Sprintf("*:%d", port) {
			addr = "::"
		}
		l := model.Listener{Port: port, Protocol: r.protocol, Address: addr, PID: r.pid,
			Process: &model.Process{PID: r.pid, UID: r.uid, Name: r.command, Command: r.command, User: r.user}}
		key := l.Key()
		if seen[key] {
			continue
		}
		seen[key] = true
		pids[r.pid] = true
		for _, c := range connections[connectionKey{r.pid, port, r.protocol}] {
			if c.family != r.family {
				continue
			}
			endpoints := strings.SplitN(c.name, "->", 2)
			if len(endpoints) != 2 {
				continue
			}
			local, lp := parseAddressPort(endpoints[0])
			remote, rp := parseAddressPort(endpoints[1])
			if lp != port || addr != "0.0.0.0" && addr != "::" && addr != local {
				continue
			}
			l.Connections = append(l.Connections, model.Connection{LocalAddr: local, LocalPort: lp, RemoteAddr: remote, RemotePort: rp, State: c.state})
		}
		l.ConnectionCount = len(l.Connections)
		listeners = append(listeners, l)
	}
	if enrich {
		ids := make([]int, 0, len(pids))
		for pid := range pids {
			ids = append(ids, pid)
		}
		uptimes := batchGetUptimes(ids)
		for i := range listeners {
			p := listeners[i].Process
			p.UptimeSeconds = uptimes[p.PID]
			if p.UptimeSeconds > 0 {
				p.StartTime = time.Now().Add(-time.Duration(p.UptimeSeconds) * time.Second)
			}
		}
	}
	sort.Slice(listeners, func(i, j int) bool {
		a, b := listeners[i], listeners[j]
		if a.Port != b.Port {
			return a.Port < b.Port
		}
		if a.Protocol != b.Protocol {
			return a.Protocol < b.Protocol
		}
		if a.PID != b.PID {
			return a.PID < b.PID
		}
		return a.Address < b.Address
	})
	return listeners
}

func (s *DarwinScanner) getProcessStats(pid int) *model.ProcessStats {
	stats := &model.ProcessStats{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id := strconv.Itoa(pid)
	out, err := exec.CommandContext(ctx, "ps", "-o", "rss=,%cpu=", "-p", id).Output()
	if err == nil {
		f := strings.Fields(string(out))
		if len(f) >= 2 {
			rss, _ := strconv.ParseInt(f[0], 10, 64)
			stats.MemoryRSS = rss * 1024
			stats.CPUPercent, _ = strconv.ParseFloat(f[1], 64)
		}
	}
	out, err = exec.CommandContext(ctx, "lsof", "-p", id, "-Ff").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "f") {
				fd := strings.TrimSuffix(strings.TrimPrefix(line, "f"), "u")
				if _, err := strconv.Atoi(fd); err == nil {
					stats.FDCount++
				}
			}
		}
	}
	out, err = exec.CommandContext(ctx, "ps", "-M", "-p", id).Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 1 {
			stats.ThreadCount = len(lines) - 1
		}
	}
	return stats
}

func batchGetUptimes(pids []int) map[int]int64 {
	result := make(map[int]int64, len(pids))
	if len(pids) == 0 {
		return result
	}

	// Build comma-separated PID list for ps -p
	pidStrs := make([]string, len(pids))
	for i, pid := range pids {
		pidStrs[i] = strconv.Itoa(pid)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ps", "-o", "pid=,etime=", "-p", strings.Join(pidStrs, ","))
	output, err := cmd.Output()
	if err != nil {
		// Fallback: return empty (all uptimes will be 0)
		return result
	}

	// Parse output lines: "  1234   1-02:03:04"
	sc := bufio.NewScanner(strings.NewReader(string(output)))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		result[pid] = parseElapsedTime(fields[1])
	}

	return result
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

func newPlatformScanner(opts Options) (Scanner, error) { return NewDarwinScanner(opts), nil }
