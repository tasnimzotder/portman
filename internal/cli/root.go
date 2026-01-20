package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/kill"
	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/scanner"
	"github.com/tasnimzotder/portman/internal/ui"
	"github.com/tasnimzotder/portman/internal/wait"
)

var (
	// Global flags
	jsonOutput bool
	noHeader   bool
	tcpOnly    bool
	udpOnly    bool
	sortBy     string
)

var RootCmd = &cobra.Command{
	Use:   "portman [port]",
	Short: "See what's using your ports",
	Long:  `portman is a cross-platform CLI tool for inspecting and managing network port usage.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRoot,
}

func init() {
	// Global flags
	RootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	RootCmd.PersistentFlags().BoolVar(&noHeader, "no-header", false, "Omit header row")
	RootCmd.PersistentFlags().BoolVarP(&tcpOnly, "tcp", "t", false, "Show only TCP")
	RootCmd.PersistentFlags().BoolVarP(&udpOnly, "udp", "u", false, "Show only UDP")
	RootCmd.PersistentFlags().StringVar(&sortBy, "sort", "port", "Sort by: port, pid, user, conns")

	// Add subcommands
	RootCmd.AddCommand(findCmd)
	RootCmd.AddCommand(killCmd)
	RootCmd.AddCommand(waitCmd)
	RootCmd.AddCommand(portCmd)
	RootCmd.AddCommand(pidCmd)
}

func runRoot(cmd *cobra.Command, args []string) error {
	opts := scanner.DefaultOptions()
	if tcpOnly {
		opts.IncludeUDP = false
	}
	if udpOnly {
		opts.IncludeTCP = false
	}

	s := scanner.New(opts)

	// If port specified, show details
	if len(args) == 1 {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port: %s", args[0])
		}
		return showPortDetail(s, port)
	}

	// Otherwise list all
	return listAllPorts(s)
}

func listAllPorts(s scanner.Scanner) error {
	listeners, err := s.ListListeners()
	if err != nil {
		return err
	}

	// Sort listeners
	output.SortListeners(listeners, sortBy)

	if jsonOutput {
		formatter := output.NewJSONFormatter(true)
		out, err := formatter.Format(listeners)
		if err != nil {
			return err
		}
		fmt.Println(out)
	} else {
		formatter := output.NewTableFormatter()
		formatter.NoHeader = noHeader
		fmt.Print(formatter.Format(listeners))
	}

	return nil
}

func showPortDetail(s scanner.Scanner, port int) error {
	listener, err := s.GetPort(port)
	if err != nil {
		return err
	}

	if listener == nil {
		if jsonOutput {
			fmt.Println("{}")
		} else {
			fmt.Printf("Port %d is not in use.\n", port)
		}
		return nil
	}

	if jsonOutput {
		formatter := output.NewJSONFormatter(true)
		out, err := formatter.FormatSingle(listener)
		if err != nil {
			return err
		}
		fmt.Println(out)
	} else {
		formatter := output.NewTableFormatter()
		fmt.Print(formatter.FormatDetail(listener))
	}

	return nil
}

var findCmd = &cobra.Command{
	Use:   "find <pattern>",
	Short: "Find ports by process name, command, or user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := scanner.DefaultOptions()
		s := scanner.New(opts)

		listeners, err := s.FindByPattern(args[0])
		if err != nil {
			return err
		}

		if len(listeners) == 0 {
			fmt.Printf("No ports found matching '%s'\n", args[0])
			return nil
		}

		if jsonOutput {
			formatter := output.NewJSONFormatter(true)
			out, err := formatter.Format(listeners)
			if err != nil {
				return err
			}
			fmt.Println(out)
		} else {
			formatter := output.NewTableFormatter()
			formatter.NoHeader = noHeader
			fmt.Print(formatter.Format(listeners))
		}

		return nil
	},
}

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
	port, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid port: %s", args[0])
	}

	// Find process using the port
	opts := scanner.DefaultOptions()
	s := scanner.New(opts)

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
		fmt.Printf("Sent SIG%s to PID %d\n", strings.ToUpper(killSignal), pid)
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

var (
	waitTimeout  time.Duration
	waitInterval time.Duration
	waitExec     string
	waitQuiet    bool
	waitInvert   bool
)

func init() {
	waitCmd.Flags().DurationVar(&waitTimeout, "timeout", 30*time.Second, "Maximum wait time")
	waitCmd.Flags().DurationVarP(&waitInterval, "interval", "i", 100*time.Millisecond, "Check interval")
	waitCmd.Flags().StringVarP(&waitExec, "exec", "e", "", "Command to run once available")
	waitCmd.Flags().BoolVarP(&waitQuiet, "quiet", "q", false, "No output, just exit code")
	waitCmd.Flags().BoolVar(&waitInvert, "invert", false, "Wait for port to be FREE instead")
}

var waitCmd = &cobra.Command{
	Use:   "wait <port>",
	Short: "Wait until a port is available",
	Args:  cobra.ExactArgs(1),
	RunE:  runWait,
}

func runWait(cmd *cobra.Command, args []string) error {
	port, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid port: %s", args[0])
	}

	opts := scanner.DefaultOptions()
	s := scanner.New(opts)

	if !waitQuiet {
		if waitInvert {
			fmt.Printf("Waiting for port %d to be free...\n", port)
		} else {
			fmt.Printf("Waiting for port %d...\n", port)
		}
	}

	result := wait.Wait(s, port, waitTimeout, waitInterval, waitInvert)

	if !result.Success {
		if !waitQuiet {
			fmt.Printf("Timeout: port %d ", port)
			if waitInvert {
				fmt.Println("is still in use.")
			} else {
				fmt.Println("is not available.")
			}
		}
		os.Exit(1)
	}

	if !waitQuiet {
		if waitInvert {
			fmt.Printf("✓ Port %d is now free after %s\n", port, result.Elapsed.Round(time.Millisecond))
		} else {
			processInfo := ""
			if result.ProcessName != "" {
				processInfo = fmt.Sprintf(" (%s)", result.ProcessName)
			}
			fmt.Printf("✓ Port %d is now open%s after %s\n", port, processInfo, result.Elapsed.Round(time.Millisecond))
		}
	}

	// Execute command if --exec provided
	if waitExec != "" {
		execCmd := exec.Command("sh", "-c", waitExec)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		if err := execCmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			return err
		}
	}

	return nil
}

var portCmd = &cobra.Command{
	Use:   "port <port>",
	Short: "Show detailed information about a specific port",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid port: %s", args[0])
		}

		opts := scanner.DefaultOptions()
		s := scanner.New(opts)
		return showPortDetail(s, port)
	},
}

var pidCmd = &cobra.Command{
	Use:   "pid <pid>",
	Short: "Show all ports used by a specific process ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid pid: %s", args[0])
		}

		opts := scanner.DefaultOptions()
		s := scanner.New(opts)

		listeners, err := s.ListListeners()
		if err != nil {
			return err
		}

		var matches []model.Listener
		for _, l := range listeners {
			if l.PID == pid {
				matches = append(matches, l)
			}
		}

		if len(matches) == 0 {
			fmt.Printf("No ports found for PID %d\n", pid)
			return nil
		}

		output.SortListeners(matches, sortBy)

		if jsonOutput {
			formatter := output.NewJSONFormatter(true)
			out, err := formatter.Format(matches)
			if err != nil {
				return err
			}
			fmt.Println(out)
		} else {
			formatter := output.NewTableFormatter()
			formatter.NoHeader = noHeader
			fmt.Print(formatter.Format(matches))
		}

		return nil
	},
}
