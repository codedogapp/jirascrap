package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/config"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/store"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

type AppModel struct {
	// Dependencies
	jiraClient jira.TicketClient
	store      store.MetaStore
	config     *config.Config

	// State
	list         *views.ListModel
	previousList *views.ListModel
	activeModel  views.ActiveModel
	debugModel   *views.DebugModel
	tagModel     *views.TagModel
	todoModel    *views.TodoModel
	statusModel  *views.StatusModel
	toastModel   *views.ToastModel
	epicChildren map[string][]model.Ticket
	err          error
	syncing      bool
	synced       bool

	styles views.Styles
	width  int
	height int
}

func NewApp(client jira.TicketClient, s store.MetaStore, cfg *config.Config) *AppModel {
	styles := views.NewStyles()
	listModel := views.NewListModel(nil, styles.App)
	debugModel := views.NewDebugModel(0, 0)
	return &AppModel{
		jiraClient:   client,
		store:        s,
		config:       cfg,
		list:         listModel,
		activeModel:  listModel,
		debugModel:   debugModel,
		tagModel:     views.NewTagModel(0, 0, nil),
		todoModel:    views.NewTodoModel(0, 0, "", nil),
		statusModel:  views.NewStatusModel(0, 0),
		toastModel:   views.NewToastModel(0, 0),
		epicChildren: make(map[string][]model.Ticket),
		styles:       styles,
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

	case epicChildrenLoadedMsg:
		return m.handleEpicChildrenLoaded(msg)

	case epicChildrenErrorMsg:
		return m.handleError(views.ErrMsg{Err: msg.err})

	case views.GoToListMsg:
		return m.handleGoToList(msg)

	case views.TagsFilledMsg:
		return m.handleTagFilled(msg)

	case views.TodosChangedMsg:
		return m.handleTodosChanged(msg)

	case tagSavedMsg:
		return m.handleTagSaved(msg)

	case copilotLaunchedMsg:
		return m.handleCopilotLaunched(msg)

	case transitionsLoadedMsg:
		return m.handleTransitionsLoaded(msg)

	case transitionsErrorMsg:
		m.statusModel.Hide()
		return m.handleError(views.ErrMsg{Err: msg.err})

	case views.StatusTransitionMsg:
		return m.handleStatusTransition(msg)

	case statusTransitionCompleteMsg:
		return m.handleStatusTransitionComplete(msg)

	case statusTransitionErrorMsg:
		return m.handleError(views.ErrMsg{Err: msg.err})

	case views.ToastTimeoutMsg:
		m.toastModel.Hide()
		return m, nil

	case tea.KeyPressMsg:
		cmd := m.handleQuit(msg)
		if cmd != nil {
			return m, cmd
		}
		if consumed, cmd := m.handleDebug(msg); consumed {
			return m, cmd
		}
		// Route keys to popups when active
		if m.tagModel.IsVisible() {
			return m.handleTagKey(msg)
		}
		if m.todoModel.IsVisible() {
			return m.handleTodoKey(msg)
		}
		if m.statusModel.IsVisible() {
			return m.handleStatusKey(msg)
		}
		if consumed, cmd := m.handleRefresh(msg); consumed {
			return m, cmd
		}
		if consumed, cmd := m.handleExitEpic(msg); consumed {
			return m, cmd
		}
		if consumed, cmd := m.handleGoHome(msg); consumed {
			return m, cmd
		}
		if consumed, cmd := m.handleOpenInBrowser(msg); consumed {
			return m, cmd
		}
		if consumed, cmd := m.handleToggleTag(msg); consumed {
			return m, cmd
		}
		if consumed, cmd := m.handleToggleTodo(msg); consumed {
			return m, cmd
		}
		if consumed, cmd := m.handleToggleStatus(msg); consumed {
			return m, cmd
		}
		if consumed, cmd := m.handleSendToCopilot(msg); consumed {
			return m, cmd
		}
		m.activeModel, cmd = m.activeModel.Update(msg)
		return m, cmd

	default:
		// Route non-key messages to active popup for text input blinking etc.
		if m.tagModel.IsVisible() {
			return m, m.tagModel.UpdateMsg(msg)
		}
		if m.todoModel.IsVisible() && m.todoModel.IsAdding() {
			return m, m.todoModel.UpdateMsg(msg)
		}
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
	todoOverlay := m.todoModel.View()
	tagOverlay := m.tagModel.View()
	statusOverlay := m.statusModel.View()
	toastOverlay := m.toastModel.View()

	hasOverlay := debug != nil || todoOverlay != nil || tagOverlay != nil || statusOverlay != nil || toastOverlay != nil
	if hasOverlay {
		layers := []*lipgloss.Layer{lipgloss.NewLayer(base)}
		if todoOverlay != nil {
			layers = append(layers, todoOverlay)
		}
		if tagOverlay != nil {
			layers = append(layers, tagOverlay)
		}
		if statusOverlay != nil {
			layers = append(layers, statusOverlay)
		}
		if debug != nil {
			layers = append(layers, debug)
		}
		if toastOverlay != nil {
			layers = append(layers, toastOverlay)
		}
		v := tea.NewView(lipgloss.NewCompositor(layers...).Render())
		v.AltScreen = true
		return v
	}

	v := tea.NewView(base)
	v.AltScreen = true
	return v
}

func Run(client jira.TicketClient, s store.MetaStore, cfg *config.Config) error {
	app := NewApp(client, s, cfg)
	p := tea.NewProgram(app)
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
