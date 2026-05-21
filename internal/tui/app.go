package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/config"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/store"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

// navLevel tracks whether user is at root ticket list or inside an epic.
type navLevel int

const (
	navRoot navLevel = iota
	navEpic
)

type AppModel struct {
	// Dependencies
	jiraClient  jira.TicketClient
	tagStore    store.TagStore
	todoStore   store.TodoStore
	ticketCache store.TicketCache
	config      *config.Config

	// State
	list         *views.ListModel
	previousList *views.ListModel
	navLevel     navLevel
	activeModel  views.ActiveModel
	popups       *PopupManager
	epicChildren map[string][]model.Ticket
	err          error
	syncing      bool
	synced       bool

	styles views.Styles
	width  int
	height int
}

func NewApp(
	client jira.TicketClient,
	tags store.TagStore,
	todos store.TodoStore,
	cache store.TicketCache,
	cfg *config.Config,
) *AppModel {
	styles := views.NewStyles()
	listModel := views.NewListModel(nil, styles.App)
	tagModel := views.NewTagModel(0, 0, nil)
	todoModel := views.NewTodoModel(0, 0, "", nil)
	statusModel := views.NewStatusModel(0, 0)
	toastModel := views.NewToastModel(0, 0)

	app := &AppModel{
		jiraClient:   client,
		tagStore:     tags,
		todoStore:    todos,
		ticketCache:  cache,
		config:       cfg,
		list:         listModel,
		navLevel:     navRoot,
		activeModel:  listModel,
		popups:       newPopupManager(tagModel, todoModel, statusModel, toastModel),
		epicChildren: make(map[string][]model.Ticket),
		styles:       styles,
	}

	app.popups.SetKeyHandlers(app.handleTagKey, app.handleTodoKey, app.handleStatusKey)

	return app
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

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case cachedTicketsLoadedMsg,
		syncCompleteMsg,
		syncErrorMsg:
		return m.updateSyncMsg(msg)

	case views.SelectTicketMsg,
		views.GoToListMsg,
		epicChildrenLoadedMsg,
		epicChildrenErrorMsg,
		copilotLaunchedMsg:
		return m.updateNavigationMsg(msg)

	case views.TagsFilledMsg,
		views.TodosChangedMsg,
		tagSavedMsg,
		todoSavedMsg:
		return m.updatePopupMsg(msg)

	case transitionsLoadedMsg,
		transitionsErrorMsg,
		views.StatusTransitionMsg,
		statusTransitionCompleteMsg,
		statusTransitionErrorMsg:
		return m.updateStatusMsg(msg)

	case commentsLoadedMsg,
		commentsErrorMsg,
		views.CommentSubmitMsg,
		views.CommentCancelMsg,
		commentPostSuccessMsg,
		commentPostErrorMsg,
		views.UserSearchRequestMsg,
		userSearchResultMsg:
		return m.updateCommentMsg(msg)

	case views.ErrMsg:
		return m.handleError(msg)

	case views.ToastTimeoutMsg:
		if m.popups.toast.ShouldHide(msg) {
			m.popups.toast.Hide()
		}
		return m, nil

	default:
		return m.handleOtherMsg(msg)
	}
}

func (m *AppModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("\nError: %v\n\nPress 'r' to retry or 'q' to quit.", m.err))
	}

	base := m.styles.App.Render(m.activeModel.View().Content)

	layers := m.popups.Layers()
	if len(layers) > 0 {
		all := append([]*lipgloss.Layer{lipgloss.NewLayer(base)}, layers...)
		v := tea.NewView(lipgloss.NewCompositor(all...).Render())
		v.AltScreen = true
		return v
	}

	v := tea.NewView(base)
	v.AltScreen = true
	return v
}

func Run(
	ctx context.Context,
	client jira.TicketClient,
	tags store.TagStore,
	todos store.TodoStore,
	cache store.TicketCache,
	cfg *config.Config,
) error {
	app := NewApp(client, tags, todos, cache, cfg)
	p := tea.NewProgram(app)

	// Quit TUI gracefully when context is cancelled (e.g. SIGINT/SIGTERM).
	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
