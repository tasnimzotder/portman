package tui

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/tasnimzotder/portman/internal/kill"
	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/scanner"
)

type killPhase int

const (
	killLoading killPhase = iota
	killConfirm
	killSending
	killDone
)

type killResultMsg struct {
	success bool
	message string
}

type KillModel struct {
	scanner  scanner.Scanner
	port     int
	signal   syscall.Signal
	sigName  string
	force    bool
	listener *model.Listener
	phase    killPhase
	spinner  spinner.Model
	result   string
	err      error
}

func NewKillModel(s scanner.Scanner, port int, sig syscall.Signal, sigName string, force bool) KillModel {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorAccent)),
	)
	return KillModel{
		scanner: s,
		port:    port,
		signal:  sig,
		sigName: sigName,
		force:   force,
		phase:   killLoading,
		spinner: sp,
	}
}

func (m KillModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchPort())
}

func (m KillModel) fetchPort() tea.Cmd {
	return func() tea.Msg {
		listener, err := m.scanner.GetPort(m.port)
		return portMsg{listener: listener, err: err}
	}
}

func (m KillModel) doKill() tea.Cmd {
	pid := m.listener.PID
	sig := m.signal
	force := m.force

	return func() tea.Msg {
		err := kill.Kill(pid, sig)
		if err != nil {
			if errors.Is(err, kill.ErrPermissionDenied) {
				return killResultMsg{
					success: false,
					message: "Permission denied. Try running with sudo.",
				}
			}
			return killResultMsg{
				success: false,
				message: fmt.Sprintf("Error: %v", err),
			}
		}

		if kill.WaitForExit(pid, 3*time.Second) {
			return killResultMsg{
				success: true,
				message: "Process terminated.",
			}
		}

		if force && sig != syscall.SIGKILL {
			if err := kill.Kill(pid, syscall.SIGKILL); err != nil {
				return killResultMsg{
					success: false,
					message: fmt.Sprintf("SIGKILL failed: %v", err),
				}
			}
			if kill.WaitForExit(pid, 2*time.Second) {
				return killResultMsg{
					success: true,
					message: "Process killed (SIGKILL).",
				}
			}
		}

		return killResultMsg{
			success: false,
			message: "Process didn't terminate.",
		}
	}
}

func (m KillModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case portMsg:
		if msg.err != nil {
			m.err = msg.err
			m.result = fmt.Sprintf("Error: %v", msg.err)
			m.phase = killDone
			return m, nil
		}
		if msg.listener == nil {
			m.result = StyleDim.Render(fmt.Sprintf("Port %d is not in use.", m.port))
			m.phase = killDone
			return m, nil
		}
		m.listener = msg.listener
		m.phase = killConfirm
		return m, nil

	case killResultMsg:
		if msg.success {
			m.result = StyleSuccess.Render(msg.message)
		} else {
			m.result = StyleError.Render(msg.message)
		}
		m.phase = killDone
		return m, nil

	case tea.KeyPressMsg:
		switch m.phase {
		case killConfirm:
			switch msg.String() {
			case "y", "Y":
				m.phase = killSending
				return m, tea.Batch(m.spinner.Tick, m.doKill())
			case "n", "N", "q", "ctrl+c":
				m.result = StyleDim.Render("Aborted.")
				m.phase = killDone
				return m, nil
			}
		case killDone:
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m KillModel) View() tea.View {
	switch m.phase {
	case killLoading:
		return tea.NewView(fmt.Sprintf("\n  %s Scanning port %d...\n", m.spinner.View(), m.port))

	case killConfirm:
		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString(StyleHeader.Render(fmt.Sprintf("  Kill process on port %d?", m.port)))
		sb.WriteString("\n\n")

		if m.listener != nil {
			pid := m.listener.PID
			processName := m.listener.ProcessName()
			userName := m.listener.ProcessUser()
			uptime := ""
			if m.listener.Process != nil && m.listener.Process.UptimeSeconds > 0 {
				uptime = (time.Duration(m.listener.Process.UptimeSeconds) * time.Second).String()
			}

			sb.WriteString(fmt.Sprintf("    %s %s\n", StyleLabel.Render("Process:"), processName))
			sb.WriteString(fmt.Sprintf("    %s %d\n", StyleLabel.Render("PID:    "), pid))
			sb.WriteString(fmt.Sprintf("    %s %s\n", StyleLabel.Render("User:   "), userName))
			if uptime != "" {
				sb.WriteString(fmt.Sprintf("    %s %s\n", StyleLabel.Render("Uptime: "), uptime))
			}
			sb.WriteString(fmt.Sprintf("    %s SIG%s\n", StyleLabel.Render("Signal: "), strings.ToUpper(m.sigName)))
		}

		sb.WriteString("\n")
		sb.WriteString(StyleBold.Render("  Confirm? [y/n]: "))
		return tea.NewView(sb.String())

	case killSending:
		pid := 0
		if m.listener != nil {
			pid = m.listener.PID
		}
		return tea.NewView(fmt.Sprintf("\n  %s Sending SIG%s to PID %d...\n",
			m.spinner.View(), strings.ToUpper(m.sigName), pid))

	case killDone:
		return tea.NewView("\n  " + m.result + "\n")
	}

	return tea.NewView("")
}
