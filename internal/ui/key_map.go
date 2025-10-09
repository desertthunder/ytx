package ui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines the [key.Binding] mapping for the TUI.
type keyMap struct {
	up      key.Binding
	down    key.Binding
	enter   key.Binding
	back    key.Binding
	yes     key.Binding
	no      key.Binding
	restart key.Binding
	quit    key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		yes:     key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yes")),
		no:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "no")),
		restart: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart")),
		quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.up, k.down, k.enter},
		{k.back, k.yes, k.no},
		{k.restart, k.quit},
	}
}
