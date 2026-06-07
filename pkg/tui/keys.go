package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Back      key.Binding
	New       key.Binding
	Edit      key.Binding
	Delete    key.Binding
	Export    key.Binding
	SelectDir key.Binding
	Mail      key.Binding
	Quit      key.Binding
	ForceQuit key.Binding
	Confirm   key.Binding
	Cancel    key.Binding
	Tab       key.Binding
	ShiftTab  key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Back:     key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
	New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Export:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "export")),
	SelectDir: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "export here")),
	Mail:      key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "email")),
	Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	ForceQuit: key.NewBinding(key.WithKeys("ctrl+c")),
	Confirm:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	Cancel:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
	ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev field")),
}
