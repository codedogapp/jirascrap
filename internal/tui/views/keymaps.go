package views

import "charm.land/bubbles/v2/key"

type ListKeyMap struct {
	ForceQuit     key.Binding
	GoBack        key.Binding
	ToggleTagging key.Binding
}

func NewListKeyMap() *ListKeyMap {
	return &ListKeyMap{
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
		GoBack: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "go back"),
		),
		ToggleTagging: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle tag"),
		),
	}
}
