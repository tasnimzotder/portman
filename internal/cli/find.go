package cli

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/tui"
)

var findCmd = &cobra.Command{
	Use:   "find <pattern>",
	Short: "Find ports by process name, command, or user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := newScanner()
		if err != nil {
			return err
		}

		// Interactive TUI
		if interactive && !jsonOutput && tui.IsTerminal() {
			m := tui.NewFindModel(s, args[0])
			_, err := tea.NewProgram(m).Run()
			return err
		}

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
