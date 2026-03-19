package wait

import (
	"errors"
	"testing"
	"time"

	"github.com/tasnimzotder/portman/internal/model"
)

// ---------------------------------------------------------------------------
// mockScanner implements scanner.Scanner for testing.
// ---------------------------------------------------------------------------

type mockScanner struct {
	listeners []model.Listener
	port      *model.Listener
	err       error
	callCount int
}

func (m *mockScanner) ListListeners() ([]model.Listener, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.listeners, nil
}

func (m *mockScanner) GetPort(port int) (*model.Listener, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.port, nil
}

func (m *mockScanner) FindByPattern(pattern string) ([]model.Listener, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.listeners, nil
}

func (m *mockScanner) ListByPID(pid int) ([]model.Listener, error) {
	return m.listeners, m.err
}

// changingScanner returns an error for the first N calls, then succeeds.
type changingScanner struct {
	failFor   int
	port      *model.Listener
	callCount int
}

func (c *changingScanner) ListListeners() ([]model.Listener, error) {
	return nil, nil
}

func (c *changingScanner) GetPort(port int) (*model.Listener, error) {
	c.callCount++
	if c.callCount <= c.failFor {
		return nil, errors.New("temporary error")
	}
	return c.port, nil
}

func (c *changingScanner) FindByPattern(pattern string) ([]model.Listener, error) {
	return nil, nil
}

func (c *changingScanner) ListByPID(pid int) ([]model.Listener, error) {
	return nil, nil
}

// delayedScanner returns nil (port free) for the first N calls, then returns
// a listener (port in use), or vice-versa.
type delayedScanner struct {
	nilFor    int // number of calls that return nil (port free)
	listener  *model.Listener
	callCount int
}

func (d *delayedScanner) ListListeners() ([]model.Listener, error) {
	return nil, nil
}

func (d *delayedScanner) GetPort(port int) (*model.Listener, error) {
	d.callCount++
	if d.callCount <= d.nilFor {
		return nil, nil
	}
	return d.listener, nil
}

func (d *delayedScanner) FindByPattern(pattern string) ([]model.Listener, error) {
	return nil, nil
}

func (d *delayedScanner) ListByPID(pid int) ([]model.Listener, error) {
	return nil, nil
}

// invertDelayedScanner returns a listener (port in use) for the first N
// calls, then returns nil (port free).
type invertDelayedScanner struct {
	inUseFor  int
	listener  *model.Listener
	callCount int
}

func (s *invertDelayedScanner) ListListeners() ([]model.Listener, error) {
	return nil, nil
}

func (s *invertDelayedScanner) GetPort(port int) (*model.Listener, error) {
	s.callCount++
	if s.callCount <= s.inUseFor {
		return s.listener, nil
	}
	return nil, nil
}

func (s *invertDelayedScanner) FindByPattern(pattern string) ([]model.Listener, error) {
	return nil, nil
}

func (s *invertDelayedScanner) ListByPID(pid int) ([]model.Listener, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Wait tests
// ---------------------------------------------------------------------------

func TestWait_PortAvailableImmediately(t *testing.T) {
	m := &mockScanner{
		port: &model.Listener{
			Port:     8080,
			Protocol: "tcp",
			Process: &model.Process{
				Name:    "myapp",
				Command: "myapp serve",
			},
		},
	}

	result := Wait(m, 8080, 2*time.Second, 50*time.Millisecond, false)

	if !result.Success {
		t.Fatal("expected success when port is immediately available")
	}
	if result.ProcessName != "myapp serve" {
		t.Fatalf("expected process name 'myapp serve', got %q", result.ProcessName)
	}
	if m.callCount != 1 {
		t.Fatalf("expected 1 scanner call, got %d", m.callCount)
	}
}

func TestWait_PortAvailable_ProcessNameOnly(t *testing.T) {
	m := &mockScanner{
		port: &model.Listener{
			Port:     3000,
			Protocol: "tcp",
			Process: &model.Process{
				Name: "node",
				// Command is empty, so ProcessName should fall back to Name.
			},
		},
	}

	result := Wait(m, 3000, 2*time.Second, 50*time.Millisecond, false)

	if !result.Success {
		t.Fatal("expected success")
	}
	if result.ProcessName != "node" {
		t.Fatalf("expected process name 'node', got %q", result.ProcessName)
	}
}

func TestWait_PortAvailable_NilProcess(t *testing.T) {
	m := &mockScanner{
		port: &model.Listener{
			Port:     3000,
			Protocol: "tcp",
			Process:  nil,
		},
	}

	result := Wait(m, 3000, 2*time.Second, 50*time.Millisecond, false)

	if !result.Success {
		t.Fatal("expected success")
	}
	if result.ProcessName != "" {
		t.Fatalf("expected empty process name, got %q", result.ProcessName)
	}
}

func TestWait_Timeout(t *testing.T) {
	m := &mockScanner{
		port: nil, // port never becomes available
	}

	result := Wait(m, 8080, 300*time.Millisecond, 50*time.Millisecond, false)

	if result.Success {
		t.Fatal("expected timeout (success=false) when port never becomes available")
	}
	if result.Elapsed < 250*time.Millisecond {
		t.Fatalf("expected elapsed >= 250ms, got %v", result.Elapsed)
	}
}

func TestWait_InvertMode_PortBecomesFree(t *testing.T) {
	listener := &model.Listener{Port: 8080, Protocol: "tcp"}
	s := &invertDelayedScanner{
		inUseFor: 2, // port in use for first 2 calls, then free
		listener: listener,
	}

	result := Wait(s, 8080, 2*time.Second, 50*time.Millisecond, true)

	if !result.Success {
		t.Fatal("expected success when port becomes free in invert mode")
	}
	if s.callCount < 3 {
		t.Fatalf("expected at least 3 scanner calls, got %d", s.callCount)
	}
}

func TestWait_InvertMode_Timeout(t *testing.T) {
	// Port is always in use -- invert mode should time out.
	m := &mockScanner{
		port: &model.Listener{Port: 8080, Protocol: "tcp"},
	}

	result := Wait(m, 8080, 300*time.Millisecond, 50*time.Millisecond, true)

	if result.Success {
		t.Fatal("expected timeout in invert mode when port stays in use")
	}
}

func TestWait_ErrorDuringCheck_ContinuesPolling(t *testing.T) {
	listener := &model.Listener{
		Port:     8080,
		Protocol: "tcp",
		Process:  &model.Process{Name: "myapp"},
	}
	s := &changingScanner{
		failFor: 3, // first 3 calls error, then succeed
		port:    listener,
	}

	result := Wait(s, 8080, 2*time.Second, 50*time.Millisecond, false)

	if !result.Success {
		t.Fatal("expected success after transient errors clear")
	}
	if s.callCount < 4 {
		t.Fatalf("expected at least 4 scanner calls, got %d", s.callCount)
	}
}

func TestWait_PortBecomesAvailableAfterDelay(t *testing.T) {
	listener := &model.Listener{
		Port:     9090,
		Protocol: "tcp",
		Process:  &model.Process{Name: "server", Command: "server --port 9090"},
	}
	s := &delayedScanner{
		nilFor:   3, // free for first 3 calls, then in use
		listener: listener,
	}

	result := Wait(s, 9090, 2*time.Second, 50*time.Millisecond, false)

	if !result.Success {
		t.Fatal("expected success after port becomes available")
	}
	if result.ProcessName != "server --port 9090" {
		t.Fatalf("expected 'server --port 9090', got %q", result.ProcessName)
	}
}

// ---------------------------------------------------------------------------
// IsPortOpen tests
// ---------------------------------------------------------------------------

func TestIsPortOpen_InUse(t *testing.T) {
	m := &mockScanner{
		port: &model.Listener{Port: 8080, Protocol: "tcp"},
	}
	if !IsPortOpen(m, 8080) {
		t.Fatal("expected IsPortOpen=true when scanner returns a listener")
	}
}

func TestIsPortOpen_NotInUse(t *testing.T) {
	m := &mockScanner{
		port: nil,
	}
	if IsPortOpen(m, 8080) {
		t.Fatal("expected IsPortOpen=false when scanner returns nil")
	}
}

func TestIsPortOpen_Error(t *testing.T) {
	m := &mockScanner{
		err: errors.New("scan failed"),
	}
	if IsPortOpen(m, 8080) {
		t.Fatal("expected IsPortOpen=false when scanner returns an error")
	}
}
