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
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Sending signal 0 checks if process exists without actually signaling
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// KillWithTimeout sends SIGTERM, waits for exit, then sends SIGKILL if needed.
func KillWithTimeout(pid int, timeout time.Duration) error {
	// First try SIGTERM
	if err := Kill(pid, syscall.SIGTERM); err != nil {
		return err
	}

	// Wait for graceful exit
	if WaitForExit(pid, timeout) {
		return nil
	}

	// Force kill with SIGKILL
	if err := Kill(pid, syscall.SIGKILL); err != nil {
		return err
	}

	// Wait a bit more for SIGKILL to take effect
	if WaitForExit(pid, 2*time.Second) {
		return nil
	}

	return ErrProcessRunning
}
