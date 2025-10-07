package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// interface Painter defines coloring text with [lipgloss] styles
type Painter interface {
	On(string, lipgloss.Color) string // Sets background color
	As(string, lipgloss.Color) string // Sets foreground color
}
