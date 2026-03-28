package tui

import "github.com/codedogapp/jirascrap/internal/model"

type (
	ticketsFetchedMsg []model.Ticket
	errMsg            struct{ err error }
)

func (e errMsg) Error() string {
	return e.err.Error()
}
