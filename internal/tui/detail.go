package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/output"
	"github.com/tasnimzotder/portman/internal/scanner"
	"github.com/tasnimzotder/portman/internal/style"
)

type DetailModel struct {
	scanner  scanner.Scanner
	port     int
	listener *model.Listener
	err      error
	loaded   bool
}

func NewDetailModel(s scanner.Scanner, port int) DetailModel {
	return DetailModel{scanner: s, port: port}
}

func (m DetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		listener, err := m.scanner.GetPort(m.port)
		return portMsg{listener: listener, err: err}
	}
}

func (m DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case portMsg:
		m.listener = msg.listener
		m.err = msg.err
		m.loaded = true
		return m, nil
	case tea.KeyPressMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m DetailModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(style.Error.Render(fmt.Sprintf("Error: %v", m.err)) + "\n")
	}
	if !m.loaded {
		return tea.NewView(style.Dim.Render("  Loading port details...") + "\n")
	}
	if m.listener == nil {
		return tea.NewView(style.Dim.Render(fmt.Sprintf("  Port %d is not in use.", m.port)) + "\n")
	}

	content := "\n" + output.RenderDetail(m.listener)
	content += style.Dim.Render("  Press any key to exit") + "\n"
	return tea.NewView(content)
}

func (m DetailModel) Err() error { return m.err }
