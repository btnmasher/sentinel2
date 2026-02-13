package headless

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	nextFocus   key.Binding
	prevFocus   key.Binding
	prevTab     key.Binding
	nextTab     key.Binding
	activate    key.Binding
	quit        key.Binding
	modalToggle key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		nextFocus: key.NewBinding(
			key.WithKeys("tab", "down"),
			key.WithHelp("tab/down", "next"),
		),
		prevFocus: key.NewBinding(
			key.WithKeys("shift+tab", "up"),
			key.WithHelp("shift+tab/up", "prev"),
		),
		prevTab: key.NewBinding(
			key.WithKeys("ctrl+left"),
			key.WithHelp("ctrl+left", "overview"),
		),
		nextTab: key.NewBinding(
			key.WithKeys("ctrl+right"),
			key.WithHelp("ctrl+right", "settings"),
		),
		activate: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "activate"),
		),
		quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		modalToggle: key.NewBinding(
			key.WithKeys("tab", "up", "down", "left", "right"),
			key.WithHelp("tab/arrows", "toggle"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.nextFocus, k.activate, k.prevTab, k.nextTab, k.quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.nextFocus, k.prevFocus, k.activate},
		{k.prevTab, k.nextTab, k.quit},
	}
}
