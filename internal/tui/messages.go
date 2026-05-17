package tui

import (
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/model"
)

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

type copilotLaunchedMsg struct {
	ticketID string
	err      error
}

type transitionsLoadedMsg struct {
	ticketID    string
	transitions []jira.Transition
}

type transitionsErrorMsg struct {
	err error
}

type statusTransitionCompleteMsg struct {
	ticketID          string
	newStatus         string
	newStatusCategory string
}

type statusTransitionErrorMsg struct {
	err error
}

type commentsLoadedMsg struct {
	ticketID string
	comments []model.Comment
	total    int
}

type commentsErrorMsg struct {
	ticketID string
	err      error
}
