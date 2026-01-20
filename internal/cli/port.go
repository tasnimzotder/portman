package cli

import (
	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/scanner"
	"github.com/tasnimzotder/portman/internal/ui"
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

		opts := scanner.DefaultOptions()
		s, err := scanner.New(opts)
		if err != nil {
			return err
		}

		if watchMode {
			return ui.RunWatchPort(ui.WatchPortConfig{
				Scanner:  s,
				Port:     port,
				Interval: watchInterval,
			})
		}

		return showPortDetail(s, port)
	},
}
