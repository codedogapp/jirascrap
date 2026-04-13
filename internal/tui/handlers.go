package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

func (m *AppModel) loadCachedTickets() tea.Cmd {
	return func() tea.Msg {
		tickets, err := m.store.GetCachedTickets()
		if err != nil || len(tickets) == 0 {
			return cachedTicketsLoadedMsg(nil)
		}
		m.mergeLocalMeta(tickets)
		return cachedTicketsLoadedMsg(tickets)
	}
}

func (m *AppModel) syncFromJira() tea.Cmd {
	return func() tea.Msg {
		tickets, err := m.jiraClient.FetchTickets()
		if err != nil {
			return syncErrorMsg{err: err}
		}
		_ = m.store.CacheTickets(tickets)
		m.mergeLocalMeta(tickets)
		return syncCompleteMsg(tickets)
	}
}

func (m *AppModel) mergeLocalMeta(tickets []model.Ticket) {
	localData, err := m.store.GetAllMeta()
	if err != nil {
		return
	}
	for i, t := range tickets {
		if meta, ok := localData[t.ID]; ok {
			tickets[i].Tags = meta.Tags
		}
	}
}

func (m *AppModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.list.SetSize(msg.Width, msg.Height)
	m.debugModel.SetSize(msg.Width, msg.Height)

	m.width = msg.Width
	m.height = msg.Height

	return m, nil
}

func (m *AppModel) handleCachedTicketsLoaded(msg cachedTicketsLoadedMsg) (tea.Model, tea.Cmd) {
	if m.synced {
		return m, nil
	}
	tickets := []model.Ticket(msg)
	if len(tickets) > 0 {
		m.list.Initialize(tickets)
		m.list.SetTitle("Jira Tickets (syncing...)")
	}
	return m, nil
}

func (m *AppModel) handleSyncComplete(msg syncCompleteMsg) (tea.Model, tea.Cmd) {
	m.synced = true
	m.syncing = false
	m.list.SetItems([]model.Ticket(msg))
	m.list.StopSpinner()
	m.list.SetTitle("Jira Tickets")
	return m, nil
}

func (m *AppModel) handleSyncError(msg syncErrorMsg) (tea.Model, tea.Cmd) {
	m.syncing = false
	m.list.SetTitle("Jira Tickets")
	if m.list.HasTickets() {
		return m, nil
	}
	m.err = views.ErrMsg{Err: msg.err}
	m.list.StopSpinner()
	return m, nil
}

func (m *AppModel) handleError(msg views.ErrMsg) (tea.Model, tea.Cmd) {
	m.err = msg
	m.list.StopSpinner()
	return m, nil
}

func (m *AppModel) handleSelectTicket(msg views.SelectTicketMsg) (tea.Model, tea.Cmd) {
	todos, err := m.store.GetTodos(msg.Ticket.ID)
	if err != nil {
		todos = nil
	}
	allTags, err := m.store.GetUniqueTags()
	if err != nil {
		allTags = nil
	}
	m.activeModel = views.NewDetailModel(msg.Ticket, m.width, m.height, m.styles, allTags, todos)
	return m, nil
}

func (m *AppModel) handleGoToList(_ views.GoToListMsg) (tea.Model, tea.Cmd) {
	m.activeModel = m.list
	return m, nil
}

func (m *AppModel) handleTagFilled(msg views.TagsFilledMsg) (tea.Model, tea.Cmd) {
	return m, m.saveTagsCmd(msg.ID, msg.Tags)
}

func (m *AppModel) handleTodosChanged(msg views.TodosChangedMsg) (tea.Model, tea.Cmd) {
	return m, m.saveTodosCmd(msg.TicketID, msg.Todos)
}

func (m *AppModel) handleTagSaved(msg tagSavedMsg) (tea.Model, tea.Cmd) {
	ticket, err := m.list.UpdateTicket(msg.id, msg.tags)
	if err != nil {
		return m, func() tea.Msg {
			return views.ErrMsg{Err: err}
		}
	}
	if dm, ok := m.activeModel.(*views.DetailModel); ok {
		allTags, err := m.store.GetUniqueTags()
		if err != nil {
			allTags = nil
		}
		dm.UpdateTags(*ticket, allTags)
	}
	return m, nil
}

func handleQuit(m *AppModel, msg tea.KeyPressMsg) tea.Cmd {
	if key.Matches(msg, keymaps.DefaultKeyMap.ForceQuit) {
		return tea.Quit
	}

	if key.Matches(msg, keymaps.DefaultKeyMap.Quit) && !m.isPopupActive() {
		return tea.Quit
	}

	return nil
}

func handleDebug(m *AppModel, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if key.Matches(msg, keymaps.DefaultKeyMap.ToggleDebug) && !m.isPopupActive() {
		if m.debugModel.IsVisible() {
			m.debugModel.Hide()
		} else {
			m.debugModel.Show()
		}
		return true, nil
	}

	isVisible := m.debugModel.IsVisible()

	if key.Matches(msg, keymaps.DefaultKeyMap.GoBack) && isVisible {
		m.debugModel.Hide()
		return true, nil
	}

	if isVisible {
		return true, m.debugModel.Update(msg)
	}

	return false, nil
}

func handleRefresh(m *AppModel, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if key.Matches(msg, keymaps.DefaultKeyMap.Refresh) && !m.isPopupActive() && !m.syncing {
		m.syncing = true
		m.list.SetTitle("Jira Tickets (syncing...)")
		return true, m.syncFromJira()
	}
	return false, nil
}

func (m *AppModel) isPopupActive() bool {
	if dm, ok := m.activeModel.(*views.DetailModel); ok {
		return dm.IsTagging() || dm.IsTodoing()
	}
	return false
}

func (m *AppModel) saveTagsCmd(id string, tags []string) tea.Cmd {
	return func() tea.Msg {
		err := m.store.SaveMeta(id, tags)
		if err != nil {
			return views.ErrMsg{Err: err}
		}

		return tagSavedMsg{id: id, tags: tags}
	}
}

func (m *AppModel) saveTodosCmd(ticketID string, todos []model.Todo) tea.Cmd {
	return func() tea.Msg {
		err := m.store.SaveTodos(ticketID, todos)
		if err != nil {
			return views.ErrMsg{Err: err}
		}
		return todoSavedMsg{}
	}
}
