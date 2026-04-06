package views

import tea "charm.land/bubbletea/v2"

type ActiveModel interface {
	Update(tea.KeyPressMsg) (ActiveModel, tea.Cmd)
	View() tea.View
}

type MsgUpdater interface {
	UpdateMsg(tea.Msg) tea.Cmd
}

type ErrMsg struct{ Err error }

func (e ErrMsg) Error() string {
	return e.Err.Error()
}
