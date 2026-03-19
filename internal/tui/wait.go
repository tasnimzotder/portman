package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/tasnimzotder/portman/internal/scanner"
)

type waitPhase int

const (
	waitPolling waitPhase = iota
	waitSuccess
	waitTimeout
)

type waitCheckMsg struct {
	found       bool
	processName string
}

type WaitModel struct {
	scanner   scanner.Scanner
	port      int
	timeout   time.Duration
	interval  time.Duration
	invert    bool
	spinner   spinner.Model
	phase     waitPhase
	startTime time.Time
	elapsed   time.Duration
	result    string
}

func NewWaitModel(s scanner.Scanner, port int, timeout, interval time.Duration, invert bool) WaitModel {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorAccent)),
	)

	return WaitModel{
		scanner:   s,
		port:      port,
		timeout:   timeout,
		interval:  interval,
		invert:    invert,
		spinner:   sp,
		phase:     waitPolling,
		startTime: time.Now(),
	}
}

func (m WaitModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.check(), tick(m.interval))
}

func (m WaitModel) check() tea.Cmd {
	return func() tea.Msg {
		listener, err := m.scanner.GetPort(m.port)
		if err != nil {
			if m.invert {
				return waitCheckMsg{found: true, processName: ""}
			}
			return waitCheckMsg{found: false, processName: ""}
		}

		if m.invert {
			if listener == nil {
				return waitCheckMsg{found: true, processName: ""}
			}
			return waitCheckMsg{found: false, processName: ""}
		}

		if listener != nil {
			processName := ""
			if listener.Process != nil {
				processName = listener.Process.Name
			}
			return waitCheckMsg{found: true, processName: processName}
		}
		return waitCheckMsg{found: false, processName: ""}
	}
}

func (m WaitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case waitCheckMsg:
		if msg.found {
			m.phase = waitSuccess
			elapsed := time.Since(m.startTime).Round(time.Millisecond)
			if m.invert {
				m.result = fmt.Sprintf("Port %d is now free after %s", m.port, elapsed)
			} else {
				processInfo := ""
				if msg.processName != "" {
					processInfo = fmt.Sprintf(" (%s)", msg.processName)
				}
				m.result = fmt.Sprintf("Port %d is now open%s after %s", m.port, processInfo, elapsed)
			}
			return m, nil
		}
		if m.elapsed >= m.timeout {
			m.phase = waitTimeout
			if m.invert {
				m.result = fmt.Sprintf("Timeout: port %d is still in use.", m.port)
			} else {
				m.result = fmt.Sprintf("Timeout: port %d is not available.", m.port)
			}
			return m, nil
		}
		return m, nil

	case tickMsg:
		m.elapsed = time.Since(m.startTime)
		if m.phase == waitPolling {
			return m, tea.Batch(m.check(), tick(m.interval))
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		default:
			if m.phase == waitSuccess || m.phase == waitTimeout {
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m WaitModel) View() tea.View {
	switch m.phase {
	case waitPolling:
		elapsed := m.elapsed.Round(time.Second)
		action := "available"
		if m.invert {
			action = "free"
		}
		s := fmt.Sprintf("\n  %s Waiting for port %d to be %s...  %s\n",
			m.spinner.View(),
			m.port,
			action,
			StyleDim.Render(fmt.Sprintf("%s", elapsed)),
		)
		return tea.NewView(s)

	case waitSuccess:
		return tea.NewView("\n  " + StyleSuccess.Render(m.result) + "\n")

	case waitTimeout:
		return tea.NewView("\n  " + StyleError.Render(m.result) + "\n")
	}

	return tea.NewView("")
}
