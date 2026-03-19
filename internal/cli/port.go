package cli

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/tui"
)

var portCmd = &cobra.Command{
	Use:   "port <port>",
	Short: "Show detailed information about a specific port",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := parsePort(args[0])
		if err != nil {
			return err
		}

		s, err := newScanner()
		if err != nil {
			return err
		}

		if watchMode {
			if !tui.IsTerminal() {
				return fmt.Errorf("watch mode requires an interactive terminal")
			}
			m := tui.NewWatchPortModel(s, port, watchInterval)
			_, err := tea.NewProgram(m).Run()
			return err
		}

		if interactive && tui.IsTerminal() {
			m := tui.NewDetailModel(s, port)
			_, err := tea.NewProgram(m).Run()
			return err
		}

		return showPortDetail(s, port)
	},
}
