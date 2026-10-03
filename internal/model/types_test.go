package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestListenerJSONSerialization(t *testing.T) {
	start := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	listener := Listener{
		Port:     8080,
		Protocol: "tcp",
		Address:  "0.0.0.0",
		PID:      1234,
		Process: &Process{
			PID:           1234,
			Name:          "myapp",
			Command:       "myapp serve",
			Cmdline:       []string{"myapp", "serve", "--port=8080"},
			User:          "root",
			UID:           0,
			StartTime:     start,
			UptimeSeconds: 3600,
		},
		Connections: []Connection{
			{
				LocalAddr:       "127.0.0.1",
				LocalPort:       8080,
				RemoteAddr:      "192.168.1.10",
				RemotePort:      54321,
				State:           "ESTABLISHED",
				DurationSeconds: 120,
			},
		},
		ConnectionCount: 1,
		Stats: &ProcessStats{
			MemoryRSS:   1048576,
			CPUPercent:   2.5,
			FDCount:      42,
			ThreadCount:  8,
		},
	}

	data, err := json.Marshal(listener)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var got Listener
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got.Port != listener.Port {
		t.Errorf("Port: got %d, want %d", got.Port, listener.Port)
	}
	if got.Protocol != listener.Protocol {
		t.Errorf("Protocol: got %q, want %q", got.Protocol, listener.Protocol)
	}
	if got.Address != listener.Address {
		t.Errorf("Address: got %q, want %q", got.Address, listener.Address)
	}
	if got.PID != listener.PID {
		t.Errorf("PID: got %d, want %d", got.PID, listener.PID)
	}
	if got.ConnectionCount != listener.ConnectionCount {
		t.Errorf("ConnectionCount: got %d, want %d", got.ConnectionCount, listener.ConnectionCount)
	}

	if got.Process == nil {
		t.Fatal("Process: got nil, want non-nil")
	}
	if got.Process.Name != "myapp" {
		t.Errorf("Process.Name: got %q, want %q", got.Process.Name, "myapp")
	}
	if got.Process.User != "root" {
		t.Errorf("Process.User: got %q, want %q", got.Process.User, "root")
	}
	if got.Process.UptimeSeconds != 3600 {
		t.Errorf("Process.UptimeSeconds: got %d, want %d", got.Process.UptimeSeconds, 3600)
	}
	if !got.Process.StartTime.Equal(start) {
		t.Errorf("Process.StartTime: got %v, want %v", got.Process.StartTime, start)
	}

	if len(got.Connections) != 1 {
		t.Fatalf("Connections: got %d, want 1", len(got.Connections))
	}
	if got.Connections[0].State != "ESTABLISHED" {
		t.Errorf("Connection.State: got %q, want %q", got.Connections[0].State, "ESTABLISHED")
	}
	if got.Connections[0].DurationSeconds != 120 {
		t.Errorf("Connection.DurationSeconds: got %d, want 120", got.Connections[0].DurationSeconds)
	}

	if got.Stats == nil {
		t.Fatal("Stats: got nil, want non-nil")
	}
	if got.Stats.MemoryRSS != 1048576 {
		t.Errorf("Stats.MemoryRSS: got %d, want 1048576", got.Stats.MemoryRSS)
	}
	if got.Stats.CPUPercent != 2.5 {
		t.Errorf("Stats.CPUPercent: got %f, want 2.5", got.Stats.CPUPercent)
	}
}

func TestProcessJSONWithAllFields(t *testing.T) {
	start := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	p := Process{
		PID:           9999,
		Name:          "nginx",
		Command:       "nginx -g daemon off",
		Cmdline:       []string{"nginx", "-g", "daemon off"},
		User:          "www-data",
		UID:           33,
		StartTime:     start,
		UptimeSeconds: 86400,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var got Process
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got.PID != 9999 {
		t.Errorf("PID: got %d, want 9999", got.PID)
	}
	if got.Name != "nginx" {
		t.Errorf("Name: got %q, want %q", got.Name, "nginx")
	}
	if got.Command != "nginx -g daemon off" {
		t.Errorf("Command: got %q, want %q", got.Command, "nginx -g daemon off")
	}
	if len(got.Cmdline) != 3 {
		t.Fatalf("Cmdline: got %d items, want 3", len(got.Cmdline))
	}
	if got.User != "www-data" {
		t.Errorf("User: got %q, want %q", got.User, "www-data")
	}
	if got.UID != 33 {
		t.Errorf("UID: got %d, want 33", got.UID)
	}
	if !got.StartTime.Equal(start) {
		t.Errorf("StartTime: got %v, want %v", got.StartTime, start)
	}
	if got.UptimeSeconds != 86400 {
		t.Errorf("UptimeSeconds: got %d, want 86400", got.UptimeSeconds)
	}
}

func TestProcessJSONWithZeroOptionalFields(t *testing.T) {
	p := Process{
		PID:  42,
		Name: "minimal",
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var got Process
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got.PID != 42 {
		t.Errorf("PID: got %d, want 42", got.PID)
	}
	if got.Name != "minimal" {
		t.Errorf("Name: got %q, want %q", got.Name, "minimal")
	}
	if got.Command != "" {
		t.Errorf("Command: got %q, want empty", got.Command)
	}
	if got.User != "" {
		t.Errorf("User: got %q, want empty", got.User)
	}
	if got.UptimeSeconds != 0 {
		t.Errorf("UptimeSeconds: got %d, want 0", got.UptimeSeconds)
	}
}

func TestScanResultJSONSerialization(t *testing.T) {
	scanTime := time.Date(2025, 3, 10, 8, 0, 0, 0, time.UTC)

	sr := ScanResult{
		Listeners: []Listener{
			{Port: 80, Protocol: "tcp", Address: "0.0.0.0", PID: 100, ConnectionCount: 5},
			{Port: 443, Protocol: "tcp", Address: "0.0.0.0", PID: 100, ConnectionCount: 12},
		},
		ScanTime: scanTime,
		Platform: "darwin",
		Hostname: "myhost",
	}

	data, err := json.Marshal(sr)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var got ScanResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(got.Listeners) != 2 {
		t.Fatalf("Listeners: got %d, want 2", len(got.Listeners))
	}
	if got.Listeners[0].Port != 80 {
		t.Errorf("Listeners[0].Port: got %d, want 80", got.Listeners[0].Port)
	}
	if got.Listeners[1].Port != 443 {
		t.Errorf("Listeners[1].Port: got %d, want 443", got.Listeners[1].Port)
	}
	if !got.ScanTime.Equal(scanTime) {
		t.Errorf("ScanTime: got %v, want %v", got.ScanTime, scanTime)
	}
	if got.Platform != "darwin" {
		t.Errorf("Platform: got %q, want %q", got.Platform, "darwin")
	}
	if got.Hostname != "myhost" {
		t.Errorf("Hostname: got %q, want %q", got.Hostname, "myhost")
	}
}

func TestOmitemptyFields(t *testing.T) {
	t.Run("Listener_without_optional_fields", func(t *testing.T) {
		l := Listener{
			Port:            3000,
			Protocol:        "tcp",
			Address:         "127.0.0.1",
			PID:             500,
			ConnectionCount: 0,
			// Process, Connections, and Stats are all nil/empty
		}

		data, err := json.Marshal(l)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		raw := string(data)

		// process is omitempty so it should not appear
		if contains(raw, `"process"`) {
			t.Error("expected process to be omitted from JSON")
		}
		// connections is omitempty so it should not appear
		if contains(raw, `"connections"`) {
			t.Error("expected connections to be omitted from JSON")
		}
		// stats is omitempty so it should not appear
		if contains(raw, `"stats"`) {
			t.Error("expected stats to be omitted from JSON")
		}
		// port should still be present (not omitempty)
		if !contains(raw, `"port"`) {
			t.Error("expected port to be present in JSON")
		}
	})

	t.Run("Listener_with_optional_fields", func(t *testing.T) {
		l := Listener{
			Port:     3000,
			Protocol: "tcp",
			Address:  "127.0.0.1",
			PID:      500,
			Process: &Process{
				PID:  500,
				Name: "node",
			},
			Connections: []Connection{
				{LocalAddr: "127.0.0.1", LocalPort: 3000},
			},
			ConnectionCount: 1,
			Stats: &ProcessStats{
				MemoryRSS: 2048,
			},
		}

		data, err := json.Marshal(l)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		raw := string(data)

		if !contains(raw, `"process"`) {
			t.Error("expected process to be present in JSON")
		}
		if !contains(raw, `"connections"`) {
			t.Error("expected connections to be present in JSON")
		}
		if !contains(raw, `"stats"`) {
			t.Error("expected stats to be present in JSON")
		}
	})

	t.Run("Connection_DurationSeconds_omitempty", func(t *testing.T) {
		c := Connection{
			LocalAddr:  "127.0.0.1",
			LocalPort:  8080,
			RemoteAddr: "10.0.0.1",
			RemotePort: 5000,
			State:      "ESTABLISHED",
			// DurationSeconds is 0, should be omitted
		}

		data, err := json.Marshal(c)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		raw := string(data)
		if contains(raw, `"durationSeconds"`) {
			t.Error("expected durationSeconds to be omitted when zero")
		}

		// Now set it to a non-zero value
		c.DurationSeconds = 60
		data, err = json.Marshal(c)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		raw = string(data)
		if !contains(raw, `"durationSeconds"`) {
			t.Error("expected durationSeconds to be present when non-zero")
		}
	})
}

// ---------------------------------------------------------------------------
// Process.DisplayName / DisplayUser
// ---------------------------------------------------------------------------

func TestProcess_DisplayName(t *testing.T) {
	tests := []struct {
		name    string
		process *Process
		want    string
	}{
		{"nil process", nil, "unknown"},
		{"empty fields", &Process{}, "unknown"},
		{"command only", &Process{Command: "nginx"}, "nginx"},
		{"name only", &Process{Name: "node"}, "node"},
		{"command preferred over name", &Process{Command: "nginx -g daemon off", Name: "nginx"}, "nginx -g daemon off"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.process.DisplayName()
			if got != tt.want {
				t.Errorf("DisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcess_DisplayUser(t *testing.T) {
	tests := []struct {
		name    string
		process *Process
		want    string
	}{
		{"nil process", nil, "-"},
		{"empty user", &Process{}, "-"},
		{"has user", &Process{User: "root"}, "root"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.process.DisplayUser()
			if got != tt.want {
				t.Errorf("DisplayUser() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListener_ProcessName(t *testing.T) {
	l := Listener{Process: &Process{Command: "node", Name: "node"}}
	if got := l.ProcessName(); got != "node" {
		t.Errorf("ProcessName() = %q, want %q", got, "node")
	}

	l2 := Listener{Process: nil}
	if got := l2.ProcessName(); got != "unknown" {
		t.Errorf("ProcessName() with nil process = %q, want %q", got, "unknown")
	}
}

// ---------------------------------------------------------------------------
// FilterByProtocol
// ---------------------------------------------------------------------------

func TestFilterByProtocol(t *testing.T) {
	listeners := []Listener{
		{Port: 80, Protocol: ProtoTCP},
		{Port: 53, Protocol: ProtoUDP},
		{Port: 443, Protocol: ProtoTCP},
	}

	// No filter
	result := FilterByProtocol(listeners, false, false)
	if len(result) != 3 {
		t.Errorf("no filter: expected 3, got %d", len(result))
	}

	// TCP only
	result = FilterByProtocol(listeners, true, false)
	if len(result) != 2 {
		t.Errorf("tcp only: expected 2, got %d", len(result))
	}

	// UDP only
	result = FilterByProtocol(listeners, false, true)
	if len(result) != 1 {
		t.Errorf("udp only: expected 1, got %d", len(result))
	}

	// Both (nothing matches)
	result = FilterByProtocol(listeners, true, true)
	if len(result) != 0 {
		t.Errorf("both: expected 0, got %d", len(result))
	}
}

// contains checks if substr is present in s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
