package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/kill"
	"github.com/tasnimzotder/portman/internal/tui"
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

func resolveSignal() (syscall.Signal, string, error) {
	if killForce {
		return syscall.SIGKILL, "KILL", nil
	}
	sig, ok := kill.ParseSignal(killSignal)
	if !ok {
		return 0, "", fmt.Errorf("unknown signal: %s (valid: HUP, INT, TERM, KILL)", killSignal)
	}
	return sig, strings.ToUpper(killSignal), nil
}

func runKill(cmd *cobra.Command, args []string) error {
	port, err := parsePort(args[0])
	if err != nil {
		return err
	}

	sig, sigName, err := resolveSignal()
	if err != nil {
		return err
	}

	s, err := newScanner()
	if err != nil {
		return err
	}

	listener, err := s.GetPort(port)
	if err != nil {
		return err
	}

	if listener == nil {
		return fmt.Errorf("port %d is not in use", port)
	}

	pid := listener.PID
	processName := listener.ProcessName()
	userName := listener.ProcessUser()
	uptime := ""
	if listener.Process != nil && listener.Process.UptimeSeconds > 0 {
		uptime = (time.Duration(listener.Process.UptimeSeconds) * time.Second).String()
	}

	// TUI kill for interactive terminals (unless --yes or --quiet)
	if tui.IsTerminal() && !killYes && !killQuiet {
		m := tui.NewKillModel(s, port, sig, sigName, killForce)
		_, err := tea.NewProgram(m).Run()
		return err
	}

	// Text confirmation
	if !killYes {
		fmt.Printf("Kill process on port %d?\n", port)
		fmt.Printf("  Process: %s\n", processName)
		fmt.Printf("  PID:     %d\n", pid)
		fmt.Printf("  User:    %s\n", userName)
		if uptime != "" {
			fmt.Printf("  Uptime:  %s\n", uptime)
		}
		fmt.Println()

		fmt.Print("Confirm [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		answer := strings.TrimSpace(strings.ToLower(input))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if !killQuiet {
		fmt.Printf("Sent SIG%s to PID %d\n", sigName, pid)
	}

	if err := kill.Kill(pid, sig); err != nil {
		if errors.Is(err, kill.ErrPermissionDenied) {
			return fmt.Errorf("permission denied — try running with sudo")
		}
		return err
	}

	if kill.WaitForExit(pid, 3*time.Second) {
		if !killQuiet {
			fmt.Println("Process terminated.")
		}
		return nil
	}

	if killForce && sig != syscall.SIGKILL {
		if !killQuiet {
			fmt.Println("Process didn't exit, sending SIGKILL...")
		}
		if err := kill.Kill(pid, syscall.SIGKILL); err != nil {
			return fmt.Errorf("SIGKILL failed: %w", err)
		}
		if kill.WaitForExit(pid, 2*time.Second) {
			if !killQuiet {
				fmt.Println("Process killed.")
			}
			return nil
		}
	}

	return fmt.Errorf("process %d didn't terminate", pid)
}
