package tui

import "github.com/codedogapp/jirascrap/internal/model"

type (
	ticketsFetchedMsg []model.Ticket
)

type tagSavedMsg struct {
	id   string
	tags []string
}

type todoSavedMsg struct{}
