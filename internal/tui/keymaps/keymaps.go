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
	ToggleTodo     key.Binding
	ToggleDebug    key.Binding
	Refresh        key.Binding
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
			key.WithHelp("enter", "select"),
		),

		ToggleTagging: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "tag"),
		),

		ToggleTodo: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "todo"),
		),

		ToggleDebug: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "debug"),
		),

		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "more"),
		),

		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
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
		k.ToggleTodo,
		k.ToggleHelp,
	}
}

func (k *KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Viewport.PageUp, k.Viewport.Up},
		{k.Viewport.PageDown, k.Viewport.Down},
		{k.ToggleTagging, k.ToggleTodo},
		{k.GoBack, k.Quit, k.ToggleDebug, k.Refresh},
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

type TagKeyMap struct {
	Autocomplete   key.Binding
	NextSuggestion key.Binding
	PrevSuggestion key.Binding
}

var DefaultTagKeyMap = newTagKeyMap()

func newTagKeyMap() *TagKeyMap {
	return &TagKeyMap{
		Autocomplete: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "autocomplete"),
		),
		NextSuggestion: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "next"),
		),
		PrevSuggestion: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "prev"),
		),
	}
}

type TodoKeyMap struct {
	Add     key.Binding
	Toggle  key.Binding
	Delete  key.Binding
	Confirm key.Binding
	Cancel  key.Binding
}

var DefaultTodoKeyMap = newTodoKeyMap()

func newTodoKeyMap() *TodoKeyMap {
	return &TodoKeyMap{
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "enter"),
			key.WithHelp("space", "toggle"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
	}
}

func (k *TodoKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Add, k.Toggle, k.Delete, k.Cancel}
}

func (k *TodoKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}
