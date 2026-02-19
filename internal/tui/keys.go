package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	NextFile  key.Binding
	PrevFile  key.Binding
	NextHunk  key.Binding
	PrevHunk  key.Binding
	Toggle    key.Binding
	Trace     key.Binding
	FocusSwap key.Binding
	Search    key.Binding
	Help      key.Binding
	Approve   key.Binding
	Reject    key.Binding
	Undo      key.Binding
	Finish    key.Binding
	Quit      key.Binding
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
		key.WithKeys("n"),
		key.WithHelp("n", "next file"),
	),
	PrevFile: key.NewBinding(
		key.WithKeys("N"),
		key.WithHelp("N", "prev file"),
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
	Trace: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "toggle trace"),
	),
	FocusSwap: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch panel"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Approve: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "approve file"),
	),
	Reject: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "reject file"),
	),
	Undo: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "undo decision"),
	),
	Finish: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "finish review"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
