package wait

import (
	"time"

	"github.com/tasnimzotder/portman/internal/scanner"
)

// Result contains the outcome of a wait operation.
type Result struct {
	Success     bool
	Elapsed     time.Duration
	ProcessName string
}

// Wait polls for a port to become available (or free if invert is true).
// Returns the result with success status, elapsed time, and process name if found.
func Wait(s scanner.Scanner, port int, timeout, interval time.Duration, invert bool) Result {
	start := time.Now()
	deadline := start.Add(timeout)

	for time.Now().Before(deadline) {
		listener, err := s.GetPort(port)
		if err != nil {
			// Error during check, continue polling
			time.Sleep(interval)
			continue
		}

		portInUse := listener != nil

		if invert {
			// Wait for port to be FREE
			if !portInUse {
				return Result{
					Success: true,
					Elapsed: time.Since(start),
				}
			}
		} else {
			// Wait for port to be IN USE (available/listening)
			if portInUse {
				processName := ""
				if listener.Process != nil {
					processName = listener.Process.Name
					if listener.Process.Command != "" {
						processName = listener.Process.Command
					}
				}
				return Result{
					Success:     true,
					Elapsed:     time.Since(start),
					ProcessName: processName,
				}
			}
		}

		time.Sleep(interval)
	}

	// Timeout reached
	return Result{
		Success: false,
		Elapsed: time.Since(start),
	}
}

// IsPortOpen checks if a port is currently in use.
func IsPortOpen(s scanner.Scanner, port int) bool {
	listener, err := s.GetPort(port)
	if err != nil {
		return false
	}
	return listener != nil
}
