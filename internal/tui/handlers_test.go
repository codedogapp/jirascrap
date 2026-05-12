package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

type mockStore struct {
	tags         []string
	todos        map[string][]model.Todo
	tickets      []model.Ticket
	epicChildren map[string][]model.Ticket
	tagsErr      error
	todosErr     error
	cacheErr     error
}

func (m *mockStore) SaveTags(id string, tags []string) error {
	return nil
}

func (m *mockStore) GetUniqueTags() ([]string, error) {
	return m.tags, m.tagsErr
}

func (m *mockStore) GetTodos(ticketID string) ([]model.Todo, error) {
	if m.todos == nil {
		return nil, m.todosErr
	}
	return m.todos[ticketID], m.todosErr
}

func (m *mockStore) SaveTodos(ticketID string, todos []model.Todo) error {
	if m.todos == nil {
		m.todos = make(map[string][]model.Todo)
	}
	m.todos[ticketID] = todos
	return m.todosErr
}

func (m *mockStore) CacheTickets(tickets []model.Ticket) error {
	m.tickets = tickets
	return m.cacheErr
}

func (m *mockStore) GetCachedTickets() ([]model.Ticket, error) {
	return m.tickets, m.cacheErr
}

func (m *mockStore) CacheEpicChildren(epicKey string, tickets []model.Ticket) error {
	if m.epicChildren == nil {
		m.epicChildren = make(map[string][]model.Ticket)
	}
	m.epicChildren[epicKey] = tickets
	return nil
}

func (m *mockStore) GetAllCachedEpicChildren() (map[string][]model.Ticket, error) {
	if m.epicChildren == nil {
		return map[string][]model.Ticket{}, nil
	}
	return m.epicChildren, nil
}

// --- Helper ---

func newHandlerTestApp(tickets []model.Ticket) *AppModel {
	styles := views.NewStyles()
	list := views.NewListModel(tickets, styles.App)
	list.SetSize(120, 40)
	tagModel := views.NewTagModel(80, 30, nil)
	todoModel := views.NewTodoModel(80, 30, "", nil)
	statusModel := views.NewStatusModel(80, 30)
	toastModel := views.NewToastModel(120, 40)
	ms := &mockStore{tickets: tickets}
	return &AppModel{
		tagStore:     ms,
		todoStore:    ms,
		ticketCache:  ms,
		list:         list,
		activeModel:  list,
		popups:       newPopupManager(tagModel, todoModel, statusModel, toastModel),
		epicChildren: make(map[string][]model.Ticket),
		styles:       styles,
		width:        120,
		height:       40,
	}
}

// --- Popup state tests ---

func TestIsPopupActive_OnList(t *testing.T) {
	app := newHandlerTestApp(nil)

	if app.isPopupActive() {
		t.Error("expected false on list view")
	}
}

func TestIsPopupActive_OnDetail_NoPopup(t *testing.T) {
	styles := views.NewStyles()
	detail := views.NewDetailModel(
		model.Ticket{ID: "T-1", Summary: "test"},
		80, 24, styles,
	)
	app := newHandlerTestApp(nil)
	app.activeModel = detail

	if app.isPopupActive() {
		t.Error("expected false when no popup is open")
	}
}

func TestIsPopupActive_StatusVisible(t *testing.T) {
	app := newHandlerTestApp(nil)
	app.popups.status.Show(model.Ticket{ID: "T-1", Summary: "test"})

	if !app.isPopupActive() {
		t.Error("expected true when status dropdown is visible")
	}
}

// --- Handler tests ---

func TestHandleSyncComplete_UpdatesList(t *testing.T) {
	now := time.Now()
	tickets := []model.Ticket{
		{ID: "T-1", Summary: "First", CreatedAt: now, UpdatedAt: now},
		{ID: "T-2", Summary: "Second", CreatedAt: now, UpdatedAt: now},
	}
	app := newHandlerTestApp(nil)
	app.syncing = true

	app.handleSyncComplete(syncCompleteMsg{tickets: tickets, epicChildren: map[string][]model.Ticket{}})

	if !app.synced {
		t.Error("expected synced=true after handleSyncComplete")
	}
	if app.syncing {
		t.Error("expected syncing=false after handleSyncComplete")
	}
	if !app.list.HasTickets() {
		t.Error("expected list to have tickets after sync")
	}
}

func TestHandleSyncError_WithCachedData(t *testing.T) {
	now := time.Now()
	tickets := []model.Ticket{
		{ID: "T-1", Summary: "Cached", CreatedAt: now, UpdatedAt: now},
	}
	app := newHandlerTestApp(tickets)
	app.list.Initialize(tickets)
	app.syncing = true

	app.handleSyncError(syncErrorMsg{err: nil})

	if app.syncing {
		t.Error("expected syncing=false")
	}
	if app.err != nil {
		t.Error("expected no error when cached data exists")
	}
}

func TestHandleSyncError_NoCachedData(t *testing.T) {
	app := newHandlerTestApp(nil)
	app.syncing = true

	app.handleSyncError(syncErrorMsg{err: views.ErrMsg{Err: nil}})

	if app.syncing {
		t.Error("expected syncing=false")
	}
}

func TestHandleCachedTicketsLoaded_PopulatesList(t *testing.T) {
	now := time.Now()
	tickets := []model.Ticket{
		{ID: "T-1", Summary: "Cached", CreatedAt: now, UpdatedAt: now},
	}
	app := newHandlerTestApp(nil)

	app.handleCachedTicketsLoaded(cachedTicketsLoadedMsg{tickets: tickets})

	if !app.list.HasTickets() {
		t.Error("expected list populated from cache")
	}
}

func TestHandleCachedTicketsLoaded_SkipsIfAlreadySynced(t *testing.T) {
	now := time.Now()
	app := newHandlerTestApp(nil)
	app.synced = true

	app.handleCachedTicketsLoaded(cachedTicketsLoadedMsg{
		tickets: []model.Ticket{{ID: "T-1", Summary: "x", CreatedAt: now, UpdatedAt: now}},
	})

	if app.list.HasTickets() {
		t.Error("expected list unchanged when already synced")
	}
}

func TestActiveDetailModel_ReturnsNilOnList(t *testing.T) {
	app := newHandlerTestApp(nil)

	_, ok := app.activeDetailModel()
	if ok {
		t.Error("expected false when active model is list")
	}
}

func TestActiveDetailModel_ReturnsDetailWhenActive(t *testing.T) {
	styles := views.NewStyles()
	detail := views.NewDetailModel(
		model.Ticket{ID: "T-1", Summary: "test"},
		80, 24, styles,
	)
	app := newHandlerTestApp(nil)
	app.activeModel = detail

	dm, ok := app.activeDetailModel()
	if !ok {
		t.Fatal("expected true when active model is detail")
	}
	if dm.Ticket().ID != "T-1" {
		t.Errorf("expected T-1, got %s", dm.Ticket().ID)
	}
}

func TestFindTicket_InRootList(t *testing.T) {
	now := time.Now()
	tickets := []model.Ticket{
		{ID: "T-1", Summary: "First", CreatedAt: now, UpdatedAt: now},
		{ID: "T-2", Summary: "Second", CreatedAt: now, UpdatedAt: now},
	}
	app := newHandlerTestApp(tickets)
	app.list.Initialize(tickets)

	ticket, ok := app.findTicket("T-2")
	if !ok {
		t.Fatal("expected to find T-2")
	}
	if ticket.Summary != "Second" {
		t.Errorf("expected Second, got %s", ticket.Summary)
	}
}

func TestFindTicket_InEpicChildren(t *testing.T) {
	now := time.Now()
	app := newHandlerTestApp(nil)
	app.epicChildren = map[string][]model.Ticket{
		"EPIC-1": {{ID: "CHILD-1", Summary: "Child", CreatedAt: now, UpdatedAt: now}},
	}

	ticket, ok := app.findTicket("CHILD-1")
	if !ok {
		t.Fatal("expected to find CHILD-1 in epic children")
	}
	if ticket.Summary != "Child" {
		t.Errorf("expected Child, got %s", ticket.Summary)
	}
}

func TestFindTicket_NotFound(t *testing.T) {
	app := newHandlerTestApp(nil)

	_, ok := app.findTicket("NONEXIST")
	if ok {
		t.Error("expected not found")
	}
}

func TestHandleWindowSize_UpdatesAllModels(t *testing.T) {
	app := newHandlerTestApp(nil)

	app.handleWindowSize(tea.WindowSizeMsg{Width: 200, Height: 60})

	if app.width != 200 || app.height != 60 {
		t.Errorf("expected 200x60, got %dx%d", app.width, app.height)
	}
}
