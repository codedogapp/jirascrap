package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/store"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

type AppModel struct {
	// Dependencies
	jiraClient *jira.Client
	store      store.MetaStore

	// State
	list        *views.ListModel
	activeModel views.ActiveModel
	debugModel  *views.DebugModel
	err         error
	syncing     bool
	synced      bool

	styles views.Styles
	width  int
	height int
}

func NewApp(client *jira.Client, s store.MetaStore) *AppModel {
	styles := views.NewStyles()
	listModel := views.NewListModel(nil, styles.App)
	debugModel := views.NewDebugModel(0, 0)
	return &AppModel{
		jiraClient:  client,
		store:       s,
		list:        listModel,
		activeModel: listModel,
		debugModel:  debugModel,
		styles:      styles,
	}
}

func (m *AppModel) Init() tea.Cmd {
	m.syncing = true
	return tea.Batch(
		m.list.StartSpinner(),
		m.loadCachedTickets(),
		m.syncFromJira(),
	)
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case cachedTicketsLoadedMsg:
		return m.handleCachedTicketsLoaded(msg)

	case syncCompleteMsg:
		return m.handleSyncComplete(msg)

	case syncErrorMsg:
		return m.handleSyncError(msg)

	case views.ErrMsg:
		return m.handleError(msg)

	case views.SelectTicketMsg:
		return m.handleSelectTicket(msg)

	case views.GoToListMsg:
		return m.handleGoToList(msg)

	case views.TagsFilledMsg:
		return m.handleTagFilled(msg)

	case views.TodosChangedMsg:
		return m.handleTodosChanged(msg)

	case tagSavedMsg:
		return m.handleTagSaved(msg)

	case tea.KeyPressMsg:
		cmd := handleQuit(m, msg)
		if cmd != nil {
			return m, cmd
		}
		if consumed, cmd := handleDebug(m, msg); consumed {
			return m, cmd
		}
		if consumed, cmd := handleRefresh(m, msg); consumed {
			return m, cmd
		}
		m.activeModel, cmd = m.activeModel.Update(msg)
		return m, cmd

	default:
		if mu, ok := m.activeModel.(views.MsgUpdater); ok {
			return m, mu.UpdateMsg(msg)
		}
	}

	return m, nil
}

func (m *AppModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("\nError: %v\n\nPress 'q' to quit.", m.err))
	}

	base := m.styles.App.Render(m.activeModel.View().Content)

	debug := m.debugModel.View()

	if debug != nil {
		return tea.NewView(
			lipgloss.NewCompositor(
				lipgloss.NewLayer(base),
				debug,
			).Render(),
		)
	}

	return tea.NewView(base)
}

func Run(client *jira.Client, s store.MetaStore) error {
	app := NewApp(client, s)
	p := tea.NewProgram(app)
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
