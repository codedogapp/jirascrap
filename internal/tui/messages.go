package tui

import "github.com/codedogapp/jirascrap/internal/model"

type (
	cachedTicketsLoadedMsg []model.Ticket
	syncCompleteMsg        []model.Ticket
)

type syncErrorMsg struct {
	err error
}

type tagSavedMsg struct {
	id   string
	tags []string
}

type todoSavedMsg struct{}
