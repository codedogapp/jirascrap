package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/model"
)

type TagModel struct {
	tagInput textinput.Model
	ticket   model.Ticket
}

type (
	TagsCancelledMsg struct {
		Ticket model.Ticket
	}
	TagsFilledMsg struct {
		ID   string
		Tags []string
	}
)

func NewTagModel(ticket model.Ticket) *TagModel {
	ti := textinput.New()
	ti.Placeholder = "tag1, tag2, ..."
	ti.SetValue(strings.Join(ticket.Tags, ", "))
	ti.Focus()
	return &TagModel{
		tagInput: ti,
		ticket:   ticket,
	}
}

func (m *TagModel) Update(msg tea.KeyPressMsg) (ActiveModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg {
			return TagsCancelledMsg{
				Ticket: m.ticket,
			}
		}
	case "enter":
		tags := strings.Split(m.tagInput.Value(), ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		return m, func() tea.Msg {
			return TagsFilledMsg{
				ID:   m.ticket.ID,
				Tags: tags,
			}
		}
	}

	var cmd tea.Cmd
	m.tagInput, cmd = m.tagInput.Update(msg)
	return m, cmd
}

func (m *TagModel) View() tea.View {
	return tea.NewView(fmt.Sprintf("Tagging %s:\n\n%s", m.ticket.ID, m.tagInput.View()))
}
