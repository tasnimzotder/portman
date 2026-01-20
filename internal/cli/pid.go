package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/scanner"
)

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
		s, err := scanner.New(opts)
		if err != nil {
			return err
		}

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
