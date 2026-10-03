package cli

import (
	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/tui"
)

var findCmd = &cobra.Command{
	Use:               "find <pattern>",
	Short:             "Find ports by process name, command, or user",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProcessArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := newScanner()
		if err != nil {
			return err
		}

		// Interactive TUI
		if interactive && !jsonOutput && outputFormat == "" && tui.IsTerminal() {
			m := tui.NewFindModel(s, args[0])
			return runInspectionTUI(m)
		}

		listeners, err := s.FindByPattern(args[0])
		if err != nil {
			return err
		}

		return printListeners(listeners)
	},
}
