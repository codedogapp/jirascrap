package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

// keyHandler checks whether a key press is consumed and optionally returns a command.
type keyHandler func(tea.KeyPressMsg) (bool, tea.Cmd)

func (m *AppModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.isCommentInputActive() {
		var cmd tea.Cmd
		m.activeModel, cmd = m.activeModel.Update(msg)
		return m, cmd
	}

	if cmd := m.handleQuit(msg); cmd != nil {
		return m, cmd
	}
	if consumed, cmd := m.popups.RouteKeyPress(msg); consumed {
		return m, cmd
	}

	for _, h := range m.globalKeyHandlers() {
		if consumed, cmd := h(msg); consumed {
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.activeModel, cmd = m.activeModel.Update(msg)
	return m, cmd
}

func (m *AppModel) globalKeyHandlers() []keyHandler {
	return []keyHandler{
		m.handleRefresh,
		m.handleExitEpic,
		m.handleGoHome,
		m.handleOpenInBrowser,
		m.handleToggleTag,
		m.handleToggleTodo,
		m.handleToggleStatus,
		m.handleSendToCopilot,
	}
}

func (m *AppModel) handleOtherMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	if handled, cmd := m.popups.RouteMsg(msg); handled {
		return m, cmd
	}
	if mu, ok := m.activeModel.(views.MsgUpdater); ok {
		return m, mu.UpdateMsg(msg)
	}
	return m, nil
}

func (m *AppModel) isCommentInputActive() bool {
	dm, ok := m.activeDetailModel()
	return ok && dm.CommentInput().Visible()
}
