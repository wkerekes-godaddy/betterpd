package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Select  key.Binding
	Ack     key.Binding
	Resolve key.Binding
	Snooze  key.Binding
	Open    key.Binding
	Detail  key.Binding
	Back    key.Binding
	Refresh key.Binding
	Scope1  key.Binding
	Scope2  key.Binding
	Scope3  key.Binding
	Help    key.Binding
	Quit    key.Binding
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Select:  key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
	Ack:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "acknowledge")),
	Resolve: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "resolve")),
	Snooze:  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "snooze")),
	Open:    key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
	Detail:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "view details")),
	Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Scope1:  key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "my incidents")),
	Scope2:  key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "team incidents")),
	Scope3:  key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "all incidents")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
