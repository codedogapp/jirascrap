package store

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"github.com/codedogapp/jirascrap/internal/model"
)

func setupTestDB(t *testing.T) *SqliteMetaStore {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(db, "migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return NewSqliteMetaStore(db)
}

// --- Tags ---

func TestSaveMeta_AndGetAllMeta(t *testing.T) {
	s := setupTestDB(t)

	if err := s.SaveMeta("TICK-1", []string{"bug", "urgent"}); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	meta, err := s.GetAllMeta()
	if err != nil {
		t.Fatalf("GetAllMeta: %v", err)
	}
	if len(meta) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(meta))
	}
	tags := meta["TICK-1"].Tags
	if len(tags) != 2 || tags[0] != "bug" || tags[1] != "urgent" {
		t.Errorf("tags = %v, want [bug urgent]", tags)
	}
}

func TestSaveMeta_Overwrite(t *testing.T) {
	s := setupTestDB(t)

	_ = s.SaveMeta("TICK-1", []string{"old-tag"})
	_ = s.SaveMeta("TICK-1", []string{"new-tag"})

	meta, _ := s.GetAllMeta()
	tags := meta["TICK-1"].Tags
	if len(tags) != 1 || tags[0] != "new-tag" {
		t.Errorf("tags = %v, want [new-tag]", tags)
	}
}

func TestSaveMeta_EmptyTags(t *testing.T) {
	s := setupTestDB(t)

	_ = s.SaveMeta("TICK-1", []string{"tag"})
	_ = s.SaveMeta("TICK-1", []string{})

	meta, _ := s.GetAllMeta()
	if _, ok := meta["TICK-1"]; ok {
		t.Error("expected TICK-1 to have no entry after clearing tags")
	}
}

func TestSaveMeta_MultipleTickets(t *testing.T) {
	s := setupTestDB(t)

	_ = s.SaveMeta("TICK-1", []string{"a"})
	_ = s.SaveMeta("TICK-2", []string{"b", "c"})

	meta, _ := s.GetAllMeta()
	if len(meta) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(meta))
	}
}

func TestGetUniqueTags(t *testing.T) {
	s := setupTestDB(t)

	_ = s.SaveMeta("TICK-1", []string{"bug", "urgent"})
	_ = s.SaveMeta("TICK-2", []string{"bug", "feature"})

	tags, err := s.GetUniqueTags()
	if err != nil {
		t.Fatalf("GetUniqueTags: %v", err)
	}
	// Should be sorted: bug, feature, urgent
	want := []string{"bug", "feature", "urgent"}
	if len(tags) != len(want) {
		t.Fatalf("tags = %v, want %v", tags, want)
	}
	for i, w := range want {
		if tags[i] != w {
			t.Errorf("tags[%d] = %q, want %q", i, tags[i], w)
		}
	}
}

func TestGetUniqueTags_Empty(t *testing.T) {
	s := setupTestDB(t)

	tags, err := s.GetUniqueTags()
	if err != nil {
		t.Fatalf("GetUniqueTags: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected empty, got %v", tags)
	}
}

// --- Todos ---

func TestSaveTodos_AndGetTodos(t *testing.T) {
	s := setupTestDB(t)

	todos := []model.Todo{
		{Title: "Write tests", Done: false},
		{Title: "Fix bug", Done: true},
	}
	if err := s.SaveTodos("TICK-1", todos); err != nil {
		t.Fatalf("SaveTodos: %v", err)
	}

	got, err := s.GetTodos("TICK-1")
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

	_ = s.SaveTodos("TICK-1", []model.Todo{{Title: "old"}})
	_ = s.SaveTodos("TICK-1", []model.Todo{{Title: "new"}})

	got, _ := s.GetTodos("TICK-1")
	if len(got) != 1 || got[0].Title != "new" {
		t.Errorf("got %+v", got)
	}
}

func TestSaveTodos_Empty(t *testing.T) {
	s := setupTestDB(t)

	_ = s.SaveTodos("TICK-1", []model.Todo{{Title: "a"}})
	_ = s.SaveTodos("TICK-1", []model.Todo{})

	got, _ := s.GetTodos("TICK-1")
	if len(got) != 0 {
		t.Errorf("expected empty, got %+v", got)
	}
}

func TestGetTodos_NonExistent(t *testing.T) {
	s := setupTestDB(t)

	got, err := s.GetTodos("NONEXIST")
	if err != nil {
		t.Fatalf("GetTodos: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %+v", got)
	}
}

// --- Ticket Cache ---

func TestCacheTickets_AndGetCached(t *testing.T) {
	s := setupTestDB(t)

	now := time.Now().Truncate(time.Second)
	tickets := []model.Ticket{
		{
			ID: "PROJ-1", Summary: "First", Reporter: "Alice",
			Status: "Open", StatusCategory: "To Do", Priority: "High",
			CreatedAt: now, UpdatedAt: now, Markdown: "# Description",
		},
		{
			ID: "PROJ-2", Summary: "Second", Reporter: "Bob",
			Status: "Done", StatusCategory: "Done", Priority: "Low",
			CreatedAt: now, UpdatedAt: now, Markdown: "body",
		},
	}

	if err := s.CacheTickets(tickets); err != nil {
		t.Fatalf("CacheTickets: %v", err)
	}

	got, err := s.GetCachedTickets()
	if err != nil {
		t.Fatalf("GetCachedTickets: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tickets, got %d", len(got))
	}
	if got[0].ID != "PROJ-1" || got[0].Summary != "First" || got[0].Reporter != "Alice" {
		t.Errorf("ticket[0] = %+v", got[0])
	}
	if got[0].Priority != "High" || got[0].Status != "Open" {
		t.Errorf("ticket[0] fields = %+v", got[0])
	}
	if got[0].Markdown != "# Description" {
		t.Errorf("ticket[0] markdown = %q", got[0].Markdown)
	}
}

func TestCacheTickets_FullReplacement(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	old := []model.Ticket{
		{ID: "OLD-1", Summary: "old", CreatedAt: now, UpdatedAt: now},
		{ID: "OLD-2", Summary: "old2", CreatedAt: now, UpdatedAt: now},
	}
	_ = s.CacheTickets(old)

	new := []model.Ticket{
		{ID: "NEW-1", Summary: "new", CreatedAt: now, UpdatedAt: now},
	}
	_ = s.CacheTickets(new)

	got, _ := s.GetCachedTickets()
	if len(got) != 1 {
		t.Fatalf("expected 1 ticket after replacement, got %d", len(got))
	}
	if got[0].ID != "NEW-1" {
		t.Errorf("got ID %q, want NEW-1", got[0].ID)
	}
}

func TestCacheTickets_EmptyClears(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.CacheTickets([]model.Ticket{
		{ID: "T-1", Summary: "x", CreatedAt: now, UpdatedAt: now},
	})
	_ = s.CacheTickets([]model.Ticket{})

	got, _ := s.GetCachedTickets()
	if len(got) != 0 {
		t.Errorf("expected empty after clear, got %d", len(got))
	}
}

func TestCacheTickets_TimeRoundTrip(t *testing.T) {
	s := setupTestDB(t)
	created := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	updated := time.Date(2024, 7, 20, 14, 0, 0, 0, time.UTC)

	_ = s.CacheTickets([]model.Ticket{
		{ID: "T-1", Summary: "x", CreatedAt: created, UpdatedAt: updated},
	})

	got, _ := s.GetCachedTickets()
	if !got[0].CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", got[0].CreatedAt, created)
	}
	if !got[0].UpdatedAt.Equal(updated) {
		t.Errorf("UpdatedAt = %v, want %v", got[0].UpdatedAt, updated)
	}
}

func TestGetCachedTickets_Empty(t *testing.T) {
	s := setupTestDB(t)

	got, err := s.GetCachedTickets()
	if err != nil {
		t.Fatalf("GetCachedTickets: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

// --- Tags preserved after cache clear ---

func TestCacheTickets_PreservesLocalMeta(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	// Save tags for a ticket
	_ = s.SaveMeta("TICK-1", []string{"important"})

	// Cache tickets, then clear cache (ticket removed from Jira)
	_ = s.CacheTickets([]model.Ticket{{ID: "TICK-1", Summary: "x", CreatedAt: now, UpdatedAt: now}})
	_ = s.CacheTickets([]model.Ticket{})

	// Tags should still be preserved
	meta, _ := s.GetAllMeta()
	if tags := meta["TICK-1"].Tags; len(tags) != 1 || tags[0] != "important" {
		t.Errorf("expected tags preserved, got %v", tags)
	}
}
