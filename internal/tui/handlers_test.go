package tui

import (
	"testing"

	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/store"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

type mockStore struct {
	meta     map[string]store.LocalMeta
	tags     []string
	todos    map[string][]model.Todo
	tickets  []model.Ticket
	metaErr  error
	tagsErr  error
	todosErr error
	cacheErr error
}

func (m *mockStore) SaveMeta(id string, tags []string) error {
	if m.meta == nil {
		m.meta = make(map[string]store.LocalMeta)
	}
	m.meta[id] = store.LocalMeta{Tags: tags}
	return m.metaErr
}

func (m *mockStore) GetAllMeta() (map[string]store.LocalMeta, error) {
	if m.meta == nil {
		return map[string]store.LocalMeta{}, m.metaErr
	}
	return m.meta, m.metaErr
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

func TestMergeLocalMeta(t *testing.T) {
	s := &mockStore{
		meta: map[string]store.LocalMeta{
			"TICK-1": {Tags: []string{"bug", "urgent"}},
			"TICK-3": {Tags: []string{"feature"}},
		},
	}
	app := &AppModel{store: s}

	tickets := []model.Ticket{
		{ID: "TICK-1", Summary: "First"},
		{ID: "TICK-2", Summary: "Second"},
		{ID: "TICK-3", Summary: "Third"},
	}

	app.mergeLocalMeta(tickets)

	if len(tickets[0].Tags) != 2 || tickets[0].Tags[0] != "bug" {
		t.Errorf("TICK-1 tags = %v", tickets[0].Tags)
	}
	if tickets[1].Tags != nil {
		t.Errorf("TICK-2 tags = %v, want nil", tickets[1].Tags)
	}
	if len(tickets[2].Tags) != 1 || tickets[2].Tags[0] != "feature" {
		t.Errorf("TICK-3 tags = %v", tickets[2].Tags)
	}
}

func TestMergeLocalMeta_EmptyMeta(t *testing.T) {
	s := &mockStore{}
	app := &AppModel{store: s}

	tickets := []model.Ticket{{ID: "TICK-1"}}
	app.mergeLocalMeta(tickets)

	if tickets[0].Tags != nil {
		t.Errorf("tags = %v, want nil", tickets[0].Tags)
	}
}

func TestIsPopupActive_OnList(t *testing.T) {
	styles := views.NewStyles()
	list := views.NewListModel(nil, styles.App)
	app := &AppModel{
		activeModel: list,
		tagModel:    views.NewTagModel(0, 0, nil),
		todoModel:   views.NewTodoModel(0, 0, "", nil),
	}

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
	app := &AppModel{
		activeModel: detail,
		tagModel:    views.NewTagModel(0, 0, nil),
		todoModel:   views.NewTodoModel(0, 0, "", nil),
	}

	if app.isPopupActive() {
		t.Error("expected false when no popup is open")
	}
}
