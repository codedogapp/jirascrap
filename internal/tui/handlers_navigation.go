package tui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/logger"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

func (m *AppModel) updateNavigationMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case views.SelectTicketMsg:
		return m.handleSelectTicket(msg)
	case views.GoToListMsg:
		return m.handleGoToList(msg)
	case epicChildrenLoadedMsg:
		return m.handleEpicChildrenLoaded(msg)
	case epicChildrenErrorMsg:
		return m.handleError(views.ErrMsg{Err: msg.err})
	case copilotLaunchedMsg:
		return m.handleCopilotLaunched(msg)
	default:
		return m, nil
	}
}

func (m *AppModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.list.SetSize(msg.Width, msg.Height)

	if m.navLevel == navEpic {
		m.previousList.SetSize(msg.Width, msg.Height)
	}

	m.width = msg.Width
	m.height = msg.Height

	w, h := m.styles.App.GetFrameSize()
	m.popups.SetSize(msg.Width-w, msg.Height-h, msg.Width, msg.Height)

	return m, nil
}

// rootList returns the main ticket list, even when navigated into an epic.
func (m *AppModel) rootList() *views.ListModel {
	if m.navLevel == navEpic {
		return m.previousList
	}
	return m.list
}

func (m *AppModel) handleSelectTicket(msg views.SelectTicketMsg) (tea.Model, tea.Cmd) {
	if msg.Ticket.IsEpic() {
		if children, ok := m.epicChildren[msg.Ticket.ID]; ok {
			return m.showEpicChildren(msg.Ticket.ID, children)
		}
		return m, tea.Batch(m.list.StartSpinner(), m.fetchEpicChildrenCmd(msg.Ticket.ID))
	}

	m.activeModel = views.NewDetailModel(msg.Ticket, m.width, m.height, m.styles)
	return m, m.fetchCommentsCmd(msg.Ticket.ID)
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
	m.navLevel = navEpic
	m.activeModel = epicList
	return m, nil
}

func (m *AppModel) fetchEpicChildrenCmd(epicKey string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		tickets, err := m.jiraClient.FetchEpicChildren(ctx, epicKey)
		if err != nil {
			return epicChildrenErrorMsg{err: err}
		}
		return epicChildrenLoadedMsg{epicKey: epicKey, tickets: tickets}
	}
}

func (m *AppModel) handleEpicChildrenLoaded(msg epicChildrenLoadedMsg) (tea.Model, tea.Cmd) {
	if err := m.ticketCache.CacheEpicChildren(msg.epicKey, msg.tickets); err != nil {
		logger.Log.Warn(fmt.Sprintf("failed to cache epic children: %v", err))
	}
	// Re-read from DB to get tags joined in
	allChildren, err := m.ticketCache.GetAllCachedEpicChildren()
	if err != nil {
		logger.Log.Warn(fmt.Sprintf("failed to re-read epic children: %v", err))
		// Fall back to the data we have
		m.epicChildren[msg.epicKey] = msg.tickets
		m.list.StopSpinner()
		return m.showEpicChildren(msg.epicKey, msg.tickets)
	}
	children := allChildren[msg.epicKey]
	m.epicChildren[msg.epicKey] = children
	m.list.StopSpinner()
	return m.showEpicChildren(msg.epicKey, children)
}

func (m *AppModel) handleExitEpic(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !key.Matches(msg, keymaps.DefaultKeyMap.GoBack) {
		return false, nil
	}

	if m.navLevel != navEpic {
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
	if m.navLevel == navEpic {
		m.list = m.previousList
		m.previousList = nil
		m.navLevel = navRoot
	}
	m.activeModel = m.list
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

// activeDetailModel returns the DetailModel if it's the currently active view.
func (m *AppModel) activeDetailModel() (*views.DetailModel, bool) {
	dm, ok := m.activeModel.(*views.DetailModel)
	return dm, ok
}

// refreshListsFromDB re-reads tickets and epic children from DB, then updates all list views.
func (m *AppModel) refreshListsFromDB() {
	tickets, err := m.ticketCache.GetCachedTickets()
	if err != nil {
		logger.Log.Warn(fmt.Sprintf("failed to re-read tickets from DB: %v", err))
	} else {
		m.rootList().SetItems(tickets)
	}

	epicChildren, err := m.ticketCache.GetAllCachedEpicChildren()
	if err != nil {
		logger.Log.Warn(fmt.Sprintf("failed to re-read epic children from DB: %v", err))
	} else {
		m.epicChildren = epicChildren
	}

	m.refreshCurrentEpicView()
}

// refreshCurrentEpicView updates the current epic list view if user is inside one.
func (m *AppModel) refreshCurrentEpicView() {
	if m.navLevel != navEpic {
		return
	}
	for epicKey, children := range m.epicChildren {
		if strings.HasPrefix(m.list.Title(), fmt.Sprintf("⚡ %s", epicKey)) {
			m.list.SetItems(children)
			break
		}
	}
}

func (m *AppModel) handleQuit(msg tea.KeyPressMsg) tea.Cmd {
	if key.Matches(msg, keymaps.DefaultKeyMap.ForceQuit) ||
		(key.Matches(msg, keymaps.DefaultKeyMap.Quit) && !m.isPopupActive()) {
		return tea.Quit
	}
	return nil
}

func (m *AppModel) isPopupActive() bool {
	return m.popups.IsActive()
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

// withSelectedTicket guards actions that require a selected ticket and no active filtering/popup.
func (m *AppModel) withSelectedTicket() (model.Ticket, bool) {
	if m.isPopupActive() || m.list.IsFiltering() {
		return model.Ticket{}, false
	}
	return m.selectedTicket()
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
		_ = exec.Command("open", ticketURL).Start() // #nosec G204 -- ticketURL is from trusted config
		return nil
	}
}
