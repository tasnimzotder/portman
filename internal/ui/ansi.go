package ui

import "fmt"

// ANSI escape codes for terminal control
const (
	ClearScreen = "\033[2J"
	MoveCursor  = "\033[H"
	ClearLine   = "\033[2K"
	HideCursor  = "\033[?25l"
	ShowCursor  = "\033[?25h"

	// Colors
	Green  = "\033[32m"
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Reset  = "\033[0m"
)

// ClearAndReset clears the screen and moves cursor to top-left
func ClearAndReset() {
	fmt.Print(ClearScreen + MoveCursor)
}

// MoveToTop moves cursor to top-left without clearing screen
func MoveToTop() {
	fmt.Print(MoveCursor)
}

// PrintLine clears current line and prints text
func PrintLine(format string, args ...any) {
	fmt.Print(ClearLine)
	fmt.Printf(format, args...)
}

// Hide hides the cursor
func Hide() {
	fmt.Print(HideCursor)
}

// Show shows the cursor
func Show() {
	fmt.Print(ShowCursor)
}
