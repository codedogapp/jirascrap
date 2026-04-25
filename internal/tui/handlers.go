package tui

import (
	"fmt"
	"os/exec"
	"strings"

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

	w, h := m.styles.App.GetFrameSize()
	contentWidth := msg.Width - w
	contentHeight := msg.Height - h
	m.tagModel.SetSize(contentWidth, contentHeight)
	m.todoModel.SetSize(contentWidth, contentHeight)

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
	m.activeModel = views.NewDetailModel(msg.Ticket, m.width, m.height, m.styles)
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
	allTags, _ := m.store.GetUniqueTags()
	m.tagModel.SetAllTags(allTags)
	if dm, ok := m.activeModel.(*views.DetailModel); ok {
		dm.UpdateTags(*ticket)
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
	if key.Matches(msg, keymaps.DefaultKeyMap.Refresh) && !m.isPopupActive() && !m.syncing && !m.list.IsFiltering() {
		m.syncing = true
		m.list.SetTitle("Jira Tickets (syncing...)")
		return true, m.syncFromJira()
	}
	return false, nil
}

func (m *AppModel) isPopupActive() bool {
	return m.tagModel.IsVisible() || m.todoModel.IsVisible()
}

func (m *AppModel) selectedTicket() (model.Ticket, bool) {
	switch v := m.activeModel.(type) {
	case *views.ListModel:
		return v.SelectedTicket()
	case *views.DetailModel:
		return v.Ticket(), true
	}
	return model.Ticket{}, false
}

func (m *AppModel) handleTagKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keymaps.DefaultKeyMap.GoBack):
		m.tagModel.Hide()
		return m, nil
	case key.Matches(msg, keymaps.DefaultKeyMap.Select):
		id := m.tagModel.TicketID()
		tags := m.tagModel.CurrentTags()
		m.tagModel.Hide()
		return m, func() tea.Msg {
			return views.TagsFilledMsg{ID: id, Tags: tags}
		}
	default:
		return m, m.tagModel.Update(msg)
	}
}

func (m *AppModel) handleTodoKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keymaps.DefaultKeyMap.GoBack) && !m.todoModel.IsAdding():
		m.todoModel.Hide()
		return m, nil
	default:
		return m, m.todoModel.Update(msg)
	}
}

func (m *AppModel) handleToggleTag(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !key.Matches(msg, keymaps.DefaultKeyMap.ToggleTagging) {
		return false, nil
	}
	if m.list.IsFiltering() {
		return false, nil
	}
	ticket, ok := m.selectedTicket()
	if !ok {
		return false, nil
	}
	allTags, _ := m.store.GetUniqueTags()
	m.tagModel.SetAllTags(allTags)
	return true, m.tagModel.Show(ticket)
}

func (m *AppModel) handleToggleTodo(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !key.Matches(msg, keymaps.DefaultKeyMap.ToggleTodo) {
		return false, nil
	}
	if m.list.IsFiltering() {
		return false, nil
	}
	ticket, ok := m.selectedTicket()
	if !ok {
		return false, nil
	}
	if m.todoModel.IsVisible() {
		m.todoModel.Hide()
		return true, nil
	}
	todos, _ := m.store.GetTodos(ticket.ID)
	w, h := m.styles.App.GetFrameSize()
	m.todoModel = views.NewTodoModel(m.width-w, m.height-h, ticket.ID, todos)
	m.todoModel.Show()
	return true, nil
}

func (m *AppModel) handleOpenInBrowser(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !key.Matches(msg, keymaps.DefaultKeyMap.OpenInBrowser) {
		return false, nil
	}
	if m.list.IsFiltering() {
		return false, nil
	}
	ticket, ok := m.selectedTicket()
	if !ok {
		return false, nil
	}
	domain := strings.TrimRight(m.domain, "/")
	ticketURL := fmt.Sprintf("%s/browse/%s", domain, ticket.ID)
	return true, func() tea.Msg {
		_ = exec.Command("open", ticketURL).Start()
		return nil
	}
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
