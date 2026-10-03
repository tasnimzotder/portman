package kill

import (
	"errors"
	"os"
	"strings"
	"syscall"
	"time"
)

var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrProcessNotFound  = errors.New("process not found")
	ErrProcessRunning   = errors.New("process still running")
)

// SignalMap maps signal names to syscall signals.
var SignalMap = map[string]syscall.Signal{
	"HUP":     syscall.SIGHUP,
	"SIGHUP":  syscall.SIGHUP,
	"INT":     syscall.SIGINT,
	"SIGINT":  syscall.SIGINT,
	"TERM":    syscall.SIGTERM,
	"SIGTERM": syscall.SIGTERM,
	"KILL":    syscall.SIGKILL,
	"SIGKILL": syscall.SIGKILL,
}

// ParseSignal converts a signal name to syscall.Signal.
func ParseSignal(name string) (syscall.Signal, bool) {
	sig, ok := SignalMap[strings.ToUpper(name)]
	return sig, ok
}

// Kill sends a signal to the process with the given PID.
func Kill(pid int, signal syscall.Signal) error {
	if pid <= 0 {
		return ErrProcessNotFound
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return ErrProcessNotFound
	}

	err = process.Signal(signal)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return ErrPermissionDenied
		}
		if errors.Is(err, os.ErrProcessDone) {
			return nil // Process already exited
		}
		// ESRCH = no such process — treat as already gone
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}

	return nil
}

// WaitForExit waits for a process to exit within the given timeout.
// Returns true if the process exited, false if timeout reached.
func WaitForExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	checkInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		if !IsRunning(pid) {
			return true
		}
		time.Sleep(checkInterval)
	}

	return !IsRunning(pid)
}

// IsRunning checks if a process is still running.
func IsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Sending signal 0 checks if process exists without actually signaling
	err = process.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

// KillWithTimeout sends SIGTERM, waits for exit, then sends SIGKILL if needed.
func KillWithTimeout(pid int, timeout time.Duration) error {
	return SendAndWait(pid, syscall.SIGTERM, timeout, true)
}

// SendAndWait applies the same signal semantics in text and interactive modes.
// HUP requests a reload and need not terminate the process.
func SendAndWait(pid int, signal syscall.Signal, timeout time.Duration, force bool) error {
	if pid <= 0 {
		return ErrProcessNotFound
	}
	if timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if err := Kill(pid, signal); err != nil {
		return err
	}
	if signal == syscall.SIGHUP {
		return nil
	}
	if WaitForExit(pid, timeout) {
		return nil
	}
	if force && signal != syscall.SIGKILL {
		if err := Kill(pid, syscall.SIGKILL); err != nil {
			return err
		}
		if WaitForExit(pid, 2*time.Second) {
			return nil
		}
	}
	return ErrProcessRunning
}
