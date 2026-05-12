package store

import (
	"testing"
	"time"

	"github.com/codedogapp/jirascrap/internal/model"
)

func TestCacheTickets_AndGetCached(t *testing.T) {
	s := setupTestDB(t)

	now := time.Now().Truncate(time.Second)
	tickets := []model.Ticket{
		{
			ID:             "PROJ-1",
			Summary:        "First",
			Reporter:       "Alice",
			Status:         "Open",
			StatusCategory: "To Do",
			Priority:       "High",
			CreatedAt:      now,
			UpdatedAt:      now,
			Markdown:       "# Description",
		},
		{
			ID:             "PROJ-2",
			Summary:        "Second",
			Reporter:       "Bob",
			Status:         "Done",
			StatusCategory: "Done",
			Priority:       "Low",
			CreatedAt:      now,
			UpdatedAt:      now,
			Markdown:       "body",
		},
	}

	if err := s.Tickets.CacheTickets(tickets); err != nil {
		t.Fatalf("CacheTickets: %v", err)
	}

	got, err := s.Tickets.GetCachedTickets()
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
	_ = s.Tickets.CacheTickets(old)

	new := []model.Ticket{
		{ID: "NEW-1", Summary: "new", CreatedAt: now, UpdatedAt: now},
	}
	_ = s.Tickets.CacheTickets(new)

	got, _ := s.Tickets.GetCachedTickets()
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

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "T-1", Summary: "x", CreatedAt: now, UpdatedAt: now},
	})
	_ = s.Tickets.CacheTickets([]model.Ticket{})

	got, _ := s.Tickets.GetCachedTickets()
	if len(got) != 0 {
		t.Errorf("expected empty after clear, got %d", len(got))
	}
}

func TestCacheTickets_TimeRoundTrip(t *testing.T) {
	s := setupTestDB(t)
	created := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	updated := time.Date(2024, 7, 20, 14, 0, 0, 0, time.UTC)

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "T-1", Summary: "x", CreatedAt: created, UpdatedAt: updated},
	})

	got, _ := s.Tickets.GetCachedTickets()
	if !got[0].CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", got[0].CreatedAt, created)
	}
	if !got[0].UpdatedAt.Equal(updated) {
		t.Errorf("UpdatedAt = %v, want %v", got[0].UpdatedAt, updated)
	}
}

func TestGetCachedTickets_Empty(t *testing.T) {
	s := setupTestDB(t)

	got, err := s.Tickets.GetCachedTickets()
	if err != nil {
		t.Fatalf("GetCachedTickets: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestCacheTickets_PreservesLocalMeta(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.Tags.SaveMeta("TICK-1", []string{"important"})

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "TICK-1", Summary: "x", CreatedAt: now, UpdatedAt: now},
	})
	_ = s.Tickets.CacheTickets([]model.Ticket{})

	tags, _ := s.Tags.GetUniqueTags()
	if len(tags) != 1 || tags[0] != "important" {
		t.Errorf("expected tags preserved, got %v", tags)
	}
}

func TestCacheEpicChildren_AndGetAll(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "EPIC-1", Summary: "My Epic", Type: "Epic", CreatedAt: now, UpdatedAt: now},
	})

	children := []model.Ticket{
		{
			ID:             "CHILD-1",
			Summary:        "First child",
			Reporter:       "Alice",
			Status:         "Open",
			StatusCategory: "To Do",
			Priority:       "High",
			Type:           "Task",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			ID:             "CHILD-2",
			Summary:        "Second child",
			Reporter:       "Bob",
			Status:         "Done",
			StatusCategory: "Done",
			Priority:       "Low",
			Type:           "Task",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	if err := s.Tickets.CacheEpicChildren("EPIC-1", children); err != nil {
		t.Fatalf("CacheEpicChildren: %v", err)
	}

	got, err := s.Tickets.GetAllCachedEpicChildren()
	if err != nil {
		t.Fatalf("GetAllCachedEpicChildren: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(got))
	}
	if len(got["EPIC-1"]) != 2 {
		t.Fatalf("expected 2 children, got %d", len(got["EPIC-1"]))
	}
	if got["EPIC-1"][0].EpicID == nil || *got["EPIC-1"][0].EpicID != "EPIC-1" {
		t.Errorf("child EpicID = %v, want EPIC-1", got["EPIC-1"][0].EpicID)
	}
}

func TestCacheEpicChildren_ReplacesPerEpic(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "EPIC-1", Summary: "Epic 1", Type: "Epic", CreatedAt: now, UpdatedAt: now},
		{ID: "EPIC-2", Summary: "Epic 2", Type: "Epic", CreatedAt: now, UpdatedAt: now},
	})

	_ = s.Tickets.CacheEpicChildren("EPIC-1", []model.Ticket{
		{ID: "OLD-1", Summary: "old", CreatedAt: now, UpdatedAt: now},
	})
	_ = s.Tickets.CacheEpicChildren("EPIC-2", []model.Ticket{
		{ID: "OTHER-1", Summary: "other", CreatedAt: now, UpdatedAt: now},
	})
	_ = s.Tickets.CacheEpicChildren("EPIC-1", []model.Ticket{
		{ID: "NEW-1", Summary: "new", CreatedAt: now, UpdatedAt: now},
	})

	got, _ := s.Tickets.GetAllCachedEpicChildren()
	if len(got["EPIC-1"]) != 1 || got["EPIC-1"][0].ID != "NEW-1" {
		t.Errorf("EPIC-1 children = %+v, want [NEW-1]", got["EPIC-1"])
	}
	if len(got["EPIC-2"]) != 1 || got["EPIC-2"][0].ID != "OTHER-1" {
		t.Errorf("EPIC-2 should be unchanged, got %+v", got["EPIC-2"])
	}
}

func TestCacheEpicChildren_SeparateFromMainTickets(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "EPIC-1", Summary: "An epic", Type: "Epic", CreatedAt: now, UpdatedAt: now},
		{ID: "TASK-1", Summary: "A task", Type: "Task", CreatedAt: now, UpdatedAt: now},
	})
	_ = s.Tickets.CacheEpicChildren("EPIC-1", []model.Ticket{
		{ID: "CHILD-1", Summary: "child", CreatedAt: now, UpdatedAt: now},
	})

	tickets, _ := s.Tickets.GetCachedTickets()
	if len(tickets) != 2 {
		t.Fatalf("expected 2 main tickets, got %d", len(tickets))
	}
	ids := map[string]bool{}
	for _, t := range tickets {
		ids[t.ID] = true
	}
	if !ids["EPIC-1"] || !ids["TASK-1"] {
		t.Errorf("main list should contain EPIC-1 and TASK-1, got %v", ids)
	}
	if ids["CHILD-1"] {
		t.Error("CHILD-1 should not appear in main list")
	}
}

func TestGetCachedTickets_EpicsHaveIsEpicFlag(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "EPIC-1", Summary: "an epic", Type: "Epic", CreatedAt: now, UpdatedAt: now},
		{ID: "TASK-1", Summary: "a task", Type: "Task", CreatedAt: now, UpdatedAt: now},
	})

	got, _ := s.Tickets.GetCachedTickets()
	epicFound, taskFound := false, false
	for _, t := range got {
		if t.ID == "EPIC-1" && t.IsEpic() {
			epicFound = true
		}
		if t.ID == "TASK-1" && !t.IsEpic() {
			taskFound = true
		}
	}
	if !epicFound {
		t.Error("EPIC-1 should have IsEpic()=true")
	}
	if !taskFound {
		t.Error("TASK-1 should have IsEpic()=false")
	}
}

func TestCacheTickets_PreservesEpicChildren(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "EPIC-1", Summary: "epic", Type: "Epic", CreatedAt: now, UpdatedAt: now},
	})
	_ = s.Tickets.CacheEpicChildren("EPIC-1", []model.Ticket{
		{ID: "CHILD-1", Summary: "child", Type: "Task", CreatedAt: now, UpdatedAt: now},
	})

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "EPIC-1", Summary: "epic updated", Type: "Epic", CreatedAt: now, UpdatedAt: now},
	})

	children, _ := s.Tickets.GetAllCachedEpicChildren()
	if len(children["EPIC-1"]) != 1 || children["EPIC-1"][0].ID != "CHILD-1" {
		t.Errorf("epic children should survive re-cache, got %+v", children["EPIC-1"])
	}
}
