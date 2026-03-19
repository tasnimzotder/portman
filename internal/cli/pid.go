package cli

import (
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/tui"
)

var pidCmd = &cobra.Command{
	Use:   "pid <pid>",
	Short: "Show all ports used by a specific process ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := strconv.Atoi(args[0])
		if err != nil || pid <= 0 {
			return fmt.Errorf("invalid pid: %s (must be a positive integer)", args[0])
		}

		s, err := newScanner()
		if err != nil {
			return err
		}

		matches, err := s.ListByPID(pid)
		if err != nil {
			return err
		}

		if len(matches) == 0 {
			fmt.Printf("No ports found for PID %d\n", pid)
			return nil
		}

		// Interactive TUI
		if interactive && !jsonOutput && tui.IsTerminal() {
			m := tui.NewListModelWithData(matches, sortBy)
			_, err := tea.NewProgram(m).Run()
			return err
		}

		output.SortListeners(matches, sortBy)

		if jsonOutput {
			formatter := output.NewJSONFormatter(true)
			out, err := formatter.Format(matches)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		}

		// Default: tree view
		formatter := output.NewTableFormatter()
		fmt.Print(formatter.FormatTree(matches, pid))
		return nil
	},
}
