package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// interface Painter defines coloring text with [lipgloss] styles
type Painter interface {
	On(string, lipgloss.Color) string // Sets background color
	As(string, lipgloss.Color) string // Sets foreground color
}

type colors struct {
	title   lipgloss.Style
	success lipgloss.Style
	error   lipgloss.Style
	warning lipgloss.Style
	help    lipgloss.Style
}

var styles = colors{
	title:   NewBold("#7D56F4").MarginBottom(1),
	success: NewBold("#04B575"),
	error:   NewBold("#FF0000"),
	warning: NewStyle("#FFA500"),
	help:    NewEm("#626262"),
}

func NewStyle(fg string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(fg))
}

func NewBold(fg string) lipgloss.Style {
	return NewStyle(fg).Bold(true)
}

func NewEm(fg string) lipgloss.Style {
	return NewStyle(fg).Italic(true)
}
