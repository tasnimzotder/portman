package cli

import (
	"fmt"
	"strconv"
	"strings"
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
	outputFormat  string // "", "csv", "tsv"
)

var RootCmd = &cobra.Command{
	Use:               "portman [port]",
	Short:             "See what's using your ports",
	Long:              `portman — fast, minimal port inspector for macOS.`,
	Args:              cobra.MaximumNArgs(1),
	RunE:              runRoot,
	ValidArgsFunction: completePortArgs,
	SilenceErrors:     true,
	SilenceUsage:      true,
}

func init() {
	RootCmd.PersistentPreRunE = validateFlags
	RootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output as JSON")
	RootCmd.PersistentFlags().BoolVar(&noHeader, "no-header", false, "Omit header row")
	RootCmd.PersistentFlags().BoolVarP(&tcpOnly, "tcp", "t", false, "Show only TCP")
	RootCmd.PersistentFlags().BoolVarP(&udpOnly, "udp", "u", false, "Show only UDP")
	RootCmd.PersistentFlags().StringVar(&sortBy, "sort", "port", "Sort by: port, pid, user, conns, uptime")
	RootCmd.PersistentFlags().BoolVarP(&watchMode, "watch", "w", false, "Live updating display")
	RootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "Interactive TUI mode")
	RootCmd.PersistentFlags().DurationVar(&watchInterval, "interval", time.Second, "Watch refresh interval")
	RootCmd.PersistentFlags().BoolVarP(&grouped, "group", "g", false, "Group ports by process")
	RootCmd.PersistentFlags().StringVar(&outputFormat, "format", "", "Output format: csv, tsv")

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

// parsePortRange parses "3000-3010" into (3000, 3010, true) or returns false for single ports.
func parsePortRange(s string) (int, int, bool) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	lo, err1 := strconv.Atoi(parts[0])
	hi, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || lo < 1 || hi > 65535 || lo > hi {
		return 0, 0, false
	}
	return lo, hi, true
}

func newScanner() (scanner.Scanner, error) {
	return newScannerWithStats(true)
}

func newScannerWithStats(stats bool) (scanner.Scanner, error) {
	opts := scanner.DefaultOptions()
	opts.FetchStats = stats
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
		// Check for port range (e.g. "3000-3010")
		if lo, hi, ok := parsePortRange(args[0]); ok {
			return showPortRange(s, lo, hi)
		}

		port, err := parsePort(args[0])
		if err != nil {
			return err
		}

		if watchMode {
			return requireTerminal(func() error {
				m := tui.NewWatchPortModel(s, port, watchInterval)
				return runInspectionTUI(m)
			})
		}

		if interactive && !jsonOutput && outputFormat == "" && tui.IsTerminal() {
			m := tui.NewDetailModel(s, port)
			return runInspectionTUI(m)
		}

		return showPortDetail(s, port)
	}

	return listAllPorts(s)
}

func listAllPorts(s scanner.Scanner) error {
	if watchMode {
		return requireTerminal(func() error {
			m := tui.NewWatchModel(s, watchInterval, sortBy, tcpOnly, udpOnly)
			return runInspectionTUI(m)
		})
	}

	if interactive && !jsonOutput && outputFormat == "" && tui.IsTerminal() {
		m := tui.NewListModel(s, sortBy, tcpOnly, udpOnly)
		return runInspectionTUI(m)
	}

	listeners, err := s.ListListeners()
	if err != nil {
		return err
	}

	return printListeners(listeners)
}

func showPortDetail(s scanner.Scanner, port int) error {
	listener, err := s.GetPort(port)
	if err != nil {
		return err
	}

	if listener == nil {
		if outputFormat != "" || grouped && !jsonOutput {
			return printListeners(nil)
		}
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

	if outputFormat != "" || grouped {
		return printListeners([]model.Listener{*listener})
	}
	formatter := output.NewTableFormatter()
	fmt.Print(formatter.FormatDetail(listener))
	return nil
}

func showPortRange(s scanner.Scanner, lo, hi int) error {
	listeners, err := s.ListListeners()
	if err != nil {
		return err
	}

	var matches []model.Listener
	for _, l := range listeners {
		if l.Port >= lo && l.Port <= hi {
			matches = append(matches, l)
		}
	}

	return printListeners(matches)
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

func printListeners(listeners []model.Listener) error {
	listeners = model.FilterByProtocol(listeners, tcpOnly, udpOnly)
	output.SortListeners(listeners, sortBy)
	if jsonOutput {
		return printJSON(listeners)
	}
	if outputFormat != "" {
		fmt.Print(output.FormatDelimited(listeners, outputFormat, noHeader))
		return nil
	}
	f := output.NewTableFormatter()
	f.NoHeader = noHeader
	f.Grouped = grouped
	fmt.Print(f.Format(listeners))
	return nil
}

func validateFlags(cmd *cobra.Command, args []string) error {
	if tcpOnly && udpOnly {
		return fmt.Errorf("--tcp and --udp cannot be used together")
	}
	if outputFormat != "" && outputFormat != "csv" && outputFormat != "tsv" {
		return fmt.Errorf("unknown format %q (valid: csv, tsv)", outputFormat)
	}
	if jsonOutput && outputFormat != "" {
		return fmt.Errorf("--json and --format cannot be used together")
	}
	if (cmd == killCmd || cmd == waitCmd) && (jsonOutput || outputFormat != "") {
		return fmt.Errorf("--json and --format apply to inspection commands")
	}
	if watchMode && cmd != RootCmd && cmd != portCmd {
		return fmt.Errorf("--watch applies to port listings and port details")
	}
	if watchMode && len(args) == 1 && strings.Contains(args[0], "-") {
		return fmt.Errorf("watch mode does not support port ranges")
	}
	if watchMode && (jsonOutput || outputFormat != "") {
		return fmt.Errorf("watch mode cannot be combined with JSON or delimited output")
	}
	switch sortBy {
	case "port", "pid", "user", "conns", "uptime":
	default:
		return fmt.Errorf("unknown sort field %q", sortBy)
	}
	if watchInterval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	if cmd == waitCmd && (waitTimeout <= 0 || waitCmdInterval <= 0) {
		return fmt.Errorf("wait timeout and interval must be positive")
	}
	if cmd == killCmd && killTimeout <= 0 {
		return fmt.Errorf("kill timeout must be positive")
	}
	return nil
}

func runInspectionTUI(m tea.Model) error {
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}
	if result, ok := final.(interface{ Err() error }); ok {
		return result.Err()
	}
	return nil
}
