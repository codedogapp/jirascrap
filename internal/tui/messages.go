package tui

import "github.com/codedogapp/jirascrap/internal/model"

type cachedTicketsLoadedMsg struct {
	tickets      []model.Ticket
	epicChildren map[string][]model.Ticket
}

type syncCompleteMsg struct {
	tickets      []model.Ticket
	epicChildren map[string][]model.Ticket
}

type syncErrorMsg struct {
	err error
}

type tagSavedMsg struct {
	id   string
	tags []string
}

type todoSavedMsg struct{}

type epicChildrenLoadedMsg struct {
	epicKey string
	tickets []model.Ticket
}

type epicChildrenErrorMsg struct {
	err error
}
