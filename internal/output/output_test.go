package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/tasnimzotder/portman/internal/model"
)

// ---------------------------------------------------------------------------
// FormatDuration
// ---------------------------------------------------------------------------

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds int64
		want    string
	}{
		{"zero seconds", 0, "0s"},
		{"sub-minute", 42, "42s"},
		{"exactly one minute", 60, "1m"},
		{"minute with leftover seconds", 90, "1m30s"},
		{"exactly one hour", 3600, "1h"},
		{"hour with minutes and seconds (seconds dropped)", 3661, "1h1m"},
		{"exactly one day", 86400, "1d"},
		{"day with hours", 90061, "1d1h"},
		{"multiple days", 172800, "2d"},
		{"multiple days with hours", 180000, "2d2h"},
		{"one second", 1, "1s"},
		{"59 seconds", 59, "59s"},
		{"two hours even", 7200, "2h"},
		{"two hours thirty minutes", 9000, "2h30m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.seconds)
			if got != tt.want {
				t.Errorf("FormatDuration(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FormatBytes
// ---------------------------------------------------------------------------

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"sub-KB", 500, "500 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"exactly 1 MB", 1048576, "1.0 MB"},
		{"exactly 1 GB", 1073741824, "1.0 GB"},
		{"1023 bytes (boundary)", 1023, "1023 B"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"10 MB", 10 * 1048576, "10.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// truncate
// ---------------------------------------------------------------------------

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"short string unchanged", "hi", 10, "hi"},
		{"exact length", "hello", 5, "hello"},
		{"truncated with ellipsis", "hello world", 8, "hello..."},
		{"max <= 3 no ellipsis", "hello", 3, "hel"},
		{"max 1", "hello", 1, "h"},
		{"empty string", "", 5, ""},
		{"max equals string length", "abc", 3, "abc"},
		{"max 4 with long string", "abcdef", 4, "a..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SortListeners
// ---------------------------------------------------------------------------

func makeListeners() []model.Listener {
	return []model.Listener{
		{
			Port:            8080,
			PID:             200,
			ConnectionCount: 5,
			Process:         &model.Process{User: "bob", UptimeSeconds: 100},
		},
		{
			Port:            80,
			PID:             100,
			ConnectionCount: 10,
			Process:         &model.Process{User: "alice", UptimeSeconds: 300},
		},
		{
			Port:            443,
			PID:             150,
			ConnectionCount: 2,
			Process:         &model.Process{User: "charlie", UptimeSeconds: 200},
		},
	}
}

func TestSortListeners_ByPort(t *testing.T) {
	ls := makeListeners()
	SortListeners(ls, "port")
	if ls[0].Port != 80 || ls[1].Port != 443 || ls[2].Port != 8080 {
		t.Errorf("sort by port: got ports %d, %d, %d", ls[0].Port, ls[1].Port, ls[2].Port)
	}
}

func TestSortListeners_ByPID(t *testing.T) {
	ls := makeListeners()
	SortListeners(ls, "pid")
	if ls[0].PID != 100 || ls[1].PID != 150 || ls[2].PID != 200 {
		t.Errorf("sort by pid: got PIDs %d, %d, %d", ls[0].PID, ls[1].PID, ls[2].PID)
	}
}

func TestSortListeners_ByUser(t *testing.T) {
	ls := makeListeners()
	SortListeners(ls, "user")
	if ls[0].Process.User != "alice" || ls[1].Process.User != "bob" || ls[2].Process.User != "charlie" {
		t.Errorf("sort by user: got users %s, %s, %s",
			ls[0].Process.User, ls[1].Process.User, ls[2].Process.User)
	}
}

func TestSortListeners_ByConns(t *testing.T) {
	ls := makeListeners()
	SortListeners(ls, "conns")
	if ls[0].ConnectionCount != 10 || ls[1].ConnectionCount != 5 || ls[2].ConnectionCount != 2 {
		t.Errorf("sort by conns: got %d, %d, %d",
			ls[0].ConnectionCount, ls[1].ConnectionCount, ls[2].ConnectionCount)
	}
}

func TestSortListeners_ByUptime(t *testing.T) {
	ls := makeListeners()
	SortListeners(ls, "uptime")
	if ls[0].Process.UptimeSeconds != 300 || ls[1].Process.UptimeSeconds != 200 || ls[2].Process.UptimeSeconds != 100 {
		t.Errorf("sort by uptime: got %d, %d, %d",
			ls[0].Process.UptimeSeconds, ls[1].Process.UptimeSeconds, ls[2].Process.UptimeSeconds)
	}
}

func TestSortListeners_DefaultIsPort(t *testing.T) {
	ls := makeListeners()
	SortListeners(ls, "unknown_field")
	if ls[0].Port != 80 || ls[1].Port != 443 || ls[2].Port != 8080 {
		t.Errorf("default sort should be by port: got %d, %d, %d", ls[0].Port, ls[1].Port, ls[2].Port)
	}
}

func TestSortListeners_NilProcess(t *testing.T) {
	ls := []model.Listener{
		{Port: 80, Process: &model.Process{User: "alice", UptimeSeconds: 100}},
		{Port: 443, Process: nil},
	}

	SortListeners(ls, "user")
	if ls[0].Process != nil {
		t.Errorf("nil process should sort before 'alice'")
	}

	ls2 := []model.Listener{
		{Port: 80, Process: nil},
		{Port: 443, Process: &model.Process{UptimeSeconds: 500}},
	}
	SortListeners(ls2, "uptime")
	if ls2[0].Process == nil || ls2[0].Process.UptimeSeconds != 500 {
		t.Errorf("listener with uptime=500 should come first in descending sort")
	}
}

// ---------------------------------------------------------------------------
// TableFormatter.Format
// ---------------------------------------------------------------------------

func TestTableFormat_EmptySlice(t *testing.T) {
	f := NewTableFormatter()
	got := f.Format(nil)
	if !strings.Contains(got, "No listening ports found.") {
		t.Errorf("Format(nil) should contain 'No listening ports found.', got %q", got)
	}

	got = f.Format([]model.Listener{})
	if !strings.Contains(got, "No listening ports found.") {
		t.Errorf("Format([]) should contain 'No listening ports found.', got %q", got)
	}
}

func TestTableFormat_SingleListener(t *testing.T) {
	f := NewTableFormatter()
	ls := []model.Listener{
		{
			Port:            8080,
			Protocol:        "tcp",
			PID:             1234,
			ConnectionCount: 3,
			Process: &model.Process{
				User:          "testuser",
				Command:       "node",
				UptimeSeconds: 3661,
			},
		},
	}

	got := f.Format(ls)

	if !strings.Contains(got, "PORT") {
		t.Error("expected header with PORT")
	}
	if !strings.Contains(got, "PROTO") {
		t.Error("expected header with PROTO")
	}
	if !strings.Contains(got, "8080") {
		t.Error("expected port 8080 in output")
	}
	if !strings.Contains(got, "tcp") {
		t.Error("expected protocol tcp in output")
	}
	if !strings.Contains(got, "1234") {
		t.Error("expected PID 1234 in output")
	}
	if !strings.Contains(got, "testuser") {
		t.Error("expected user testuser in output")
	}
	if !strings.Contains(got, "node") {
		t.Error("expected command node in output")
	}
	if !strings.Contains(got, "1h1m") {
		t.Error("expected uptime 1h1m in output")
	}
}

func TestTableFormat_NoHeader(t *testing.T) {
	f := NewTableFormatter()
	f.NoHeader = true
	ls := []model.Listener{
		{
			Port:     80,
			Protocol: "tcp",
			PID:      1,
			Process:  &model.Process{Command: "nginx"},
		},
	}

	got := f.Format(ls)

	if strings.Contains(got, "PORT") {
		t.Error("NoHeader=true but header was present")
	}
	if !strings.Contains(got, "80") {
		t.Error("expected port 80 in output")
	}
}

func TestTableFormat_NilProcess(t *testing.T) {
	f := NewTableFormatter()
	ls := []model.Listener{
		{
			Port:     9090,
			Protocol: "udp",
			PID:      0,
			Process:  nil,
		},
	}

	got := f.Format(ls)
	// With nil process and PID 0, should show dashes for pid, user, command, uptime
	if !strings.Contains(got, "-") {
		t.Error("expected dash placeholders for nil process fields")
	}
}

func TestTableFormat_ProcessNameFallback(t *testing.T) {
	f := NewTableFormatter()
	ls := []model.Listener{
		{
			Port:     3000,
			Protocol: "tcp",
			PID:      42,
			Process:  &model.Process{Name: "myapp", Command: ""},
		},
	}

	got := f.Format(ls)
	if !strings.Contains(got, "myapp") {
		t.Error("expected process name 'myapp' when command is empty")
	}
}

func TestTableFormat_LongCommandTruncated(t *testing.T) {
	f := NewTableFormatter()
	longCmd := "this-is-a-very-long-command-name-that-exceeds-24-chars"
	ls := []model.Listener{
		{
			Port:     5000,
			Protocol: "tcp",
			PID:      99,
			Process:  &model.Process{Command: longCmd},
		},
	}

	got := f.Format(ls)
	if strings.Contains(got, longCmd) {
		t.Error("long command should have been truncated")
	}
	if !strings.Contains(got, "...") {
		t.Error("truncated command should contain ellipsis")
	}
}

func TestTableFormat_Footer(t *testing.T) {
	f := NewTableFormatter()
	ls := []model.Listener{
		{Port: 80, Protocol: "tcp", PID: 1, Process: &model.Process{Command: "nginx"}},
		{Port: 443, Protocol: "tcp", PID: 1, Process: &model.Process{Command: "nginx"}},
	}

	got := f.Format(ls)
	if !strings.Contains(got, "2 listening ports") {
		t.Error("expected footer with port count")
	}
}

// ---------------------------------------------------------------------------
// TableFormatter.FormatDetail
// ---------------------------------------------------------------------------

func TestTableFormatDetail_Nil(t *testing.T) {
	f := NewTableFormatter()
	got := f.FormatDetail(nil)
	if !strings.Contains(got, "Port not in use.") {
		t.Errorf("FormatDetail(nil) should contain 'Port not in use.', got %q", got)
	}
}

func TestTableFormatDetail_FullListener(t *testing.T) {
	f := NewTableFormatter()
	startTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	l := &model.Listener{
		Port:     8080,
		Protocol: "tcp",
		Address:  "0.0.0.0",
		PID:      1234,
		Process: &model.Process{
			PID:           1234,
			Command:       "node server.js",
			Cmdline:       []string{"node", "server.js", "--port=8080"},
			User:          "deploy",
			UID:           1000,
			StartTime:     startTime,
			UptimeSeconds: 86400,
		},
		Connections: []model.Connection{
			{
				RemoteAddr:      "192.168.1.10",
				RemotePort:      54321,
				State:           "ESTABLISHED",
				DurationSeconds: 120,
			},
			{
				RemoteAddr:      "10.0.0.5",
				RemotePort:      12345,
				State:           "CLOSE_WAIT",
				DurationSeconds: 0,
			},
		},
		ConnectionCount: 2,
		Stats: &model.ProcessStats{
			MemoryRSS:   104857600,
			CPUPercent:   12.5,
			FDCount:      42,
			ThreadCount:  8,
		},
	}

	got := f.FormatDetail(l)

	// Check port header
	if !strings.Contains(got, "Port 8080") {
		t.Error("expected 'Port 8080' header")
	}

	// Check process info
	if !strings.Contains(got, "1234") {
		t.Error("expected PID 1234")
	}
	if !strings.Contains(got, "node server.js") {
		t.Error("expected Command line")
	}
	if !strings.Contains(got, "node server.js --port=8080") {
		t.Error("expected Full cmdline")
	}
	if !strings.Contains(got, "deploy") {
		t.Error("expected user deploy")
	}
	if !strings.Contains(got, "uid: 1000") {
		t.Error("expected uid: 1000")
	}
	if !strings.Contains(got, "1d") {
		t.Error("expected formatted uptime '1d'")
	}

	// Check address with all-interfaces annotation
	if !strings.Contains(got, "0.0.0.0 (all interfaces)") {
		t.Error("expected all interfaces annotation for 0.0.0.0")
	}
	if !strings.Contains(got, "TCP") {
		t.Error("expected protocol TCP (uppercased)")
	}

	// Check connections
	if !strings.Contains(got, "2 established") {
		t.Error("expected connections header with count")
	}
	if !strings.Contains(got, "192.168.1.10:54321") {
		t.Error("expected first remote address")
	}
	if !strings.Contains(got, "ESTABLISHED") {
		t.Error("expected ESTABLISHED state")
	}
	if !strings.Contains(got, "2m") {
		t.Error("expected formatted duration for connection")
	}
	if !strings.Contains(got, "10.0.0.5:12345") {
		t.Error("expected second remote address")
	}

	// Check stats
	if !strings.Contains(got, "Stats") {
		t.Error("expected Stats section")
	}
	if !strings.Contains(got, "100.0 MB") {
		t.Error("expected memory 100.0 MB")
	}
	if !strings.Contains(got, "12.5%") {
		t.Error("expected CPU 12.5%")
	}
	if !strings.Contains(got, "42 open") {
		t.Error("expected FDs 42 open")
	}
	if !strings.Contains(got, "8") {
		t.Error("expected Threads 8")
	}
}

func TestTableFormatDetail_NilProcess(t *testing.T) {
	f := NewTableFormatter()
	l := &model.Listener{
		Port:     3000,
		Protocol: "tcp",
		Address:  "127.0.0.1",
		PID:      0,
		Process:  nil,
	}

	got := f.FormatDetail(l)
	if !strings.Contains(got, "permission denied or process info unavailable") {
		t.Error("expected unavailable process message")
	}
}

func TestTableFormatDetail_IPv6AllInterfaces(t *testing.T) {
	f := NewTableFormatter()
	l := &model.Listener{
		Port:     443,
		Protocol: "tcp",
		Address:  "::",
		Process:  &model.Process{PID: 1, Command: "nginx"},
	}

	got := f.FormatDetail(l)
	if !strings.Contains(got, ":: (all interfaces)") {
		t.Error("expected all interfaces annotation for '::'")
	}
}

func TestTableFormatDetail_UptimeWithoutStartTime(t *testing.T) {
	f := NewTableFormatter()
	l := &model.Listener{
		Port:     80,
		Protocol: "tcp",
		Address:  "127.0.0.1",
		Process: &model.Process{
			PID:           5,
			Command:       "test",
			UptimeSeconds: 7200,
		},
	}

	got := f.FormatDetail(l)
	if !strings.Contains(got, "Uptime") || !strings.Contains(got, "2h") {
		t.Error("expected 'Uptime' with '2h' when StartTime is zero")
	}
}

func TestTableFormatDetail_NoConnections(t *testing.T) {
	f := NewTableFormatter()
	l := &model.Listener{
		Port:     80,
		Protocol: "tcp",
		Address:  "127.0.0.1",
		Process:  &model.Process{PID: 1, Command: "test"},
	}

	got := f.FormatDetail(l)
	if strings.Contains(got, "Connections") {
		t.Error("should not contain Connections section when there are none")
	}
}

func TestTableFormatDetail_NoStats(t *testing.T) {
	f := NewTableFormatter()
	l := &model.Listener{
		Port:     80,
		Protocol: "tcp",
		Address:  "127.0.0.1",
		Process:  &model.Process{PID: 1, Command: "test"},
	}

	got := f.FormatDetail(l)
	if strings.Contains(got, "Stats") {
		t.Error("should not contain Stats section when stats is nil")
	}
}

// ---------------------------------------------------------------------------
// TableFormatter.FormatTree
// ---------------------------------------------------------------------------

func TestTableFormatTree_EmptyListeners(t *testing.T) {
	f := NewTableFormatter()
	got := f.FormatTree(nil, 1234)
	if !strings.Contains(got, "No ports found for PID 1234") {
		t.Error("expected 'No ports found' message")
	}
}

func TestTableFormatTree_SinglePort(t *testing.T) {
	f := NewTableFormatter()
	ls := []model.Listener{
		{
			Port:            8080,
			Protocol:        "tcp",
			PID:             1234,
			ConnectionCount: 5,
			Process:         &model.Process{Command: "node", User: "dev", UptimeSeconds: 3600},
		},
	}

	got := f.FormatTree(ls, 1234)
	if !strings.Contains(got, "node") {
		t.Error("expected process name 'node'")
	}
	if !strings.Contains(got, "PID 1234") {
		t.Error("expected 'PID 1234'")
	}
	if !strings.Contains(got, ":8080") {
		t.Error("expected port :8080")
	}
	if !strings.Contains(got, "5 conns") {
		t.Error("expected '5 conns'")
	}
}

func TestTableFormatTree_MultiplePorts(t *testing.T) {
	f := NewTableFormatter()
	ls := []model.Listener{
		{Port: 3000, Protocol: "tcp", PID: 1234, Process: &model.Process{Command: "node", User: "dev"}},
		{Port: 3001, Protocol: "tcp", PID: 1234, Process: &model.Process{Command: "node", User: "dev"}},
		{Port: 8080, Protocol: "tcp", PID: 1234, Process: &model.Process{Command: "node", User: "dev"}},
	}

	got := f.FormatTree(ls, 1234)
	if !strings.Contains(got, "3 ports") {
		t.Error("expected '3 ports' in footer")
	}
}

// ---------------------------------------------------------------------------
// TableFormatter.Format with Grouped
// ---------------------------------------------------------------------------

func TestTableFormat_Grouped(t *testing.T) {
	f := NewTableFormatter()
	f.Grouped = true
	ls := []model.Listener{
		{Port: 80, Protocol: "tcp", PID: 100, Process: &model.Process{Command: "nginx", User: "root"}},
		{Port: 443, Protocol: "tcp", PID: 100, Process: &model.Process{Command: "nginx", User: "root"}},
		{Port: 3000, Protocol: "tcp", PID: 200, Process: &model.Process{Command: "node", User: "dev"}},
	}

	got := f.Format(ls)
	if !strings.Contains(got, "nginx") {
		t.Error("expected 'nginx' group header")
	}
	if !strings.Contains(got, "node") {
		t.Error("expected 'node' group header")
	}
	if !strings.Contains(got, "3 ports across 2 processes") {
		t.Error("expected grouped footer")
	}
}

// ---------------------------------------------------------------------------
// GroupByProcess
// ---------------------------------------------------------------------------

func TestGroupByProcess(t *testing.T) {
	ls := []model.Listener{
		{Port: 80, PID: 100, Process: &model.Process{Command: "nginx", User: "root"}},
		{Port: 443, PID: 100, Process: &model.Process{Command: "nginx", User: "root"}},
		{Port: 3000, PID: 200, Process: &model.Process{Command: "node", User: "dev"}},
	}

	groups := GroupByProcess(ls)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].PID != 100 || len(groups[0].Listeners) != 2 {
		t.Error("first group should be PID 100 with 2 listeners")
	}
	if groups[1].PID != 200 || len(groups[1].Listeners) != 1 {
		t.Error("second group should be PID 200 with 1 listener")
	}
}

// ---------------------------------------------------------------------------
// JSONFormatter.Format
// ---------------------------------------------------------------------------

func TestJSONFormat_ValidOutput(t *testing.T) {
	f := NewJSONFormatter(false)
	ls := []model.Listener{
		{
			Port:     8080,
			Protocol: "tcp",
			Address:  "0.0.0.0",
			PID:      1234,
			Process: &model.Process{
				PID:     1234,
				Command: "node",
				User:    "test",
			},
			ConnectionCount: 1,
		},
	}

	got, err := f.Format(ls)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	var result model.ScanResult
	if err := json.Unmarshal([]byte(got), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if len(result.Listeners) != 1 {
		t.Errorf("expected 1 listener, got %d", len(result.Listeners))
	}
	if result.Listeners[0].Port != 8080 {
		t.Errorf("expected port 8080, got %d", result.Listeners[0].Port)
	}
	if result.Platform == "" {
		t.Error("expected platform to be set")
	}
	if result.ScanTime.IsZero() {
		t.Error("expected scanTime to be set")
	}
}

func TestJSONFormat_EmptyListeners(t *testing.T) {
	f := NewJSONFormatter(false)
	got, err := f.Format([]model.Listener{})
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	var result model.ScanResult
	if err := json.Unmarshal([]byte(got), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(result.Listeners) != 0 {
		t.Errorf("expected 0 listeners, got %d", len(result.Listeners))
	}
}

func TestJSONFormat_Pretty(t *testing.T) {
	f := NewJSONFormatter(true)
	ls := []model.Listener{
		{Port: 80, Protocol: "tcp"},
	}

	got, err := f.Format(ls)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	if !strings.Contains(got, "\n") {
		t.Error("pretty JSON should contain newlines")
	}
	if !strings.Contains(got, "  ") {
		t.Error("pretty JSON should contain indentation")
	}
}

func TestJSONFormat_NotPretty(t *testing.T) {
	f := NewJSONFormatter(false)
	ls := []model.Listener{
		{Port: 80, Protocol: "tcp"},
	}

	got, err := f.Format(ls)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}

	if strings.Contains(got, "\n") {
		t.Error("compact JSON should not contain newlines")
	}
}

// ---------------------------------------------------------------------------
// JSONFormatter.FormatSingle
// ---------------------------------------------------------------------------

func TestJSONFormatSingle_Nil(t *testing.T) {
	f := NewJSONFormatter(false)
	got, err := f.FormatSingle(nil)
	if err != nil {
		t.Fatalf("FormatSingle(nil) returned error: %v", err)
	}
	if got != "{}" {
		t.Errorf("FormatSingle(nil) = %q, want %q", got, "{}")
	}
}

func TestJSONFormatSingle_Valid(t *testing.T) {
	f := NewJSONFormatter(false)
	l := &model.Listener{
		Port:     443,
		Protocol: "tcp",
		Address:  "::",
		PID:      555,
		Process: &model.Process{
			PID:     555,
			Command: "nginx",
			User:    "root",
			UID:     0,
		},
		ConnectionCount: 10,
	}

	got, err := f.FormatSingle(l)
	if err != nil {
		t.Fatalf("FormatSingle returned error: %v", err)
	}

	var parsed model.Listener
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed.Port != 443 {
		t.Errorf("expected port 443, got %d", parsed.Port)
	}
	if parsed.Process == nil {
		t.Fatal("expected process to be present")
	}
	if parsed.Process.Command != "nginx" {
		t.Errorf("expected command 'nginx', got %q", parsed.Process.Command)
	}
}

func TestJSONFormatSingle_Pretty(t *testing.T) {
	f := NewJSONFormatter(true)
	l := &model.Listener{
		Port:     80,
		Protocol: "tcp",
	}

	got, err := f.FormatSingle(l)
	if err != nil {
		t.Fatalf("FormatSingle returned error: %v", err)
	}

	if !strings.Contains(got, "\n") {
		t.Error("pretty JSON should contain newlines")
	}
}

func TestJSONFormatSingle_WithConnections(t *testing.T) {
	f := NewJSONFormatter(false)
	l := &model.Listener{
		Port:     8080,
		Protocol: "tcp",
		Address:  "127.0.0.1",
		PID:      100,
		Connections: []model.Connection{
			{
				LocalAddr:  "127.0.0.1",
				LocalPort:  8080,
				RemoteAddr: "10.0.0.1",
				RemotePort: 50000,
				State:      "ESTABLISHED",
			},
		},
		ConnectionCount: 1,
	}

	got, err := f.FormatSingle(l)
	if err != nil {
		t.Fatalf("FormatSingle returned error: %v", err)
	}

	var parsed model.Listener
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(parsed.Connections) != 1 {
		t.Errorf("expected 1 connection, got %d", len(parsed.Connections))
	}
	if parsed.Connections[0].State != "ESTABLISHED" {
		t.Errorf("expected state ESTABLISHED, got %q", parsed.Connections[0].State)
	}
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func TestNewTableFormatter(t *testing.T) {
	f := NewTableFormatter()
	if f == nil {
		t.Fatal("NewTableFormatter() returned nil")
	}
	if f.NoHeader {
		t.Error("default NoHeader should be false")
	}
	if f.Grouped {
		t.Error("default Grouped should be false")
	}
}

func TestNewJSONFormatter(t *testing.T) {
	f := NewJSONFormatter(true)
	if f == nil {
		t.Fatal("NewJSONFormatter() returned nil")
	}
	if !f.Pretty {
		t.Error("expected Pretty=true")
	}

	f2 := NewJSONFormatter(false)
	if f2.Pretty {
		t.Error("expected Pretty=false")
	}
}
