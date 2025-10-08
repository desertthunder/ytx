package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var styles = NewPalette("#7D56F4", "#04B575", "#FF0000", "#FFA500", "#626262")

// interface Painter defines coloring text with [lipgloss] styles
type Painter interface {
	On(string, lipgloss.Color) string // Sets background color
	As(string, lipgloss.Color) string // Sets foreground color
}

// struct Palette is a simple stylesheet built with named [lipgloss.Style] fields
type Palette struct {
	title lipgloss.Style
	ok    lipgloss.Style
	err   lipgloss.Style
	warn  lipgloss.Style
	help  lipgloss.Style
}

func NewPalette(t, s, e, w, h string) *Palette {
	return &Palette{
		title: NewBold(t).MarginBottom(1),
		ok:    NewBold(s),
		err:   NewBold(e),
		warn:  NewStyle(w),
		help:  NewEm(h),
	}
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
