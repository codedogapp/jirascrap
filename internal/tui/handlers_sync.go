package tui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/logger"
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

func (m *AppModel) loadCachedTickets() tea.Cmd {
	return func() tea.Msg {
		epicChildren, err := m.store.GetAllCachedEpicChildren()
		if err != nil {
			logger.Log.Warn("failed to load cached epic children: " + err.Error())
		}
		tickets, err := m.store.GetCachedTickets()
		if err != nil {
			logger.Log.Warn("failed to load cached tickets: " + err.Error())
			return cachedTicketsLoadedMsg{epicChildren: epicChildren}
		}
		if len(tickets) == 0 {
			return cachedTicketsLoadedMsg{epicChildren: epicChildren}
		}
		return cachedTicketsLoadedMsg{tickets: tickets, epicChildren: epicChildren}
	}
}

func (m *AppModel) syncFromJira() tea.Cmd {
	return func() tea.Msg {
		tickets, err := m.jiraClient.FetchTickets(context.Background())
		if err != nil {
			return syncErrorMsg{err: err}
		}
		if err := m.store.CacheTickets(tickets); err != nil {
			logger.Log.Warn("failed to cache tickets: " + err.Error())
		}

		epicChildren, err := m.jiraClient.FetchAllEpicChildren(context.Background(), tickets)
		if err != nil {
			logger.Log.Warn("failed to fetch some epic children: " + err.Error())
		}
		for epicKey, children := range epicChildren {
			if err := m.store.CacheEpicChildren(epicKey, children); err != nil {
				logger.Log.Warn("failed to cache epic children for " + epicKey + ": " + err.Error())
			}
		}

		// Re-read from DB: tags joined, epic children excluded from main list
		mainTickets, err := m.store.GetCachedTickets()
		if err != nil {
			logger.Log.Warn("failed to re-read cached tickets: " + err.Error())
			mainTickets = tickets // fall back to API data
		}
		allChildren, err := m.store.GetAllCachedEpicChildren()
		if err != nil {
			logger.Log.Warn("failed to re-read epic children: " + err.Error())
			allChildren = epicChildren // fall back to API data
		}

		return syncCompleteMsg{tickets: mainTickets, epicChildren: allChildren}
	}
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
