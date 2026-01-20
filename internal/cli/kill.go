package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/kill"
	"github.com/tasnimzotder/portman/internal/scanner"
	"github.com/tasnimzotder/portman/internal/ui"
)

var (
	killForce   bool
	killYes     bool
	killSignal  string
	killTimeout time.Duration
	killQuiet   bool
)

func init() {
	killCmd.Flags().BoolVarP(&killForce, "force", "f", false, "Use SIGKILL instead of SIGTERM")
	killCmd.Flags().BoolVarP(&killYes, "yes", "y", false, "Skip confirmation")
	killCmd.Flags().StringVarP(&killSignal, "signal", "s", "TERM", "Signal to send (HUP, INT, TERM, KILL)")
	killCmd.Flags().DurationVar(&killTimeout, "timeout", 5*time.Second, "Wait time before SIGKILL (with --force)")
	killCmd.Flags().BoolVarP(&killQuiet, "quiet", "q", false, "No output on success")
}

var killCmd = &cobra.Command{
	Use:   "kill <port>",
	Short: "Kill the process using a port",
	Args:  cobra.ExactArgs(1),
	RunE:  runKill,
}

func runKill(cmd *cobra.Command, args []string) error {
	port, err := parsePort(args[0])
	if err != nil {
		return err
	}

	// Find process using the port
	opts := scanner.DefaultOptions()
	s, err := scanner.New(opts)
	if err != nil {
		return err
	}

	listener, err := s.GetPort(port)
	if err != nil {
		return err
	}

	if listener == nil {
		fmt.Printf("Port %d is not in use.\n", port)
		os.Exit(1)
	}

	pid := listener.PID
	processName := "unknown"
	userName := "unknown"
	uptime := ""

	if listener.Process != nil {
		processName = listener.Process.Command
		if processName == "" {
			processName = listener.Process.Name
		}
		userName = listener.Process.User
		if listener.Process.UptimeSeconds > 0 {
			uptime = (time.Duration(listener.Process.UptimeSeconds) * time.Second).String()
		}
	}

	// Show confirmation unless --yes
	if !killYes {
		fmt.Printf("Kill process on port %d?\n", port)
		fmt.Printf("  Process: %s\n", processName)
		fmt.Printf("  PID:     %d\n", pid)
		fmt.Printf("  User:    %s\n", userName)
		if uptime != "" {
			fmt.Printf("  Uptime:  %s\n", uptime)
		}
		fmt.Println()

		if !ui.Confirm("Confirm") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Determine signal to use
	var sig syscall.Signal
	if killForce {
		sig = syscall.SIGKILL
	} else {
		var ok bool
		sig, ok = kill.ParseSignal(killSignal)
		if !ok {
			return fmt.Errorf("unknown signal: %s", killSignal)
		}
	}

	// Send the signal
	if !killQuiet {
		signalName := strings.ToUpper(killSignal)
		if killForce {
			signalName = "KILL"
		}
		fmt.Printf("Sent SIG%s to PID %d\n", signalName, pid)
	}

	err = kill.Kill(pid, sig)
	if err != nil {
		if errors.Is(err, kill.ErrPermissionDenied) {
			fmt.Println("Permission denied. Try running with sudo.")
			os.Exit(2)
		}
		return err
	}

	// Wait for process to exit
	if kill.WaitForExit(pid, 3*time.Second) {
		if !killQuiet {
			fmt.Println("Process terminated.")
		}
		return nil
	}

	// If --force, try SIGKILL after timeout
	if killForce && sig != syscall.SIGKILL {
		if !killQuiet {
			fmt.Printf("Process didn't exit, sending SIGKILL...\n")
		}
		kill.Kill(pid, syscall.SIGKILL)
		if kill.WaitForExit(pid, 2*time.Second) {
			if !killQuiet {
				fmt.Println("Process killed.")
			}
			return nil
		}
	}

	fmt.Println("Process didn't terminate.")
	os.Exit(3)
	return nil
}
