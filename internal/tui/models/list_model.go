package tuimodels

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/codedogapp/jirascrap/internal/model"
)

type ListModel struct {
	list list.Model
}

type TicketItem struct {
	Ticket model.Ticket
}

func NewListModel(tickets []model.Ticket) ListModel {
	items := make([]list.Item, len(tickets))
	for i, t := range tickets {
		items[i] = TicketItem{Ticket: t}
	}

	l := list.New(items, list.NewDefaultDelegate(), 40, 40)
	l.Title = "Jira Tickets"

	return ListModel{list: l}
}

func (m *ListModel) SetItems(items []list.Item) {
	m.list.SetItems(items)
}

func (m *ListModel) SetSize(width int, height int) {
	m.list.SetSize(width, height)
}

func (i TicketItem) Title() string {
	return i.Ticket.ID
}

func (i TicketItem) Description() string {
	return i.Ticket.Summary
}

func (i TicketItem) FilterValue() string {
	return i.Ticket.ID + i.Ticket.Summary
}

type SelectTicketMsg struct {
	Ticket model.Ticket
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if i, ok := m.list.SelectedItem().(TicketItem); ok {
				return m, func() tea.Msg {
					return SelectTicketMsg(i)
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ListModel) View() string {
	return m.list.View()
}
