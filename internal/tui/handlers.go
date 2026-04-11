package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

func (m *AppModel) fetchTickets() tea.Cmd {
	return func() tea.Msg {
		tickets, err := m.jiraClient.FetchTickets()
		if err != nil {
			return views.ErrMsg{Err: err}
		}

		localData, err := m.store.GetAllMeta()
		if err != nil {
			return views.ErrMsg{Err: err}
		}

		for i, t := range tickets {
			meta, exists := localData[t.ID]
			if exists {
				tickets[i].Tags = meta.Tags
			}
		}

		return ticketsFetchedMsg(tickets)
	}
}

func (m *AppModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.list.SetSize(msg.Width, msg.Height)
	m.debugModel.SetSize(msg.Width, msg.Height)

	m.width = msg.Width
	m.height = msg.Height

	return m, nil
}

func (m *AppModel) handleTicketsFetched(msg ticketsFetchedMsg) (tea.Model, tea.Cmd) {
	m.list.Initialize(msg)
	return m, nil
}

func (m *AppModel) handleError(msg views.ErrMsg) (tea.Model, tea.Cmd) {
	m.err = msg
	m.list.StopSpinner()
	return m, nil
}

func (m *AppModel) handleSelectTicket(msg views.SelectTicketMsg) (tea.Model, tea.Cmd) {
	m.activeModel = views.NewDetailModel(msg.Ticket, m.width, m.height, m.styles, m.list.AllTags())
	return m, nil
}

func (m *AppModel) handleGoToList(_ views.GoToListMsg) (tea.Model, tea.Cmd) {
	m.activeModel = m.list
	return m, nil
}

func (m *AppModel) handleTagFilled(msg views.TagsFilledMsg) (tea.Model, tea.Cmd) {
	return m, m.saveTagsCmd(msg.ID, msg.Tags)
}

func (m *AppModel) handleTagSaved(msg tagSavedMsg) (tea.Model, tea.Cmd) {
	ticket, err := m.list.UpdateTicket(msg.id, msg.tags)
	if err != nil {
		return m, func() tea.Msg {
			return views.ErrMsg{Err: err}
		}
	}
	return m, func() tea.Msg {
		return views.SelectTicketMsg{Ticket: *ticket}
	}
}

func handleQuit(m *AppModel, msg tea.KeyPressMsg) tea.Cmd {
	s := msg.String()

	if s == "ctrl+c" {
		return tea.Quit
	}

	if s == "q" && !m.isTagging() {
		return tea.Quit
	}

	if m.activeModel == nil {
		return nil
	}

	return nil
}

func handleDebug(m *AppModel, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if msg.String() == "d" && !m.isTagging() {
		m.debugModel.Toggle()
		return true, nil
	}

	isVisible := m.debugModel.IsVisible()

	if msg.String() == "esc" && isVisible {
		m.debugModel.Toggle()
		return true, nil
	}

	if isVisible {
		return true, m.debugModel.Update(msg)
	}

	return false, nil
}

func (m *AppModel) isTagging() bool {
	if dm, ok := m.activeModel.(*views.DetailModel); ok {
		return dm.IsTagging()
	}
	return false
}

func (m *AppModel) saveTagsCmd(id string, tags []string) tea.Cmd {
	return func() tea.Msg {
		err := m.store.SaveMeta(id, tags, "")
		if err != nil {
			return views.ErrMsg{Err: err}
		}

		return tagSavedMsg{id: id, tags: tags}
	}
}
