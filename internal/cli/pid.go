package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/model"
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

		// Interactive TUI
		if interactive && !jsonOutput && outputFormat == "" && tui.IsTerminal() {
			m := tui.NewListModelWithData(matches, sortBy, s)
			return runInspectionTUI(m)
		}

		if jsonOutput || outputFormat != "" || grouped {
			return printListeners(matches)
		}
		matches = model.FilterByProtocol(matches, tcpOnly, udpOnly)
		output.SortListeners(matches, sortBy)

		// Default: tree view
		formatter := output.NewTableFormatter()
		fmt.Print(formatter.FormatTree(matches, pid))
		return nil
	},
}
