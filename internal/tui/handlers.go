package tui

import (
	"fmt"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

func (m *AppModel) loadCachedTickets() tea.Cmd {
	return func() tea.Msg {
		epicChildren, _ := m.store.GetAllCachedEpicChildren()
		tickets, err := m.store.GetCachedTickets()
		if err != nil || len(tickets) == 0 {
			return cachedTicketsLoadedMsg{epicChildren: epicChildren}
		}
		return cachedTicketsLoadedMsg{tickets: tickets, epicChildren: epicChildren}
	}
}

func (m *AppModel) syncFromJira() tea.Cmd {
	return func() tea.Msg {
		tickets, err := m.jiraClient.FetchTickets()
		if err != nil {
			return syncErrorMsg{err: err}
		}
		_ = m.store.CacheTickets(tickets)

		epicChildren, _ := m.jiraClient.FetchAllEpicChildren(tickets)
		for epicKey, children := range epicChildren {
			_ = m.store.CacheEpicChildren(epicKey, children)
		}

		// Re-read from DB: tags joined, epic children excluded from main list
		mainTickets, _ := m.store.GetCachedTickets()
		allChildren, _ := m.store.GetAllCachedEpicChildren()

		return syncCompleteMsg{tickets: mainTickets, epicChildren: allChildren}
	}
}

func (m *AppModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.list.SetSize(msg.Width, msg.Height)

	if m.previousList != nil {
		m.previousList.SetSize(msg.Width, msg.Height)
	}

	m.debugModel.SetSize(msg.Width, msg.Height)

	m.width = msg.Width
	m.height = msg.Height

	w, h := m.styles.App.GetFrameSize()
	contentWidth := msg.Width - w
	contentHeight := msg.Height - h

	m.tagModel.SetSize(contentWidth, contentHeight)
	m.todoModel.SetSize(contentWidth, contentHeight)
	m.statusModel.SetSize(contentWidth, contentHeight)
	m.toastModel.SetSize(msg.Width, msg.Height)

	return m, nil
}

// rootList returns the main ticket list, even when navigated into an epic.
func (m *AppModel) rootList() *views.ListModel {
	if m.previousList != nil {
		return m.previousList
	}
	return m.list
}

func (m *AppModel) handleCachedTicketsLoaded(msg cachedTicketsLoadedMsg) (tea.Model, tea.Cmd) {
	if m.synced {
		return m, nil
	}

	if msg.epicChildren != nil {
		m.epicChildren = msg.epicChildren
	}

	if len(msg.tickets) > 0 {
		m.list.Initialize(msg.tickets)
		m.list.SetTitle("Jira Tickets (syncing...)")
	}

	return m, nil
}

func (m *AppModel) handleSyncComplete(msg syncCompleteMsg) (tea.Model, tea.Cmd) {
	m.synced = true
	m.syncing = false

	m.epicChildren = msg.epicChildren

	root := m.rootList()
	root.SetItems(msg.tickets)
	root.StopSpinner()
	root.SetTitle("Jira Tickets")

	return m, nil
}

func (m *AppModel) handleSyncError(msg syncErrorMsg) (tea.Model, tea.Cmd) {
	m.syncing = false

	root := m.rootList()
	root.SetTitle("Jira Tickets")

	if root.HasTickets() {
		return m, nil
	}

	m.err = views.ErrMsg{Err: msg.err}

	root.StopSpinner()

	return m, nil
}

func (m *AppModel) handleError(msg views.ErrMsg) (tea.Model, tea.Cmd) {
	m.err = msg
	m.list.StopSpinner()
	return m, nil
}

func (m *AppModel) handleSelectTicket(msg views.SelectTicketMsg) (tea.Model, tea.Cmd) {
	if msg.Ticket.IsEpic() {
		if children, ok := m.epicChildren[msg.Ticket.ID]; ok {
			return m.showEpicChildren(msg.Ticket.ID, children)
		}
		return m, tea.Batch(m.list.StartSpinner(), m.fetchEpicChildrenCmd(msg.Ticket.ID))
	}

	m.activeModel = views.NewDetailModel(msg.Ticket, m.width, m.height, m.styles)
	return m, nil
}

func (m *AppModel) showEpicChildren(epicKey string, tickets []model.Ticket) (tea.Model, tea.Cmd) {
	epicList := views.NewListModel(tickets, m.styles.App)
	epicList.SetSize(m.width, m.height)

	title := fmt.Sprintf("⚡ %s", epicKey)
	if epic, ok := m.rootList().FindTicket(epicKey); ok {
		title = fmt.Sprintf("⚡ %s — %s", epicKey, epic.Summary)
	}
	epicList.SetTitle(title)

	m.previousList = m.list
	m.list = epicList
	m.activeModel = epicList
	return m, nil
}

func (m *AppModel) fetchEpicChildrenCmd(epicKey string) tea.Cmd {
	return func() tea.Msg {
		tickets, err := m.jiraClient.FetchEpicChildren(epicKey)
		if err != nil {
			return epicChildrenErrorMsg{err: err}
		}
		return epicChildrenLoadedMsg{epicKey: epicKey, tickets: tickets}
	}
}

func (m *AppModel) handleEpicChildrenLoaded(msg epicChildrenLoadedMsg) (tea.Model, tea.Cmd) {
	_ = m.store.CacheEpicChildren(msg.epicKey, msg.tickets)
	// Re-read from DB to get tags joined in
	allChildren, _ := m.store.GetAllCachedEpicChildren()
	children := allChildren[msg.epicKey]
	m.epicChildren[msg.epicKey] = children
	m.list.StopSpinner()
	return m.showEpicChildren(msg.epicKey, children)
}

func (m *AppModel) handleExitEpic(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !key.Matches(msg, keymaps.DefaultKeyMap.GoBack) {
		return false, nil
	}

	if m.previousList == nil {
		return false, nil
	}

	// Only exit epic when on the epic list itself, not from a detail view
	if _, onList := m.activeModel.(*views.ListModel); !onList {
		return false, nil
	}

	if m.list.IsFiltering() {
		return false, nil
	}

	m.restoreRootList()
	return true, nil
}

func (m *AppModel) handleGoToList(_ views.GoToListMsg) (tea.Model, tea.Cmd) {
	m.activeModel = m.list
	return m, nil
}

func (m *AppModel) handleGoHome(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !key.Matches(msg, keymaps.DefaultKeyMap.GoHome) {
		return false, nil
	}
	if m.isPopupActive() || m.list.IsFiltering() {
		return false, nil
	}
	m.restoreRootList()
	return true, nil
}

// restoreRootList navigates back to the main ticket list, discarding epic navigation.
func (m *AppModel) restoreRootList() {
	if m.previousList != nil {
		m.list = m.previousList
		m.previousList = nil
	}
	m.activeModel = m.list
}

func (m *AppModel) handleTagFilled(msg views.TagsFilledMsg) (tea.Model, tea.Cmd) {
	return m, m.saveTagsCmd(msg.ID, msg.Tags)
}

func (m *AppModel) handleTodosChanged(msg views.TodosChangedMsg) (tea.Model, tea.Cmd) {
	return m, m.saveTodosCmd(msg.TicketID, msg.Todos)
}

func (m *AppModel) handleTagSaved(msg tagSavedMsg) (tea.Model, tea.Cmd) {
	// Re-read tickets and epic children with fresh tags from DB
	tickets, _ := m.store.GetCachedTickets()
	m.rootList().SetItems(tickets)

	epicChildren, _ := m.store.GetAllCachedEpicChildren()
	m.epicChildren = epicChildren

	// Refresh current epic view if inside one
	if m.previousList != nil {
		for epicKey, children := range epicChildren {
			if strings.HasPrefix(m.list.Title(), fmt.Sprintf("⚡ %s", epicKey)) {
				m.list.SetItems(children)
				break
			}
		}
	}

	allTags, _ := m.store.GetUniqueTags()
	m.tagModel.SetAllTags(allTags)

	if dm, ok := m.activeModel.(*views.DetailModel); ok {
		if ticket, ok := m.findTicket(msg.id); ok {
			dm.UpdateTags(ticket)
		}
	}

	return m, nil
}

func (m *AppModel) findTicket(id string) (model.Ticket, bool) {
	if ticket, ok := m.rootList().FindTicket(id); ok {
		return ticket, true
	}

	for _, children := range m.epicChildren {
		for _, t := range children {
			if t.ID == id {
				return t, true
			}
		}
	}

	return model.Ticket{}, false
}

func (m *AppModel) handleQuit(msg tea.KeyPressMsg) tea.Cmd {
	if key.Matches(msg, keymaps.DefaultKeyMap.ForceQuit) ||
		(key.Matches(msg, keymaps.DefaultKeyMap.Quit) && !m.isPopupActive()) {
		return tea.Quit
	}
	return nil
}

func (m *AppModel) handleDebug(msg tea.KeyPressMsg) (bool, tea.Cmd) {
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

func (m *AppModel) handleRefresh(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if key.Matches(msg, keymaps.DefaultKeyMap.Refresh) &&
		!m.isPopupActive() &&
		!m.syncing &&
		!m.list.IsFiltering() {
		m.syncing = true
		root := m.rootList()
		root.SetTitle("Jira Tickets (syncing...)")
		return true, tea.Batch(root.StartSpinner(), m.syncFromJira())
	}

	return false, nil
}

func (m *AppModel) isPopupActive() bool {
	return m.tagModel.IsVisible() || m.todoModel.IsVisible() || m.statusModel.IsVisible()
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

// withSelectedTicket guards actions that require a selected ticket and no active filtering/popup.
func (m *AppModel) withSelectedTicket() (model.Ticket, bool) {
	if m.isPopupActive() || m.list.IsFiltering() {
		return model.Ticket{}, false
	}
	return m.selectedTicket()
}

func (m *AppModel) handleToggleTag(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !key.Matches(msg, keymaps.DefaultKeyMap.ToggleTagging) {
		return false, nil
	}

	ticket, ok := m.withSelectedTicket()
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

	ticket, ok := m.withSelectedTicket()
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

	ticket, ok := m.withSelectedTicket()
	if !ok {
		return false, nil
	}

	domain := strings.TrimRight(m.config.Domain, "/")
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

func (m *AppModel) handleToggleStatus(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !key.Matches(msg, keymaps.DefaultKeyMap.ToggleStatus) {
		return false, nil
	}

	ticket, ok := m.withSelectedTicket()
	if !ok {
		return false, nil
	}

	m.statusModel.Show(ticket)
	return true, m.fetchTransitionsCmd(ticket.ID)
}

func (m *AppModel) fetchTransitionsCmd(issueKey string) tea.Cmd {
	return func() tea.Msg {
		transitions, err := m.jiraClient.FetchTransitions(issueKey)
		if err != nil {
			return transitionsErrorMsg{err: err}
		}
		return transitionsLoadedMsg{ticketID: issueKey, transitions: transitions}
	}
}

func (m *AppModel) handleStatusKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, keymaps.DefaultKeyMap.GoBack) {
		m.statusModel.Hide()
		return m, nil
	}

	return m, m.statusModel.Update(msg)
}

func (m *AppModel) handleTransitionsLoaded(msg transitionsLoadedMsg) (tea.Model, tea.Cmd) {
	m.statusModel.SetTransitions(msg.transitions)
	return m, nil
}

func (m *AppModel) handleStatusTransition(msg views.StatusTransitionMsg) (tea.Model, tea.Cmd) {
	return m, m.doTransitionCmd(msg.TicketID, msg.Transition)
}

func (m *AppModel) doTransitionCmd(issueKey string, transition jira.Transition) tea.Cmd {
	return func() tea.Msg {
		err := m.jiraClient.DoTransition(issueKey, transition.ID)
		if err != nil {
			return statusTransitionErrorMsg{err: err}
		}
		return statusTransitionCompleteMsg{
			ticketID:          issueKey,
			newStatus:         transition.ToStatus,
			newStatusCategory: transition.ToStatusCategory,
		}
	}
}

func (m *AppModel) handleStatusTransitionComplete(msg statusTransitionCompleteMsg) (tea.Model, tea.Cmd) {
	m.updateTicketStatus(msg.ticketID, msg.newStatus, msg.newStatusCategory)

	toastCmd := m.toastModel.Show(fmt.Sprintf("→ %s", msg.newStatus))

	return m, tea.Batch(toastCmd, m.syncFromJira())
}

// updateTicketStatus updates ticket status in-memory across main list, epic children, and detail view.
func (m *AppModel) updateTicketStatus(ticketID, newStatus, newStatusCategory string) {
	update := func(tickets []model.Ticket) []model.Ticket {
		for i := range tickets {
			if tickets[i].ID == ticketID {
				tickets[i].Status = newStatus
				tickets[i].StatusCategory = newStatusCategory
			}
		}
		return tickets
	}

	if root := m.rootList(); root != nil {
		if t, ok := root.FindTicket(ticketID); ok {
			t.Status = newStatus
			t.StatusCategory = newStatusCategory
			tickets, _ := m.store.GetCachedTickets()
			update(tickets)
			root.SetItems(tickets)
		}
	}

	for epicKey, children := range m.epicChildren {
		m.epicChildren[epicKey] = update(children)
	}

	// Update current epic list view if inside one
	if m.previousList != nil {
		for epicKey, children := range m.epicChildren {
			if strings.HasPrefix(m.list.Title(), fmt.Sprintf("⚡ %s", epicKey)) {
				m.list.SetItems(children)
				break
			}
		}
	}

	// Update detail view if showing this ticket
	if dm, ok := m.activeModel.(*views.DetailModel); ok {
		if dm.Ticket().ID == ticketID {
			ticket := dm.Ticket()
			ticket.Status = newStatus
			ticket.StatusCategory = newStatusCategory
			dm.UpdateTags(ticket)
		}
	}
}
