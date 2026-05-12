package store

import (
	"testing"

	"github.com/codedogapp/jirascrap/internal/model"
)

func TestSaveTodos_AndGetTodos(t *testing.T) {
	s := setupTestDB(t)

	todos := []model.Todo{
		{Title: "Write tests", Done: false},
		{Title: "Fix bug", Done: true},
	}
	if err := s.Todos.SaveTodos("TICK-1", todos); err != nil {
		t.Fatalf("SaveTodos: %v", err)
	}

	got, err := s.Todos.GetTodos("TICK-1")
	if err != nil {
		t.Fatalf("GetTodos: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(got))
	}
	if got[0].Title != "Write tests" || got[0].Done != false {
		t.Errorf("todo[0] = %+v", got[0])
	}
	if got[1].Title != "Fix bug" || got[1].Done != true {
		t.Errorf("todo[1] = %+v", got[1])
	}
}

func TestSaveTodos_Overwrite(t *testing.T) {
	s := setupTestDB(t)

	_ = s.Todos.SaveTodos("TICK-1", []model.Todo{{Title: "old"}})
	_ = s.Todos.SaveTodos("TICK-1", []model.Todo{{Title: "new"}})

	got, _ := s.Todos.GetTodos("TICK-1")
	if len(got) != 1 || got[0].Title != "new" {
		t.Errorf("got %+v", got)
	}
}

func TestSaveTodos_Empty(t *testing.T) {
	s := setupTestDB(t)

	_ = s.Todos.SaveTodos("TICK-1", []model.Todo{{Title: "a"}})
	_ = s.Todos.SaveTodos("TICK-1", []model.Todo{})

	got, _ := s.Todos.GetTodos("TICK-1")
	if len(got) != 0 {
		t.Errorf("expected empty, got %+v", got)
	}
}

func TestGetTodos_NonExistent(t *testing.T) {
	s := setupTestDB(t)

	got, err := s.Todos.GetTodos("NONEXIST")
	if err != nil {
		t.Fatalf("GetTodos: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %+v", got)
	}
}
