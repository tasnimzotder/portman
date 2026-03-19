package tui

// Re-export shared styles for convenience within the tui package.
// All color and style definitions live in internal/style.
import "github.com/tasnimzotder/portman/internal/style"

var (
	StyleHeader  = style.Header
	StyleDim     = style.Dim
	StyleSubtle  = style.Subtle
	StyleSuccess = style.Success
	StyleError   = style.Error
	StyleWarning = style.Warning
	StyleBold    = style.Bold
	StyleSection = style.Section
	StyleAccent  = style.Accent
	StyleLabel   = style.Label

	ColorAccent  = style.ColorAccent
	ColorDim     = style.ColorDim
	ColorSubtle  = style.ColorSubtle
	ColorSuccess = style.ColorSuccess
	ColorError   = style.ColorError
	ColorWarning = style.ColorWarning
)
