//go:build darwin

package scanner

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mockLsof(t *testing.T, output string, status string) {
	t.Helper()
	dir := t.TempDir()
	fixture := filepath.Join(dir, "output")
	if err := os.WriteFile(fixture, []byte(output), 0600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n/bin/cat '" + fixture + "'\n" + status + "\n"
	if err := os.WriteFile(filepath.Join(dir, "lsof"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
}

func TestFieldOutputPreservesOwnersAndProtocols(t *testing.T) {
	fixture := "p111\x00cname with spaces,commas\x00u501\x00Luser\x00\nf3\x00tIPv4\x00PTCP\x00n127.0.0.1:34567\x00TST=LISTEN\x00\nf4\x00tIPv4\x00PTCP\x00n127.0.0.1:34567\x00TST=LISTEN\x00\nf5\x00tIPv4\x00PTCP\x00n127.0.0.1:34567->127.0.0.1:12345\x00TST=ESTABLISHED\x00\nf6\x00tIPv4\x00PUDP\x00n127.0.0.1:34567\x00\np222\x00cother\x00Luser\x00\nf7\x00tIPv4\x00PTCP\x00n127.0.0.2:34567\x00TST=LISTEN\x00\n"
	mockLsof(t, fixture, "exit 0")
	opts := DefaultOptions()
	opts.FetchStats = true
	s := NewDarwinScanner(opts)
	listeners, err := s.ListListeners()
	if err != nil || len(listeners) != 3 {
		t.Fatalf("listeners=%+v err=%v", listeners, err)
	}
	if listeners[0].ProcessName() != "name with spaces,commas" || listeners[0].ConnectionCount != 1 || listeners[1].ConnectionCount != 0 {
		t.Fatalf("incorrect owner/connection attribution: %+v", listeners)
	}
	if _, err = s.GetPort(34567); err == nil {
		t.Fatal("ambiguous detail should fail")
	}
	if l, err := s.GetPortStatusContext(context.Background(), 34567); err != nil || l == nil {
		t.Fatalf("occupied port status: %v %v", l, err)
	}
}

func TestScanFailuresAreNotFreePorts(t *testing.T) {
	for _, tc := range []struct {
		name, output, status string
		wantErr              bool
	}{
		{"no matches", "", "exit 1", false},
		{"failed", "", "exit 2", true},
		{"diagnostic", "", "echo unavailable >&2\nexit 1", true},
		{"partial", "p111\x00cserver\x00\nf3\x00PTCP\x00n*:34567\x00TST=LISTEN\x00\n", "exit 1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockLsof(t, tc.output, tc.status)
			s := NewDarwinScanner(DefaultOptions())
			_, err := s.ListListeners()
			if (err != nil) != tc.wantErr {
				t.Fatalf("list err=%v", err)
			}
			_, err = s.GetPort(34567)
			if (err != nil) != tc.wantErr {
				t.Fatalf("detail err=%v", err)
			}
		})
	}
}

func TestLiveTCPAndUDPBindings(t *testing.T) {
	tcp, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer tcp.Close()
	udp, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()
	for _, tc := range []struct {
		protocol string
		port     int
	}{{"tcp", tcp.Addr().(*net.TCPAddr).Port}, {"udp", udp.LocalAddr().(*net.UDPAddr).Port}} {
		t.Run(tc.protocol, func(t *testing.T) {
			opts := DefaultOptions()
			opts.IncludeTCP = tc.protocol == "tcp"
			opts.IncludeUDP = tc.protocol == "udp"
			s := NewDarwinScanner(opts)
			all, err := s.ListListeners()
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, l := range all {
				if l.Protocol != tc.protocol {
					t.Fatalf("protocol filter leaked %+v", l)
				}
				if l.Port == tc.port && l.PID == os.Getpid() {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing live %s socket %d", tc.protocol, tc.port)
			}
			detail, err := s.GetPort(tc.port)
			if err != nil || detail == nil || detail.Protocol != tc.protocol {
				t.Fatalf("detail=%+v err=%v", detail, err)
			}
			if detail.Stats != nil {
				t.Fatal("lightweight scan fetched stats")
			}
		})
	}
}

func TestScanDeadlineCancelsLsof(t *testing.T) {
	mockLsof(t, "", "exec /bin/sleep 5")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := NewDarwinScanner(DefaultOptions()).GetPortContext(ctx, 34567)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatalf("deadline not respected: %v elapsed=%s", err, time.Since(start))
	}
}

func TestConnectedUDPStillOccupiesLocalPort(t *testing.T) {
	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 34567})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	port := conn.LocalAddr().(*net.UDPAddr).Port
	opts := DefaultOptions()
	opts.IncludeTCP = false
	s := NewDarwinScanner(opts)
	l, err := s.GetPortStatusContext(context.Background(), port)
	if err != nil || l == nil || l.PID != os.Getpid() || l.Port != port {
		t.Fatalf("connected UDP incorrectly free: %+v %v", l, err)
	}
}
