package tui

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/logger"
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

const apiTimeout = 30 * time.Second

func (m *AppModel) updateSyncMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case cachedTicketsLoadedMsg:
		return m.handleCachedTicketsLoaded(msg)
	case syncCompleteMsg:
		return m.handleSyncComplete(msg)
	case syncErrorMsg:
		return m.handleSyncError(msg)
	default:
		return m, nil
	}
}

func (m *AppModel) loadCachedTickets() tea.Cmd {
	return func() tea.Msg {
		epicChildren, err := m.ticketCache.GetAllCachedEpicChildren()
		if err != nil {
			logger.Log.Warn(fmt.Sprintf("failed to load cached epic children: %v", err))
		}
		tickets, err := m.ticketCache.GetCachedTickets()
		if err != nil {
			logger.Log.Warn(fmt.Sprintf("failed to load cached tickets: %v", err))
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
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()

		tickets, err := m.jiraClient.FetchTickets(ctx)
		if err != nil {
			return syncErrorMsg{err: err}
		}
		if err := m.ticketCache.CacheTickets(tickets); err != nil {
			logger.Log.Warn(fmt.Sprintf("failed to cache tickets: %v", err))
		}

		epicCtx, epicCancel := context.WithTimeout(context.Background(), apiTimeout)
		defer epicCancel()

		epicChildren, err := m.jiraClient.FetchAllEpicChildren(epicCtx, tickets)
		if err != nil {
			logger.Log.Warn(fmt.Sprintf("failed to fetch some epic children: %v", err))
		}
		for epicKey, children := range epicChildren {
			if err := m.ticketCache.CacheEpicChildren(epicKey, children); err != nil {
				logger.Log.Warn(fmt.Sprintf("failed to cache epic children for %s: %v", epicKey, err))
			}
		}

		// Re-read from DB: tags joined, epic children excluded from main list
		mainTickets, err := m.ticketCache.GetCachedTickets()
		if err != nil {
			logger.Log.Warn(fmt.Sprintf("failed to re-read cached tickets: %v", err))
			mainTickets = tickets // fall back to API data
		}
		allChildren, err := m.ticketCache.GetAllCachedEpicChildren()
		if err != nil {
			logger.Log.Warn(fmt.Sprintf("failed to re-read epic children: %v", err))
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
	root.StopSpinner()

	if root.HasTickets() {
		return m, m.popups.toast.Show(fmt.Sprintf("✗ Sync failed: %v", msg.err))
	}

	// No cached data at all — show full error screen
	m.err = views.ErrMsg{Err: msg.err}
	return m, nil
}

func (m *AppModel) handleError(msg views.ErrMsg) (tea.Model, tea.Cmd) {
	m.list.StopSpinner()
	return m, m.popups.toast.Show(fmt.Sprintf("✗ %v", msg.Err))
}

func (m *AppModel) handleRefresh(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if key.Matches(msg, keymaps.DefaultKeyMap.Refresh) &&
		!m.isPopupActive() &&
		!m.syncing &&
		!m.list.IsFiltering() {
		m.err = nil
		m.syncing = true
		root := m.rootList()
		root.SetTitle("Jira Tickets (syncing...)")
		return true, tea.Batch(root.StartSpinner(), m.syncFromJira())
	}

	return false, nil
}
