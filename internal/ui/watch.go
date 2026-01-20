package ui

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/scanner"
)

// WatchConfig holds configuration for watch mode (all ports)
type WatchConfig struct {
	Scanner  scanner.Scanner
	Interval time.Duration
	SortBy   string
	TCPOnly  bool
	UDPOnly  bool
}

// WatchPortConfig holds configuration for single-port watch mode
type WatchPortConfig struct {
	Scanner  scanner.Scanner
	Port     int
	Interval time.Duration
}

// PortSnapshot tracks previous values for change detection
type PortSnapshot struct {
	PID             int
	ConnectionCount int
	MemoryRSS       int64
	CPUPercent      float64
	FDCount         int
	ThreadCount     int
}

// RunWatchPort starts watch mode for a single port
func RunWatchPort(cfg WatchPortConfig) error {
	cleanup := setupTerminal()
	defer cleanup()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	keyChan := make(chan rune, 1)
	go func() {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return
		}
		defer tty.Close()
		buf := make([]byte, 1)
		for {
			n, err := tty.Read(buf)
			if err != nil || n == 0 {
				continue
			}
			keyChan <- rune(buf[0])
		}
	}()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	var prevListener *model.Listener
	var prevSnapshot *PortSnapshot
	isFirstRender := true

	renderPort := func() {
		// Only clear screen on first render, then just move cursor to top
		if isFirstRender {
			ClearAndReset()
			isFirstRender = false
		} else {
			MoveToTop()
		}

		// Header
		PrintLine("%s%sportman port %d%s  ", Bold, Cyan, cfg.Port, Reset)
		fmt.Printf("%sRefresh: %s%s  ", Dim, cfg.Interval, Reset)
		fmt.Printf("%sPress 'q' to quit%s\n", Dim, Reset)
		PrintLine("%s\n", strings.Repeat("─", 70))

		listener, _ := cfg.Scanner.GetPort(cfg.Port)

		if listener == nil {
			PrintLine("\n")
			PrintLine("%sPort %d is not in use.%s\n", Dim, cfg.Port, Reset)
			if prevListener != nil {
				PrintLine("%s  ● Process exited%s\n", Red, Reset)
			} else {
				PrintLine("\n")
			}
			// Clear all remaining lines (must cover full output: ~25 lines)
			for range 22 {
				PrintLine("\n")
			}
			prevListener = nil
			prevSnapshot = nil
			return
		}

		// Show if newly appeared
		if prevListener == nil {
			PrintLine("%s  ● Port became active%s\n", Green, Reset)
			PrintLine("\n")
		} else {
			PrintLine("\n")
			PrintLine("\n")
		}

		// Process info
		PrintLine("%sProcess%s\n", Bold, Reset)
		PrintLine("  PID:         %d\n", listener.PID)
		if listener.Process != nil {
			PrintLine("  Command:     %s\n", listener.Process.Name)
			PrintLine("  User:        %s\n", listener.Process.User)
			if listener.Process.UptimeSeconds > 0 {
				PrintLine("  Uptime:      %s\n", time.Duration(listener.Process.UptimeSeconds)*time.Second)
			} else {
				PrintLine("\n")
			}
		} else {
			PrintLine("\n")
			PrintLine("\n")
			PrintLine("\n")
		}

		PrintLine("\n")
		PrintLine("%sListening%s\n", Bold, Reset)
		PrintLine("  Address:     %s:%d\n", listener.Address, listener.Port)
		PrintLine("  Protocol:    %s\n", strings.ToUpper(listener.Protocol))

		// Connections with change highlighting
		connChanged := prevSnapshot != nil && listener.ConnectionCount != prevSnapshot.ConnectionCount
		PrintLine("\n")
		if connChanged {
			delta := listener.ConnectionCount - prevSnapshot.ConnectionCount
			sign := "+"
			if delta < 0 {
				sign = ""
			}
			PrintLine("%sConnections%s %s%d%s (%s%d)%s\n", Bold, Reset, Yellow, listener.ConnectionCount, Reset, sign, delta, Reset)
		} else {
			PrintLine("%sConnections%s (%d)\n", Bold, Reset, listener.ConnectionCount)
		}

		// Show connections (up to 5)
		if len(listener.Connections) > 0 {
			connCount := min(len(listener.Connections), 5)
			for i := range connCount {
				c := listener.Connections[i]
				PrintLine("  %s:%d  %s\n", c.RemoteAddr, c.RemotePort, c.State)
			}
			if len(listener.Connections) > 5 {
				PrintLine("  %s... and %d more%s\n", Dim, len(listener.Connections)-5, Reset)
			}
		}

		// Stats with change highlighting
		PrintLine("\n")
		PrintLine("%sStats%s\n", Bold, Reset)
		if listener.Stats != nil {
			// Memory
			memChanged := prevSnapshot != nil && listener.Stats.MemoryRSS != prevSnapshot.MemoryRSS
			if memChanged {
				PrintLine("  Memory:      %s%s%s\n", Yellow, output.FormatBytes(listener.Stats.MemoryRSS), Reset)
			} else {
				PrintLine("  Memory:      %s\n", output.FormatBytes(listener.Stats.MemoryRSS))
			}

			// CPU
			cpuChanged := prevSnapshot != nil && listener.Stats.CPUPercent != prevSnapshot.CPUPercent
			if cpuChanged {
				PrintLine("  CPU:         %s%.1f%%%s\n", Yellow, listener.Stats.CPUPercent, Reset)
			} else {
				PrintLine("  CPU:         %.1f%%\n", listener.Stats.CPUPercent)
			}

			// FDs
			fdChanged := prevSnapshot != nil && listener.Stats.FDCount != prevSnapshot.FDCount
			if fdChanged {
				PrintLine("  FDs:         %s%d%s\n", Yellow, listener.Stats.FDCount, Reset)
			} else {
				PrintLine("  FDs:         %d\n", listener.Stats.FDCount)
			}

			// Threads
			threadChanged := prevSnapshot != nil && listener.Stats.ThreadCount != prevSnapshot.ThreadCount
			if threadChanged {
				PrintLine("  Threads:     %s%d%s\n", Yellow, listener.Stats.ThreadCount, Reset)
			} else {
				PrintLine("  Threads:     %d\n", listener.Stats.ThreadCount)
			}

			// Update snapshot
			prevSnapshot = &PortSnapshot{
				PID:             listener.PID,
				ConnectionCount: listener.ConnectionCount,
				MemoryRSS:       listener.Stats.MemoryRSS,
				CPUPercent:      listener.Stats.CPUPercent,
				FDCount:         listener.Stats.FDCount,
				ThreadCount:     listener.Stats.ThreadCount,
			}
		} else {
			PrintLine("  %sNo stats available%s\n", Dim, Reset)
			prevSnapshot = &PortSnapshot{
				PID:             listener.PID,
				ConnectionCount: listener.ConnectionCount,
			}
		}

		// Clear any leftover lines from previous renders (e.g., when connections decrease)
		for range 6 {
			PrintLine("\n")
		}

		prevListener = listener
	}

	renderPort()

	for {
		select {
		case <-sigChan:
			return nil
		case key := <-keyChan:
			if key == 'q' || key == 'Q' {
				return nil
			}
		case <-ticker.C:
			renderPort()
		}
	}
}

// WatchState tracks the current state of watch mode
type WatchState struct {
	config        WatchConfig
	previous      map[int]model.Listener
	added         map[int]bool
	removed       map[int]bool
	isFirstRender bool
}

// RunWatch starts watch mode with live updates
func RunWatch(cfg WatchConfig) error {
	state := &WatchState{
		config:        cfg,
		previous:      make(map[int]model.Listener),
		added:         make(map[int]bool),
		removed:       make(map[int]bool),
		isFirstRender: true,
	}

	// Set up terminal
	cleanup := setupTerminal()
	defer cleanup()

	// Handle signals for clean exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Key input channel - read from /dev/tty for raw input
	keyChan := make(chan rune, 1)
	go func() {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return
		}
		defer tty.Close()

		buf := make([]byte, 1)
		for {
			n, err := tty.Read(buf)
			if err != nil || n == 0 {
				continue
			}
			keyChan <- rune(buf[0])
		}
	}()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	// Initial render
	if err := state.update(); err != nil {
		return err
	}
	state.render()

	for {
		select {
		case <-sigChan:
			return nil
		case key := <-keyChan:
			if key == 'q' || key == 'Q' {
				return nil
			}
		case <-ticker.C:
			if err := state.update(); err != nil {
				continue // Keep trying on errors
			}
			state.render()
		}
	}
}

// setupTerminal prepares the terminal for watch mode
func setupTerminal() func() {
	// Put terminal in raw mode for single-key input (macOS compatible)
	// Use -f /dev/tty for macOS
	tty, _ := os.Open("/dev/tty")
	if tty != nil {
		cmd := exec.Command("stty", "-f", "/dev/tty", "cbreak", "-echo")
		cmd.Stdin = tty
		cmd.Run()
		tty.Close()
	}

	Hide()
	ClearAndReset()

	return func() {
		Show()
		// Restore terminal
		tty, _ := os.Open("/dev/tty")
		if tty != nil {
			cmd := exec.Command("stty", "-f", "/dev/tty", "sane")
			cmd.Stdin = tty
			cmd.Run()
			tty.Close()
		}
		fmt.Println()
	}
}

// update fetches new data and computes diff
func (s *WatchState) update() error {
	listeners, err := s.config.Scanner.ListListeners()
	if err != nil {
		return err
	}

	// Apply filters
	if s.config.TCPOnly {
		filtered := make([]model.Listener, 0)
		for _, l := range listeners {
			if l.Protocol == "tcp" {
				filtered = append(filtered, l)
			}
		}
		listeners = filtered
	}
	if s.config.UDPOnly {
		filtered := make([]model.Listener, 0)
		for _, l := range listeners {
			if l.Protocol == "udp" {
				filtered = append(filtered, l)
			}
		}
		listeners = filtered
	}

	// Sort
	output.SortListeners(listeners, s.config.SortBy)

	// Compute diff
	current := make(map[int]model.Listener)
	for _, l := range listeners {
		current[l.Port] = l
	}

	// Find added ports
	s.added = make(map[int]bool)
	for port := range current {
		if _, exists := s.previous[port]; !exists {
			s.added[port] = true
		}
	}

	// Find removed ports
	s.removed = make(map[int]bool)
	for port := range s.previous {
		if _, exists := current[port]; !exists {
			s.removed[port] = true
		}
	}

	s.previous = current
	return nil
}
