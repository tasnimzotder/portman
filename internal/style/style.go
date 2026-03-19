// Package style defines the shared color palette and lipgloss styles
// used across both static output and interactive TUI.
package style

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/tasnimzotder/portman/internal/model"
)

// Minimal mono color palette — mostly white/gray with one accent.
var (
	ColorAccent  = lipgloss.Color("#5F87FF")
	ColorDim     = lipgloss.Color("#6C6C6C")
	ColorSubtle  = lipgloss.Color("#4E4E4E")
	ColorSuccess = lipgloss.Color("#57CC99")
	ColorError   = lipgloss.Color("#FF6B6B")
	ColorWarning = lipgloss.Color("#FFD166")
)

// Shared styles used by output formatters and TUI models.
var (
	Header  = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	Dim     = lipgloss.NewStyle().Foreground(ColorDim)
	Subtle  = lipgloss.NewStyle().Foreground(ColorSubtle)
	Success = lipgloss.NewStyle().Foreground(ColorSuccess)
	Error   = lipgloss.NewStyle().Foreground(ColorError)
	Warning = lipgloss.NewStyle().Foreground(ColorWarning)
	Bold    = lipgloss.NewStyle().Bold(true)
	Section = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent).MarginTop(1)
	Accent  = lipgloss.NewStyle().Foreground(ColorAccent)
	Label   = lipgloss.NewStyle().Foreground(ColorDim)
	Port    = lipgloss.NewStyle().Bold(true)
)

// ConnState returns a styled connection state string.
func ConnState(state string) string {
	switch state {
	case model.StateEstablished:
		return Success.Render(state)
	case model.StateCloseWait, model.StateTimeWait:
		return Warning.Render(state)
	default:
		return state
	}
}

// Proto returns a styled protocol string.
func Proto(protocol string) string {
	switch protocol {
	case model.ProtoTCP:
		return Accent.Render(protocol)
	case model.ProtoUDP:
		return Warning.Render(protocol)
	default:
		return protocol
	}
}

// KV formats a label-value pair for detail views.
func KV(key, value string) string {
	return fmt.Sprintf("    %s %s\n",
		Label.Render(fmt.Sprintf("%-12s", key+":")),
		value,
	)
}

// Separator returns a dim horizontal rule of the given width.
func Separator(width int) string {
	return Subtle.Render("  " + strings.Repeat("─", width))
}

// HelpBar renders a dim help text line with available keyboard shortcuts.
func HelpBar(items ...string) string {
	return Dim.Render(fmt.Sprintf("  %s", strings.Join(items, " · ")))
}
