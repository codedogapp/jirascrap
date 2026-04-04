package tuimodels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/codedogapp/jirascrap/internal/model"
)

type DetailModel struct {
	ticket model.Ticket
}

type (
	GoToListMsg struct{}
	TaggingMsg  struct {
		Ticket model.Ticket
	}
)

func NewDetailModel(ticket model.Ticket) DetailModel {
	return DetailModel{ticket: ticket}
}

func (m DetailModel) Update(msg tea.KeyMsg) (DetailModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg {
			return GoToListMsg{}
		}
	case "t":
		return m, func() tea.Msg {
			return TaggingMsg{Ticket: m.ticket}
		}
	}
	return m, nil
}

func (m DetailModel) View() string {
	return fmt.Sprintf(
		"Ticket   : %s\n"+
			"Summary  : %s\n"+
			"Reporter : %s\n"+
			"UpdatedAt: %v\n"+
			"CreatedAt: %v\n"+
			"Tags     : [%s]\n"+
			"\n"+
			"%s\n"+
			"\n"+
			"Press 'esc' to return, 'q' to quit.\n",
		m.ticket.ID,
		m.ticket.Summary,
		m.ticket.Reporter,
		m.ticket.UpdatedAt.Format("Jan 02, 2006 15:04"),
		m.ticket.CreatedAt.Format("Jan 02, 2006 15:04"),
		strings.Join(m.ticket.Tags, ", "),
		m.ticket.Markdown,
	)
}
