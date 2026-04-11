package keymaps

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
)

type KeyMap struct {
	ForceQuit      key.Binding
	Quit           key.Binding
	GoBack         key.Binding
	Select         key.Binding
	ToggleTagging  key.Binding
	ToggleDebug    key.Binding
	ToggleHelp     key.Binding
	Viewport       viewport.KeyMap
	fullHelpHeight int
}

var DefaultKeyMap = newKeyMap()

func newKeyMap() *KeyMap {
	k := &KeyMap{
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),

		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),

		GoBack: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),

		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "Select"),
		),

		ToggleTagging: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "tag"),
		),

		ToggleDebug: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "debug"),
		),

		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "more"),
		),

		Viewport: viewport.DefaultKeyMap(),
	}
	k.fullHelpHeight = k.computeFullHelpHeight()
	return k
}

func (k *KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Viewport.PageUp,
		k.Viewport.PageDown,
		k.ToggleTagging,
		k.ToggleHelp,
	}
}

func (k *KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Viewport.PageUp, k.Viewport.Up},
		{k.Viewport.PageDown, k.Viewport.Down},
		{k.ToggleTagging, k.GoBack},
		{k.Quit, k.ToggleDebug},
	}
}

func (k *KeyMap) GetFullHelpHeight() int {
	return k.fullHelpHeight
}

// computeFullHelpHeight returns full help height relative to short help
func (k *KeyMap) computeFullHelpHeight() int {
	max := 0
	for _, col := range k.FullHelp() {
		if len(col) > max {
			max = len(col)
		}
	}
	return max - 1
}
