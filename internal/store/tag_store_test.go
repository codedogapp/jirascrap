package store

import (
	"testing"
	"time"

	"github.com/codedogapp/jirascrap/internal/model"
)

func TestSaveMeta_TagsJoinedInGetCachedTickets(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "TICK-1", Summary: "First", Type: "Task", CreatedAt: now, UpdatedAt: now},
	})
	if err := s.Tags.SaveTags("TICK-1", []string{"bug", "urgent"}); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	tickets, err := s.Tickets.GetCachedTickets()
	if err != nil {
		t.Fatalf("GetCachedTickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
	if len(tickets[0].Tags) != 2 {
		t.Errorf("tags = %v, want [bug urgent]", tickets[0].Tags)
	}
}

func TestSaveMeta_Overwrite(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "TICK-1", Summary: "First", Type: "Task", CreatedAt: now, UpdatedAt: now},
	})
	_ = s.Tags.SaveTags("TICK-1", []string{"old-tag"})
	_ = s.Tags.SaveTags("TICK-1", []string{"new-tag"})

	tickets, _ := s.Tickets.GetCachedTickets()
	if len(tickets[0].Tags) != 1 || tickets[0].Tags[0] != "new-tag" {
		t.Errorf("tags = %v, want [new-tag]", tickets[0].Tags)
	}
}

func TestSaveMeta_EmptyTags(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "TICK-1", Summary: "First", Type: "Task", CreatedAt: now, UpdatedAt: now},
	})
	_ = s.Tags.SaveTags("TICK-1", []string{"tag"})
	_ = s.Tags.SaveTags("TICK-1", []string{})

	tickets, _ := s.Tickets.GetCachedTickets()
	if tickets[0].Tags != nil {
		t.Errorf("tags = %v, want nil", tickets[0].Tags)
	}
}

func TestSaveMeta_MultipleTickets(t *testing.T) {
	s := setupTestDB(t)
	now := time.Now().Truncate(time.Second)

	_ = s.Tickets.CacheTickets([]model.Ticket{
		{ID: "TICK-1", Summary: "First", Type: "Task", CreatedAt: now, UpdatedAt: now},
		{ID: "TICK-2", Summary: "Second", Type: "Task", CreatedAt: now, UpdatedAt: now},
	})
	_ = s.Tags.SaveTags("TICK-1", []string{"a"})
	_ = s.Tags.SaveTags("TICK-2", []string{"b", "c"})

	tickets, _ := s.Tickets.GetCachedTickets()
	tagged := 0
	for _, t := range tickets {
		if len(t.Tags) > 0 {
			tagged++
		}
	}
	if tagged != 2 {
		t.Fatalf("expected 2 tagged tickets, got %d", tagged)
	}
}

func TestGetUniqueTags(t *testing.T) {
	s := setupTestDB(t)

	_ = s.Tags.SaveTags("TICK-1", []string{"bug", "urgent"})
	_ = s.Tags.SaveTags("TICK-2", []string{"bug", "feature"})

	tags, err := s.Tags.GetUniqueTags()
	if err != nil {
		t.Fatalf("GetUniqueTags: %v", err)
	}
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

	tags, err := s.Tags.GetUniqueTags()
	if err != nil {
		t.Fatalf("GetUniqueTags: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected empty, got %v", tags)
	}
}
