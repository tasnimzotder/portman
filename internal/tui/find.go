package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/scanner"
)

type FindModel struct {
	table     table.Model
	scanner   scanner.Scanner
	pattern   string
	listeners []model.Listener
	err       error
	loaded    bool
}

func NewFindModel(s scanner.Scanner, pattern string) FindModel {
	return FindModel{
		table:   newPortTable(20),
		scanner: s,
		pattern: pattern,
	}
}

func (m FindModel) Init() tea.Cmd {
	return func() tea.Msg {
		listeners, err := m.scanner.FindByPattern(m.pattern)
		return listenersMsg{listeners: listeners, err: err}
	}
}

func (m FindModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case listenersMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.listeners = msg.listeners
		m.table.SetRows(buildPortRows(m.listeners))
		m.loaded = true
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m FindModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(StyleError.Render(fmt.Sprintf("Error: %v", m.err)) + "\n")
	}
	if !m.loaded {
		return tea.NewView(StyleDim.Render(fmt.Sprintf("  Searching for '%s'...", m.pattern)) + "\n")
	}
	if len(m.listeners) == 0 {
		return tea.NewView(StyleDim.Render(fmt.Sprintf("  No ports found matching '%s'.", m.pattern)) + "\n")
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(StyleHeader.Render(fmt.Sprintf("  portman find '%s'", m.pattern)))
	sb.WriteString("\n\n")
	sb.WriteString(m.table.View())
	sb.WriteString("\n")
	sb.WriteString(helpBar(fmt.Sprintf("%d matches", len(m.listeners)), "q quit"))
	sb.WriteString("\n")
	return tea.NewView(sb.String())
}

func (m FindModel) Err() error { return m.err }
