//go:build darwin

package scanner

import (
	"testing"
)

// ---------------------------------------------------------------------------
// parseAddressPort
// ---------------------------------------------------------------------------

func TestParseAddressPort_IPv4(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantAddr string
		wantPort int
	}{
		{
			name:     "standard IPv4 with port",
			input:    "127.0.0.1:3000",
			wantAddr: "127.0.0.1",
			wantPort: 3000,
		},
		{
			name:     "local address with high port",
			input:    "10.0.0.1:54321",
			wantAddr: "10.0.0.1",
			wantPort: 54321,
		},
		{
			name:     "standard HTTP port",
			input:    "192.168.1.100:80",
			wantAddr: "192.168.1.100",
			wantPort: 80,
		},
		{
			name:     "port 443",
			input:    "0.0.0.0:443",
			wantAddr: "0.0.0.0",
			wantPort: 443,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, port := parseAddressPort(tt.input)
			if addr != tt.wantAddr || port != tt.wantPort {
				t.Errorf("parseAddressPort(%q) = (%q, %d), want (%q, %d)",
					tt.input, addr, port, tt.wantAddr, tt.wantPort)
			}
		})
	}
}

func TestParseAddressPort_IPv6(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantAddr string
		wantPort int
	}{
		{
			name:     "IPv6 loopback",
			input:    "[::1]:3000",
			wantAddr: "::1",
			wantPort: 3000,
		},
		{
			name:     "IPv6 link-local",
			input:    "[fe80::1]:22000",
			wantAddr: "fe80::1",
			wantPort: 22000,
		},
		{
			name:     "IPv6 all-zeros",
			input:    "[::]:8080",
			wantAddr: "::",
			wantPort: 8080,
		},
		{
			name:     "IPv6 full address",
			input:    "[2001:db8::1]:443",
			wantAddr: "2001:db8::1",
			wantPort: 443,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, port := parseAddressPort(tt.input)
			if addr != tt.wantAddr || port != tt.wantPort {
				t.Errorf("parseAddressPort(%q) = (%q, %d), want (%q, %d)",
					tt.input, addr, port, tt.wantAddr, tt.wantPort)
			}
		})
	}
}

func TestParseAddressPort_Wildcard(t *testing.T) {
	addr, port := parseAddressPort("*:80")
	if addr != "0.0.0.0" {
		t.Errorf("wildcard address: got %q, want %q", addr, "0.0.0.0")
	}
	if port != 80 {
		t.Errorf("wildcard port: got %d, want %d", port, 80)
	}
}

func TestParseAddressPort_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "no colon separator", input: "127.0.0.1"},
		{name: "empty string", input: ""},
		{name: "non-numeric port", input: "127.0.0.1:abc"},
		{name: "IPv6 missing closing bracket", input: "[::1:3000"},
		{name: "IPv6 non-numeric port", input: "[::1]:xyz"},
		{name: "just a colon", input: ":"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, port := parseAddressPort(tt.input)
			if addr != "" || port != 0 {
				t.Errorf("parseAddressPort(%q) = (%q, %d), want (\"\", 0)",
					tt.input, addr, port)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseElapsedTime
// ---------------------------------------------------------------------------

func TestParseElapsedTime_MinutesSeconds(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantSec int64
	}{
		{name: "zero", input: "00:00", wantSec: 0},
		{name: "one minute", input: "01:00", wantSec: 60},
		{name: "57 min 42 sec", input: "57:42", wantSec: 57*60 + 42},
		{name: "single digits", input: "5:3", wantSec: 5*60 + 3},
		{name: "99 min 59 sec", input: "99:59", wantSec: 99*60 + 59},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseElapsedTime(tt.input)
			if got != tt.wantSec {
				t.Errorf("parseElapsedTime(%q) = %d, want %d", tt.input, got, tt.wantSec)
			}
		})
	}
}

func TestParseElapsedTime_HoursMinutesSeconds(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantSec int64
	}{
		{name: "one hour", input: "01:00:00", wantSec: 3600},
		{name: "22h 57m 42s", input: "22:57:42", wantSec: 22*3600 + 57*60 + 42},
		{name: "leading zeros", input: "02:05:09", wantSec: 2*3600 + 5*60 + 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseElapsedTime(tt.input)
			if got != tt.wantSec {
				t.Errorf("parseElapsedTime(%q) = %d, want %d", tt.input, got, tt.wantSec)
			}
		})
	}
}

func TestParseElapsedTime_DaysHoursMinutesSeconds(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantSec int64
	}{
		{name: "one day", input: "01-00:00:00", wantSec: 86400},
		{name: "1d 22h 57m 42s", input: "01-22:57:42", wantSec: 86400 + 22*3600 + 57*60 + 42},
		{name: "30 days", input: "30-00:00:00", wantSec: 30 * 86400},
		{name: "365 days plus time", input: "365-12:30:45", wantSec: 365*86400 + 12*3600 + 30*60 + 45},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseElapsedTime(tt.input)
			if got != tt.wantSec {
				t.Errorf("parseElapsedTime(%q) = %d, want %d", tt.input, got, tt.wantSec)
			}
		})
	}
}

func TestParseElapsedTime_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantSec int64
	}{
		{name: "empty string", input: "", wantSec: 0},
		{name: "only separators", input: ":", wantSec: 0},
		{name: "non-numeric values", input: "abc:def", wantSec: 0},
		{name: "days with non-numeric rest", input: "2-abc:def:ghi", wantSec: 2 * 86400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseElapsedTime(tt.input)
			if got != tt.wantSec {
				t.Errorf("parseElapsedTime(%q) = %d, want %d", tt.input, got, tt.wantSec)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseLsofOutput
// ---------------------------------------------------------------------------

func TestParseLsofOutput_SingleListener(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)`

	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(listeners))
	}

	l := listeners[0]
	if l.Port != 80 {
		t.Errorf("port: got %d, want 80", l.Port)
	}
	if l.Address != "0.0.0.0" {
		t.Errorf("address: got %q, want %q", l.Address, "0.0.0.0")
	}
	if l.Protocol != "tcp" {
		t.Errorf("protocol: got %q, want %q", l.Protocol, "tcp")
	}
	if l.PID != 5678 {
		t.Errorf("PID: got %d, want 5678", l.PID)
	}
	if l.Process == nil {
		t.Fatal("Process should not be nil")
	}
	if l.Process.Name != "nginx" {
		t.Errorf("process name: got %q, want %q", l.Process.Name, "nginx")
	}
	if l.Process.User != "root" {
		t.Errorf("process user: got %q, want %q", l.Process.User, "root")
	}
}

func TestParseLsofOutput_MultipleListeners(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
node     1234   dev     8u  IPv6   0x789       0t0  TCP [::1]:3000 (LISTEN)
postgres 9999   pg     10u  IPv4   0xabc       0t0  TCP 127.0.0.1:5432 (LISTEN)`

	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 3 {
		t.Fatalf("expected 3 listeners, got %d", len(listeners))
	}

	// Verify each listener by port
	ports := map[int]bool{}
	for _, l := range listeners {
		ports[l.Port] = true
	}
	for _, want := range []int{80, 3000, 5432} {
		if !ports[want] {
			t.Errorf("missing listener on port %d", want)
		}
	}
}

func TestParseLsofOutput_DeduplicatesListenersByPort(t *testing.T) {
	// Same port, different file descriptors (e.g., IPv4 + IPv6)
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
nginx    5678   root    7u  IPv6   0x456       0t0  TCP [::]:80 (LISTEN)`

	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 1 {
		t.Fatalf("expected 1 deduplicated listener, got %d", len(listeners))
	}
	if listeners[0].Port != 80 {
		t.Errorf("port: got %d, want 80", listeners[0].Port)
	}
}

func TestParseLsofOutput_CountsEstablishedConnections(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
nginx    5678   root    7u  IPv4   0x456       0t0  TCP 10.0.0.1:80->192.168.1.1:54321 (ESTABLISHED)
nginx    5678   root    8u  IPv4   0x789       0t0  TCP 10.0.0.1:80->192.168.1.2:54322 (ESTABLISHED)
nginx    5678   root    9u  IPv4   0xabc       0t0  TCP 10.0.0.1:80->192.168.1.3:54323 (ESTABLISHED)`

	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(listeners))
	}
	if listeners[0].ConnectionCount != 3 {
		t.Errorf("connectionCount: got %d, want 3", listeners[0].ConnectionCount)
	}
}

func TestParseLsofOutput_SkipsNonListenNonEstablished(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
nginx    5678   root    7u  IPv4   0x456       0t0  TCP 10.0.0.1:80->192.168.1.1:54321 (CLOSE_WAIT)
nginx    5678   root    8u  IPv4   0x789       0t0  TCP 10.0.0.1:80->192.168.1.2:54322 (TIME_WAIT)`

	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(listeners))
	}
	// CLOSE_WAIT and TIME_WAIT are not ESTABLISHED, so they should not be counted
	if listeners[0].ConnectionCount != 0 {
		t.Errorf("connectionCount: got %d, want 0", listeners[0].ConnectionCount)
	}
}

func TestParseLsofOutput_EmptyOutput(t *testing.T) {
	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 0 {
		t.Errorf("expected 0 listeners for empty output, got %d", len(listeners))
	}
}

func TestParseLsofOutput_HeaderOnly(t *testing.T) {
	output := "COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME\n"
	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 0 {
		t.Errorf("expected 0 listeners for header-only output, got %d", len(listeners))
	}
}

func TestParseLsofOutput_SkipsMalformedLines(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
this is a short line
another short one`

	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(listeners))
	}
}

func TestParseLsofOutput_IPv6Listener(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
node     1234   dev     8u  IPv6   0x789       0t0  TCP [::1]:3000 (LISTEN)`

	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(listeners))
	}
	if listeners[0].Address != "::1" {
		t.Errorf("address: got %q, want %q", listeners[0].Address, "::1")
	}
	if listeners[0].Port != 3000 {
		t.Errorf("port: got %d, want 3000", listeners[0].Port)
	}
}

func TestParseLsofOutput_UDPEntry(t *testing.T) {
	// UDP entries don't have a state like "(LISTEN)" — they should be skipped
	// since the code requires state == "LISTEN" for entries.
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
mDNSResp  123   root    6u  IPv4   0x123       0t0  UDP *:5353`

	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 0 {
		t.Errorf("expected 0 listeners for UDP without LISTEN state, got %d", len(listeners))
	}
}

// ---------------------------------------------------------------------------
// parsePortDetail
// ---------------------------------------------------------------------------

func TestParsePortDetail_ListenerOnly(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)`

	s := NewDarwinScanner(DefaultOptions())
	listener, connections := s.parsePortDetail(output, 80)
	if listener == nil {
		t.Fatal("expected a listener, got nil")
	}
	if listener.Port != 80 {
		t.Errorf("port: got %d, want 80", listener.Port)
	}
	if listener.Address != "0.0.0.0" {
		t.Errorf("address: got %q, want %q", listener.Address, "0.0.0.0")
	}
	if listener.Protocol != "tcp" {
		t.Errorf("protocol: got %q, want %q", listener.Protocol, "tcp")
	}
	if listener.PID != 5678 {
		t.Errorf("PID: got %d, want 5678", listener.PID)
	}
	if listener.Process == nil {
		t.Fatal("Process should not be nil")
	}
	if listener.Process.Name != "nginx" {
		t.Errorf("name: got %q, want %q", listener.Process.Name, "nginx")
	}
	if listener.Process.User != "root" {
		t.Errorf("user: got %q, want %q", listener.Process.User, "root")
	}
	if len(connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(connections))
	}
}

func TestParsePortDetail_ListenerWithEstablished(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
nginx    5678   root    7u  IPv4   0x456       0t0  TCP 10.0.0.1:80->192.168.1.1:54321 (ESTABLISHED)
nginx    5678   root    8u  IPv4   0x789       0t0  TCP 10.0.0.1:80->192.168.1.2:54322 (ESTABLISHED)`

	s := NewDarwinScanner(DefaultOptions())
	listener, connections := s.parsePortDetail(output, 80)
	if listener == nil {
		t.Fatal("expected a listener, got nil")
	}
	if listener.Port != 80 {
		t.Errorf("port: got %d, want 80", listener.Port)
	}
	if len(connections) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(connections))
	}

	// Check first connection
	c := connections[0]
	if c.LocalAddr != "10.0.0.1" {
		t.Errorf("conn[0] localAddr: got %q, want %q", c.LocalAddr, "10.0.0.1")
	}
	if c.LocalPort != 80 {
		t.Errorf("conn[0] localPort: got %d, want 80", c.LocalPort)
	}
	if c.RemoteAddr != "192.168.1.1" {
		t.Errorf("conn[0] remoteAddr: got %q, want %q", c.RemoteAddr, "192.168.1.1")
	}
	if c.RemotePort != 54321 {
		t.Errorf("conn[0] remotePort: got %d, want 54321", c.RemotePort)
	}
	if c.State != "ESTABLISHED" {
		t.Errorf("conn[0] state: got %q, want %q", c.State, "ESTABLISHED")
	}

	// Check second connection
	c = connections[1]
	if c.RemoteAddr != "192.168.1.2" {
		t.Errorf("conn[1] remoteAddr: got %q, want %q", c.RemoteAddr, "192.168.1.2")
	}
	if c.RemotePort != 54322 {
		t.Errorf("conn[1] remotePort: got %d, want 54322", c.RemotePort)
	}
}

func TestParsePortDetail_WrongPort(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)`

	s := NewDarwinScanner(DefaultOptions())
	listener, connections := s.parsePortDetail(output, 443)
	if listener != nil {
		t.Errorf("expected nil listener for non-matching port, got port %d", listener.Port)
	}
	if len(connections) != 0 {
		t.Errorf("expected 0 connections for non-matching port, got %d", len(connections))
	}
}

func TestParsePortDetail_IPv6Listener(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
node     1234   dev     8u  IPv6   0x789       0t0  TCP [::1]:3000 (LISTEN)`

	s := NewDarwinScanner(DefaultOptions())
	listener, _ := s.parsePortDetail(output, 3000)
	if listener == nil {
		t.Fatal("expected a listener, got nil")
	}
	if listener.Address != "::1" {
		t.Errorf("address: got %q, want %q", listener.Address, "::1")
	}
	if listener.Port != 3000 {
		t.Errorf("port: got %d, want 3000", listener.Port)
	}
	if listener.Process.Name != "node" {
		t.Errorf("name: got %q, want %q", listener.Process.Name, "node")
	}
}

func TestParsePortDetail_EmptyOutput(t *testing.T) {
	s := NewDarwinScanner(DefaultOptions())
	listener, connections := s.parsePortDetail("", 80)
	if listener != nil {
		t.Error("expected nil listener for empty output")
	}
	if len(connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(connections))
	}
}

func TestParsePortDetail_EstablishedOnlyNoListener(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    7u  IPv4   0x456       0t0  TCP 10.0.0.1:80->192.168.1.1:54321 (ESTABLISHED)`

	s := NewDarwinScanner(DefaultOptions())
	listener, connections := s.parsePortDetail(output, 80)
	// No LISTEN entry, so listener should be nil
	if listener != nil {
		t.Error("expected nil listener when no LISTEN entry exists")
	}
	// But connections are still collected
	if len(connections) != 1 {
		t.Errorf("expected 1 connection, got %d", len(connections))
	}
}

func TestParsePortDetail_EstablishedOnDifferentLocalPort(t *testing.T) {
	// ESTABLISHED connection where the local port does not match the target
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
nginx    5678   root    7u  IPv4   0x456       0t0  TCP 10.0.0.1:9999->192.168.1.1:54321 (ESTABLISHED)`

	s := NewDarwinScanner(DefaultOptions())
	listener, connections := s.parsePortDetail(output, 80)
	if listener == nil {
		t.Fatal("expected a listener, got nil")
	}
	// The ESTABLISHED connection has localPort 9999, not 80, so it should not be collected
	if len(connections) != 0 {
		t.Errorf("expected 0 connections for non-matching local port, got %d", len(connections))
	}
}

func TestParsePortDetail_PicksFirstListenerForPort(t *testing.T) {
	// Two LISTEN entries on the same port (e.g., IPv4 and IPv6). Only the first should be returned.
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
nginx    5678   root    7u  IPv6   0x456       0t0  TCP [::]:80 (LISTEN)`

	s := NewDarwinScanner(DefaultOptions())
	listener, _ := s.parsePortDetail(output, 80)
	if listener == nil {
		t.Fatal("expected a listener, got nil")
	}
	// Should be the first (IPv4 wildcard) entry
	if listener.Address != "0.0.0.0" {
		t.Errorf("address: got %q, want %q (first LISTEN entry)", listener.Address, "0.0.0.0")
	}
}

func TestParsePortDetail_IgnoresCloseWait(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
nginx    5678   root    7u  IPv4   0x456       0t0  TCP 10.0.0.1:80->192.168.1.1:54321 (CLOSE_WAIT)`

	s := NewDarwinScanner(DefaultOptions())
	listener, connections := s.parsePortDetail(output, 80)
	if listener == nil {
		t.Fatal("expected a listener, got nil")
	}
	if len(connections) != 0 {
		t.Errorf("CLOSE_WAIT should not be collected, got %d connections", len(connections))
	}
}

func TestParsePortDetail_RealisticMultiService(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
nginx    5678   root    7u  IPv4   0x456       0t0  TCP 10.0.0.1:80->192.168.1.1:54321 (ESTABLISHED)
node     1234   dev     8u  IPv6   0x789       0t0  TCP [::1]:3000 (LISTEN)
postgres 9999   pg     10u  IPv4   0xabc       0t0  TCP 127.0.0.1:5432 (LISTEN)
nginx    5678   root   11u  IPv4   0xdef       0t0  TCP 10.0.0.1:80->192.168.1.5:12345 (ESTABLISHED)`

	s := NewDarwinScanner(DefaultOptions())

	// Query port 80
	listener, connections := s.parsePortDetail(output, 80)
	if listener == nil {
		t.Fatal("expected listener for port 80")
	}
	if listener.Port != 80 {
		t.Errorf("port: got %d, want 80", listener.Port)
	}
	if len(connections) != 2 {
		t.Errorf("expected 2 connections for port 80, got %d", len(connections))
	}

	// Query port 3000
	listener2, conns2 := s.parsePortDetail(output, 3000)
	if listener2 == nil {
		t.Fatal("expected listener for port 3000")
	}
	if listener2.Address != "::1" {
		t.Errorf("address: got %q, want %q", listener2.Address, "::1")
	}
	if len(conns2) != 0 {
		t.Errorf("expected 0 connections for port 3000, got %d", len(conns2))
	}

	// Query port 5432
	listener3, conns3 := s.parsePortDetail(output, 5432)
	if listener3 == nil {
		t.Fatal("expected listener for port 5432")
	}
	if listener3.Process.Name != "postgres" {
		t.Errorf("name: got %q, want %q", listener3.Process.Name, "postgres")
	}
	if len(conns3) != 0 {
		t.Errorf("expected 0 connections for port 5432, got %d", len(conns3))
	}
}

func TestParsePortDetail_MalformedLines(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
short line
nginx    5678   root    6u  IPv4   0x123       0t0  TCP *:80 (LISTEN)
another broken line here but not enough fields`

	s := NewDarwinScanner(DefaultOptions())
	listener, _ := s.parsePortDetail(output, 80)
	if listener == nil {
		t.Fatal("expected a listener despite malformed lines")
	}
	if listener.Port != 80 {
		t.Errorf("port: got %d, want 80", listener.Port)
	}
}

// ---------------------------------------------------------------------------
// decodeLsofEscapes
// ---------------------------------------------------------------------------

func TestDecodeLsofEscapes_SpaceEscape(t *testing.T) {
	got := decodeLsofEscapes(`Code\x20Helper`)
	want := "Code Helper"
	if got != want {
		t.Errorf("decodeLsofEscapes(%q) = %q, want %q", `Code\x20Helper`, got, want)
	}
}

func TestDecodeLsofEscapes_NoEscapes(t *testing.T) {
	got := decodeLsofEscapes("nginx")
	if got != "nginx" {
		t.Errorf("expected %q, got %q", "nginx", got)
	}
}

func TestDecodeLsofEscapes_MultipleEscapes(t *testing.T) {
	got := decodeLsofEscapes(`My\x20Cool\x20App`)
	want := "My Cool App"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeLsofEscapes_TabAndOther(t *testing.T) {
	got := decodeLsofEscapes(`test\x09name`)
	want := "test\tname"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseLsofOutput_EscapedCommandName(t *testing.T) {
	output := `COMMAND   PID   USER   FD   TYPE   DEVICE SIZE/OFF NODE NAME
Code\x20H  8830  tasnim   45u  IPv4  0x789       0t0  TCP *:8830 (LISTEN)`

	s := NewDarwinScanner(DefaultOptions())
	listeners, err := s.parseLsofOutput(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(listeners))
	}
	if listeners[0].Process.Name != "Code H" {
		t.Errorf("expected decoded name %q, got %q", "Code H", listeners[0].Process.Name)
	}
}
