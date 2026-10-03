package kill

import (
	"os"
	"syscall"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// ParseSignal
// ---------------------------------------------------------------------------

func TestParseSignal_TERM(t *testing.T) {
	sig, ok := ParseSignal("TERM")
	if !ok {
		t.Fatal("expected ok=true for TERM")
	}
	if sig != syscall.SIGTERM {
		t.Fatalf("expected SIGTERM, got %v", sig)
	}
}

func TestParseSignal_SIGTERM(t *testing.T) {
	sig, ok := ParseSignal("SIGTERM")
	if !ok {
		t.Fatal("expected ok=true for SIGTERM")
	}
	if sig != syscall.SIGTERM {
		t.Fatalf("expected SIGTERM, got %v", sig)
	}
}

func TestParseSignal_HUP(t *testing.T) {
	sig, ok := ParseSignal("HUP")
	if !ok {
		t.Fatal("expected ok=true for HUP")
	}
	if sig != syscall.SIGHUP {
		t.Fatalf("expected SIGHUP, got %v", sig)
	}
}

func TestParseSignal_KILL(t *testing.T) {
	sig, ok := ParseSignal("KILL")
	if !ok {
		t.Fatal("expected ok=true for KILL")
	}
	if sig != syscall.SIGKILL {
		t.Fatalf("expected SIGKILL, got %v", sig)
	}
}

func TestParseSignal_Invalid(t *testing.T) {
	sig, ok := ParseSignal("invalid")
	if ok {
		t.Fatal("expected ok=false for invalid signal")
	}
	if sig != 0 {
		t.Fatalf("expected signal 0 for invalid input, got %v", sig)
	}
}

func TestParseSignal_CaseInsensitive(t *testing.T) {
	// ParseSignal uppercases the input before lookup, so lowercase should work.
	sig, ok := ParseSignal("term")
	if !ok {
		t.Fatal("expected ok=true for lowercase term")
	}
	if sig != syscall.SIGTERM {
		t.Fatalf("expected SIGTERM, got %v", sig)
	}
}

func TestParseSignal_MixedCase(t *testing.T) {
	sig, ok := ParseSignal("SigKill")
	if !ok {
		t.Fatal("expected ok=true for mixed-case SigKill")
	}
	if sig != syscall.SIGKILL {
		t.Fatalf("expected SIGKILL, got %v", sig)
	}
}

// ---------------------------------------------------------------------------
// IsRunning
// ---------------------------------------------------------------------------

func TestIsRunning_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	if !IsRunning(pid) {
		t.Fatalf("expected current process (pid %d) to be running", pid)
	}
}

func TestIsRunning_NonExistentPID(t *testing.T) {
	// PID 99999999 is almost certainly not in use on any system.
	if IsRunning(99999999) {
		t.Fatal("expected PID 99999999 to NOT be running")
	}
}

// ---------------------------------------------------------------------------
// Kill
// ---------------------------------------------------------------------------

func TestKill_NonExistentProcess(t *testing.T) {
	// On Unix, os.FindProcess always succeeds. Signaling a non-existent PID
	// returns os.ErrProcessDone, which Kill treats as success (the process
	// is already gone). Verify Kill does not return an error in this case.
	err := Kill(99999999, syscall.SIGTERM)
	if err != nil {
		t.Fatalf("expected nil error for non-existent PID (process already done), got %v", err)
	}
}

func TestKill_SendSignalZeroToSelf(t *testing.T) {
	// Sending signal 0 to the current process should succeed without error
	// (it just checks process existence).
	err := Kill(os.Getpid(), syscall.Signal(0))
	if err != nil {
		t.Fatalf("expected nil error when sending signal 0 to self, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// WaitForExit
// ---------------------------------------------------------------------------

func TestWaitForExit_NonExistentPID(t *testing.T) {
	// A PID that is not running should return true (exited) immediately.
	exited := WaitForExit(99999999, 500*time.Millisecond)
	if !exited {
		t.Fatal("expected WaitForExit to return true for non-existent PID")
	}
}

func TestWaitForExit_CurrentProcess_Timeout(t *testing.T) {
	// The current process is alive, so WaitForExit should time out and
	// return false.
	pid := os.Getpid()
	exited := WaitForExit(pid, 300*time.Millisecond)
	if exited {
		t.Fatalf("expected WaitForExit to return false for running process (pid %d)", pid)
	}
}
