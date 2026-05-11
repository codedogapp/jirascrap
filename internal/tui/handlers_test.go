package tui

import (
	"testing"

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

func (m *mockStore) SaveMeta(id string, tags []string) error {
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

func TestIsPopupActive_OnList(t *testing.T) {
	styles := views.NewStyles()
	list := views.NewListModel(nil, styles.App)
	app := &AppModel{
		activeModel: list,
		tagModel:    views.NewTagModel(0, 0, nil),
		todoModel:   views.NewTodoModel(0, 0, "", nil),
		statusModel: views.NewStatusModel(0, 0),
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
		statusModel: views.NewStatusModel(0, 0),
	}

	if app.isPopupActive() {
		t.Error("expected false when no popup is open")
	}
}

func TestIsPopupActive_StatusVisible(t *testing.T) {
	styles := views.NewStyles()
	list := views.NewListModel(nil, styles.App)
	statusModel := views.NewStatusModel(0, 0)
	statusModel.Show(model.Ticket{ID: "T-1", Summary: "test"})

	app := &AppModel{
		activeModel: list,
		tagModel:    views.NewTagModel(0, 0, nil),
		todoModel:   views.NewTodoModel(0, 0, "", nil),
		statusModel: statusModel,
	}

	if !app.isPopupActive() {
		t.Error("expected true when status dropdown is visible")
	}
}
