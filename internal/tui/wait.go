package tui

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tasnimzotder/portman/internal/scanner"
	portwait "github.com/tasnimzotder/portman/internal/wait"
)

type waitPhase int

const (
	waitPolling waitPhase = iota
	waitSuccess
	waitTimeout
)

type waitResultMsg struct{ result portwait.Result }

type WaitModel struct {
	scanner           scanner.Scanner
	port              int
	timeout, interval time.Duration
	invert            bool
	spinner           spinner.Model
	phase             waitPhase
	startTime         time.Time
	elapsed           time.Duration
	result            string
	outcome           portwait.Result
	ctx               context.Context
	cancel            context.CancelFunc
}

func NewWaitModel(s scanner.Scanner, port int, timeout, interval time.Duration, invert bool) WaitModel {
	ctx, cancel := context.WithCancel(context.Background())
	return WaitModel{scanner: s, port: port, timeout: timeout, interval: interval, invert: invert,
		spinner:   spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorAccent))),
		startTime: time.Now(), ctx: ctx, cancel: cancel}
}
func (m WaitModel) Cancel()                  { m.cancel() }
func (m WaitModel) Outcome() portwait.Result { return m.outcome }
func (m WaitModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tick(100*time.Millisecond), func() tea.Msg {
		return waitResultMsg{portwait.WaitContext(m.ctx, m.scanner, m.port, m.timeout, m.interval, m.invert)}
	})
}
func (m WaitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case waitResultMsg:
		if m.phase != waitPolling {
			return m, nil
		}
		m.outcome = msg.result
		m.elapsed = msg.result.Elapsed
		if msg.result.Success {
			m.phase = waitSuccess
			action := "open"
			if m.invert {
				action = "free"
			}
			m.result = fmt.Sprintf("Port %d is now %s after %s", m.port, action, m.elapsed.Round(time.Millisecond))
		} else {
			m.phase = waitTimeout
			m.result = fmt.Sprintf("Port %d: %v", m.port, msg.result.Err)
		}
		return m, tea.Quit
	case tickMsg:
		m.elapsed = time.Since(m.startTime)
		if m.phase == waitPolling {
			return m, tick(100 * time.Millisecond)
		}
	case spinner.TickMsg:
		if m.phase == waitPolling {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case tea.KeyPressMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" || m.phase != waitPolling {
			if m.phase == waitPolling {
				m.outcome = portwait.Result{Err: context.Canceled, Elapsed: time.Since(m.startTime)}
				m.phase = waitTimeout
			}
			m.cancel()
			return m, tea.Quit
		}
	}
	return m, nil
}
func (m WaitModel) View() tea.View {
	if m.phase == waitSuccess {
		return tea.NewView("\n  " + StyleSuccess.Render(m.result) + "\n")
	}
	if m.phase == waitTimeout {
		return tea.NewView("\n  " + StyleError.Render(m.result) + "\n")
	}
	action := "available"
	if m.invert {
		action = "free"
	}
	return tea.NewView(fmt.Sprintf("\n  %s Waiting for port %d to be %s... %s\n", m.spinner.View(), m.port, action, m.elapsed.Round(time.Second)))
}
