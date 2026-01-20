package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/scanner"
	"github.com/tasnimzotder/portman/internal/ui"
)

var (
	// Global flags
	jsonOutput    bool
	noHeader      bool
	tcpOnly       bool
	udpOnly       bool
	sortBy        string
	watchMode     bool
	watchInterval time.Duration
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
	RootCmd.PersistentFlags().BoolVarP(&watchMode, "watch", "w", false, "Live updating display")
	RootCmd.PersistentFlags().DurationVar(&watchInterval, "interval", time.Second, "Watch refresh interval")

	// Add subcommands
	RootCmd.AddCommand(findCmd)
	RootCmd.AddCommand(killCmd)
	RootCmd.AddCommand(waitCmd)
	RootCmd.AddCommand(portCmd)
	RootCmd.AddCommand(pidCmd)
}

// parsePort validates and returns a port number
func parsePort(s string) (int, error) {
	port, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid port: %s", s)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}
	return port, nil
}

func runRoot(cmd *cobra.Command, args []string) error {
	opts := scanner.DefaultOptions()
	if tcpOnly {
		opts.IncludeUDP = false
	}
	if udpOnly {
		opts.IncludeTCP = false
	}

	s, err := scanner.New(opts)
	if err != nil {
		return err
	}

	// If port specified, show details (or watch single port)
	if len(args) == 1 {
		port, err := parsePort(args[0])
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
	}

	// Otherwise list all
	return listAllPorts(s)
}

func listAllPorts(s scanner.Scanner) error {
	// Watch mode
	if watchMode {
		return ui.RunWatch(ui.WatchConfig{
			Scanner:  s,
			Interval: watchInterval,
			SortBy:   sortBy,
			TCPOnly:  tcpOnly,
			UDPOnly:  udpOnly,
		})
	}

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
