package cli

import (
	"fmt"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/scanner"
	"github.com/tasnimzotder/portman/internal/tui"
)

var (
	jsonOutput    bool
	noHeader      bool
	tcpOnly       bool
	udpOnly       bool
	sortBy        string
	watchMode     bool
	interactive   bool
	watchInterval time.Duration
	grouped       bool
)

var RootCmd = &cobra.Command{
	Use:   "portman [port]",
	Short: "See what's using your ports",
	Long:  `portman — fast, minimal port inspector for macOS.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRoot,
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	RootCmd.PersistentFlags().BoolVar(&noHeader, "no-header", false, "Omit header row")
	RootCmd.PersistentFlags().BoolVarP(&tcpOnly, "tcp", "t", false, "Show only TCP")
	RootCmd.PersistentFlags().BoolVarP(&udpOnly, "udp", "u", false, "Show only UDP")
	RootCmd.PersistentFlags().StringVar(&sortBy, "sort", "port", "Sort by: port, pid, user, conns, uptime")
	RootCmd.PersistentFlags().BoolVarP(&watchMode, "watch", "w", false, "Live updating display")
	RootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "Interactive TUI mode")
	RootCmd.PersistentFlags().DurationVar(&watchInterval, "interval", time.Second, "Watch refresh interval")
	RootCmd.PersistentFlags().BoolVarP(&grouped, "group", "g", false, "Group ports by process")

	RootCmd.AddCommand(findCmd)
	RootCmd.AddCommand(killCmd)
	RootCmd.AddCommand(waitCmd)
	RootCmd.AddCommand(portCmd)
	RootCmd.AddCommand(pidCmd)
}

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

func newScanner() (scanner.Scanner, error) {
	opts := scanner.DefaultOptions()
	if tcpOnly {
		opts.IncludeUDP = false
	}
	if udpOnly {
		opts.IncludeTCP = false
	}
	return scanner.New(opts)
}

func runRoot(cmd *cobra.Command, args []string) error {
	s, err := newScanner()
	if err != nil {
		return err
	}

	if len(args) == 1 {
		port, err := parsePort(args[0])
		if err != nil {
			return err
		}

		if watchMode {
			return requireTerminal(func() error {
				m := tui.NewWatchPortModel(s, port, watchInterval)
				_, err := tea.NewProgram(m).Run()
				return err
			})
		}

		if interactive && tui.IsTerminal() {
			m := tui.NewDetailModel(s, port)
			_, err := tea.NewProgram(m).Run()
			return err
		}

		return showPortDetail(s, port)
	}

	return listAllPorts(s)
}

func listAllPorts(s scanner.Scanner) error {
	if watchMode {
		return requireTerminal(func() error {
			m := tui.NewWatchModel(s, watchInterval, sortBy, tcpOnly, udpOnly)
			_, err := tea.NewProgram(m).Run()
			return err
		})
	}

	if interactive && tui.IsTerminal() {
		m := tui.NewListModel(s, sortBy, tcpOnly, udpOnly)
		_, err := tea.NewProgram(m).Run()
		return err
	}

	listeners, err := s.ListListeners()
	if err != nil {
		return err
	}

	listeners = model.FilterByProtocol(listeners, tcpOnly, udpOnly)
	output.SortListeners(listeners, sortBy)

	if jsonOutput {
		return printJSON(listeners)
	}

	formatter := output.NewTableFormatter()
	formatter.NoHeader = noHeader
	formatter.Grouped = grouped
	fmt.Print(formatter.Format(listeners))
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
		return printJSONSingle(listener)
	}

	formatter := output.NewTableFormatter()
	fmt.Print(formatter.FormatDetail(listener))
	return nil
}

// requireTerminal validates TTY then runs the given function.
func requireTerminal(fn func() error) error {
	if !tui.IsTerminal() {
		return fmt.Errorf("this mode requires an interactive terminal")
	}
	return fn()
}

func printJSON(listeners []model.Listener) error {
	f := output.NewJSONFormatter(true)
	out, err := f.Format(listeners)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func printJSONSingle(l *model.Listener) error {
	f := output.NewJSONFormatter(true)
	out, err := f.FormatSingle(l)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}
