package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	NextFile key.Binding
	PrevFile key.Binding
	NextHunk key.Binding
	PrevHunk key.Binding
	Toggle   key.Binding
	Search   key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	NextFile: key.NewBinding(
		key.WithKeys("n", "tab"),
		key.WithHelp("n/tab", "next file"),
	),
	PrevFile: key.NewBinding(
		key.WithKeys("N", "shift+tab"),
		key.WithHelp("N/S-tab", "prev file"),
	),
	NextHunk: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "next hunk"),
	),
	PrevHunk: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "prev hunk"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "unified/split"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
