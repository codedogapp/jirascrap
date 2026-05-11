package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/logger"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

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

	ticket, ok := m.withSelectedTicket()
	if !ok {
		return false, nil
	}

	allTags, err := m.store.GetUniqueTags()
	if err != nil {
		logger.Log.Warn("failed to load tags: " + err.Error())
	}
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

	todos, err := m.store.GetTodos(ticket.ID)
	if err != nil {
		logger.Log.Warn("failed to load todos: " + err.Error())
	}
	w, h := m.styles.App.GetFrameSize()
	m.todoModel = views.NewTodoModel(m.width-w, m.height-h, ticket.ID, todos)
	m.todoModel.Show()

	return true, nil
}

func (m *AppModel) handleTagFilled(msg views.TagsFilledMsg) (tea.Model, tea.Cmd) {
	return m, m.saveTagsCmd(msg.ID, msg.Tags)
}

func (m *AppModel) handleTodosChanged(msg views.TodosChangedMsg) (tea.Model, tea.Cmd) {
	return m, m.saveTodosCmd(msg.TicketID, msg.Todos)
}

func (m *AppModel) handleTagSaved(msg tagSavedMsg) (tea.Model, tea.Cmd) {
	// Re-read tickets and epic children with fresh tags from DB
	tickets, err := m.store.GetCachedTickets()
	if err != nil {
		logger.Log.Warn("failed to re-read tickets after tag save: " + err.Error())
	} else {
		m.rootList().SetItems(tickets)
	}

	epicChildren, err := m.store.GetAllCachedEpicChildren()
	if err != nil {
		logger.Log.Warn("failed to re-read epic children after tag save: " + err.Error())
	} else {
		m.epicChildren = epicChildren
	}

	// Refresh current epic view if inside one
	if m.previousList != nil {
		for epicKey, children := range epicChildren {
			if strings.HasPrefix(m.list.Title(), fmt.Sprintf("⚡ %s", epicKey)) {
				m.list.SetItems(children)
				break
			}
		}
	}

	allTags, err2 := m.store.GetUniqueTags()
	if err2 != nil {
		logger.Log.Warn("failed to load tags after save: " + err2.Error())
	}
	m.tagModel.SetAllTags(allTags)

	if dm, ok := m.activeModel.(*views.DetailModel); ok {
		if ticket, ok := m.findTicket(msg.id); ok {
			dm.UpdateTags(ticket)
		}
	}

	return m, nil
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
		transitions, err := m.jiraClient.FetchTransitions(context.Background(), issueKey)
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
		err := m.jiraClient.DoTransition(context.Background(), issueKey, transition.ID)
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
			tickets, err := m.store.GetCachedTickets()
			if err != nil {
				logger.Log.Warn("failed to re-read tickets for status update: " + err.Error())
			} else {
				update(tickets)
				root.SetItems(tickets)
			}
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
