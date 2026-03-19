package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/tui"
	"github.com/tasnimzotder/portman/internal/wait"
)

var (
	waitTimeout     time.Duration
	waitCmdInterval time.Duration
	waitExec        string
	waitQuiet       bool
	waitInvert      bool
)

func init() {
	waitCmd.Flags().DurationVar(&waitTimeout, "timeout", 30*time.Second, "Maximum wait time")
	waitCmd.Flags().DurationVarP(&waitCmdInterval, "interval", "I", 100*time.Millisecond, "Check interval")
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

func runExecCommand(cmdStr string) error {
	execCmd := exec.Command("sh", "-c", cmdStr)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin
	if err := execCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

func runWait(cmd *cobra.Command, args []string) error {
	port, err := parsePort(args[0])
	if err != nil {
		return err
	}

	s, err := newScanner()
	if err != nil {
		return err
	}

	// TUI wait for interactive terminals
	if !waitQuiet && tui.IsTerminal() {
		m := tui.NewWaitModel(s, port, waitTimeout, waitCmdInterval, waitInvert)
		if _, err := tea.NewProgram(m).Run(); err != nil {
			return err
		}
		if waitExec != "" {
			return runExecCommand(waitExec)
		}
		return nil
	}

	// Non-interactive wait
	if !waitQuiet {
		if waitInvert {
			fmt.Printf("Waiting for port %d to be free...\n", port)
		} else {
			fmt.Printf("Waiting for port %d...\n", port)
		}
	}

	result := wait.Wait(s, port, waitTimeout, waitCmdInterval, waitInvert)

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
			fmt.Printf("Port %d is now free after %s\n", port, result.Elapsed.Round(time.Millisecond))
		} else {
			processInfo := ""
			if result.ProcessName != "" {
				processInfo = fmt.Sprintf(" (%s)", result.ProcessName)
			}
			fmt.Printf("Port %d is now open%s after %s\n", port, processInfo, result.Elapsed.Round(time.Millisecond))
		}
	}

	if waitExec != "" {
		return runExecCommand(waitExec)
	}

	return nil
}
