package views

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/model"
)

type ListModel struct {
	list    list.Model
	tickets []model.Ticket
	style   lipgloss.Style
}

type TicketItem struct {
	Ticket model.Ticket
}

type SelectTicketMsg struct {
	Ticket model.Ticket
}

type ticketDelegate struct {
	list.DefaultDelegate
}

func NewListModel(tickets []model.Ticket, style lipgloss.Style) *ListModel {
	items := getItemsList(tickets)

	l := list.New(
		items,
		ticketDelegate{list.NewDefaultDelegate()},
		0,
		0,
	)
	l.Title = "Jira Tickets"
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("d"),
				key.WithHelp("d", "debug view"),
			),
		}
	}

	return &ListModel{
		list:    l,
		tickets: tickets,
		style:   style,
	}
}

func (m *ListModel) StartSpinner() tea.Cmd {
	return m.list.StartSpinner()
}

func (m *ListModel) SetSize(width int, height int) {
	w, h := m.style.GetFrameSize()
	m.list.SetSize(width-w, height-h)
}

func (i TicketItem) FilterValue() string {
	return i.Ticket.ID +
		" " +
		i.Ticket.Summary +
		" " +
		i.Ticket.Priority +
		" " +
		i.Ticket.Status
}

func (m *ListModel) Update(msg tea.KeyPressMsg) (ActiveModel, tea.Cmd) {
	if msg.String() == "enter" && m.list.FilterState() != list.Filtering {
		if i, ok := m.list.SelectedItem().(TicketItem); ok {
			return m, func() tea.Msg {
				return SelectTicketMsg(i)
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *ListModel) UpdateMsg(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

func (m *ListModel) View() tea.View {
	return tea.NewView(m.list.View())
}

func (m *ListModel) UpdateTicket(id string, tags []string) (*model.Ticket, error) {
	var updatedTicket *model.Ticket
	for i, t := range m.tickets {
		if t.ID == id {
			ticket := &m.tickets[i]
			ticket.Tags = tags
			updatedTicket = ticket
			break
		}
	}

	if updatedTicket == nil {
		return nil, fmt.Errorf("ticket %s not found", id)
	}

	m.SetItems(m.tickets)

	return updatedTicket, nil
}

func getItemsList(tickets []model.Ticket) []list.Item {
	items := make([]list.Item, len(tickets))
	for i, t := range tickets {
		items[i] = TicketItem{Ticket: t}
	}

	return items
}

func (m *ListModel) SetItems(tickets []model.Ticket) {
	items := getItemsList(tickets)
	m.tickets = tickets
	m.list.SetItems(items)
}

func (m *ListModel) Initialize(tickets []model.Ticket) {
	m.SetItems(tickets)
	m.StopSpinner()
}

func (m *ListModel) StopSpinner() {
	m.list.StopSpinner()
}

func (d ticketDelegate) Height() int {
	return 2
}

func (d ticketDelegate) Spacing() int {
	return 1
}

func (d ticketDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d ticketDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(TicketItem)
	if !ok {
		return
	}

	var sb strings.Builder

	sb.WriteString(
		lipgloss.NewStyle().
			Foreground(statusColor(i.Ticket.StatusCategory)).
			Render("● "),
	)

	if i.Ticket.Priority != "" {
		sb.WriteString(
			lipgloss.NewStyle().
				Foreground(priorityColor(i.Ticket.Priority)).
				Render("▲ "),
		)
	}

	sb.WriteString(i.Ticket.ID)

	titleStyle := d.Styles.NormalTitle
	descStyle := d.Styles.NormalDesc
	if index == m.Index() {
		titleStyle = d.Styles.SelectedTitle
		descStyle = d.Styles.SelectedDesc
	}

	_, _ = fmt.Fprintf(
		w,
		"%s\n%s",
		titleStyle.Render(sb.String()),
		descStyle.Render(i.Ticket.Summary),
	)
}
