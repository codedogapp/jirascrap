package views

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type ActiveModel interface {
	Update(tea.KeyPressMsg) (ActiveModel, tea.Cmd)
	View() tea.View
}

type MsgUpdater interface {
	UpdateMsg(tea.Msg) tea.Cmd
}

// popupState provides shared visibility management for popup models.
// Embed in popup structs to get Hide() and IsVisible() for free.
// Override Hide() if cleanup beyond visibility is needed.
type popupState struct {
	visible bool
}

func (p *popupState) Hide() {
	p.visible = false
}

func (p *popupState) IsVisible() bool {
	return p.visible
}

// baseDelegate provides shared Height/Spacing/Update for list delegates.
// Embed and set height/spacing fields. Only Render() needs implementing.
type baseDelegate struct {
	height  int
	spacing int
}

func (d baseDelegate) Height() int {
	return d.height
}

func (d baseDelegate) Spacing() int {
	return d.spacing
}

func (d baseDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

type ErrMsg struct{ Err error }

func (e ErrMsg) Error() string {
	if e.Err == nil {
		return "unknown error"
	}
	return e.Err.Error()
}
