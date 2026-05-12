package tui

import (
	"context"
	"fmt"

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
		m.popups.tag.Hide()
		return m, nil

	case key.Matches(msg, keymaps.DefaultKeyMap.Select):
		id := m.popups.tag.TicketID()
		tags := m.popups.tag.CurrentTags()
		m.popups.tag.Hide()
		return m, func() tea.Msg {
			return views.TagsFilledMsg{ID: id, Tags: tags}
		}

	default:
		return m, m.popups.tag.Update(msg)
	}
}

func (m *AppModel) handleTodoKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keymaps.DefaultKeyMap.GoBack) && !m.popups.todo.IsAdding():
		m.popups.todo.Hide()
		return m, nil

	default:
		return m, m.popups.todo.Update(msg)
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
		logger.Log.Warn(fmt.Sprintf("failed to load tags: %v", err))
	}
	m.popups.tag.SetAllTags(allTags)

	return true, m.popups.tag.Show(ticket)
}

func (m *AppModel) handleToggleTodo(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !key.Matches(msg, keymaps.DefaultKeyMap.ToggleTodo) {
		return false, nil
	}

	ticket, ok := m.withSelectedTicket()
	if !ok {
		return false, nil
	}

	if m.popups.todo.IsVisible() {
		m.popups.todo.Hide()
		return true, nil
	}

	todos, err := m.store.GetTodos(ticket.ID)
	if err != nil {
		logger.Log.Warn(fmt.Sprintf("failed to load todos: %v", err))
	}
	w, h := m.styles.App.GetFrameSize()
	m.popups.todo = views.NewTodoModel(m.width-w, m.height-h, ticket.ID, todos)
	m.popups.todo.Show()

	return true, nil
}

func (m *AppModel) handleTagFilled(msg views.TagsFilledMsg) (tea.Model, tea.Cmd) {
	return m, m.saveTagsCmd(msg.ID, msg.Tags)
}

func (m *AppModel) handleTodosChanged(msg views.TodosChangedMsg) (tea.Model, tea.Cmd) {
	return m, m.saveTodosCmd(msg.TicketID, msg.Todos)
}

func (m *AppModel) handleTagSaved(msg tagSavedMsg) (tea.Model, tea.Cmd) {
	m.refreshListsFromDB()

	allTags, err := m.store.GetUniqueTags()
	if err != nil {
		logger.Log.Warn(fmt.Sprintf("failed to load tags after save: %v", err))
	}
	m.popups.tag.SetAllTags(allTags)

	if dm, ok := m.activeDetailModel(); ok {
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

	m.popups.status.Show(ticket)
	return true, m.fetchTransitionsCmd(ticket.ID)
}

func (m *AppModel) fetchTransitionsCmd(issueKey string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		transitions, err := m.jiraClient.FetchTransitions(ctx, issueKey)
		if err != nil {
			return transitionsErrorMsg{err: err}
		}
		return transitionsLoadedMsg{ticketID: issueKey, transitions: transitions}
	}
}

func (m *AppModel) handleStatusKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, keymaps.DefaultKeyMap.GoBack) {
		m.popups.status.Hide()
		return m, nil
	}

	return m, m.popups.status.Update(msg)
}

func (m *AppModel) handleTransitionsLoaded(msg transitionsLoadedMsg) (tea.Model, tea.Cmd) {
	m.popups.status.SetTransitions(msg.transitions)
	return m, nil
}

func (m *AppModel) handleStatusTransition(msg views.StatusTransitionMsg) (tea.Model, tea.Cmd) {
	return m, m.doTransitionCmd(msg.TicketID, msg.Transition)
}

func (m *AppModel) doTransitionCmd(issueKey string, transition jira.Transition) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		err := m.jiraClient.DoTransition(ctx, issueKey, transition.ID)
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

	toastCmd := m.popups.toast.Show(fmt.Sprintf("→ %s", msg.newStatus))

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
		if _, ok := root.FindTicket(ticketID); ok {
			tickets, err := m.store.GetCachedTickets()
			if err != nil {
				logger.Log.Warn(fmt.Sprintf("failed to re-read tickets for status update: %v", err))
			} else {
				update(tickets)
				root.SetItems(tickets)
			}
		}
	}

	for epicKey, children := range m.epicChildren {
		m.epicChildren[epicKey] = update(children)
	}

	m.refreshCurrentEpicView()

	// Update detail view if showing this ticket
	if dm, ok := m.activeDetailModel(); ok {
		if dm.Ticket().ID == ticketID {
			ticket := dm.Ticket()
			ticket.Status = newStatus
			ticket.StatusCategory = newStatusCategory
			dm.UpdateTags(ticket)
		}
	}
}
