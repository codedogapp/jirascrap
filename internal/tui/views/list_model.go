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
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
)

type ListModel struct {
	list      list.Model
	ticketIdx map[string]model.Ticket
	style     lipgloss.Style
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
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			keymaps.DefaultKeyMap.ToggleTagging,
			keymaps.DefaultKeyMap.ToggleTodo,
			keymaps.DefaultKeyMap.ToggleStatus,
			keymaps.DefaultKeyMap.OpenInBrowser,
			keymaps.DefaultKeyMap.SendToCopilot,
		}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keymaps.DefaultKeyMap.ToggleTagging,
			keymaps.DefaultKeyMap.ToggleTodo,
			keymaps.DefaultKeyMap.ToggleStatus,
			keymaps.DefaultKeyMap.OpenInBrowser,
			keymaps.DefaultKeyMap.SendToCopilot,
			keymaps.DefaultKeyMap.Select,
			keymaps.DefaultKeyMap.Refresh,
		}
	}

	return &ListModel{
		list:      l,
		ticketIdx: buildTicketIndex(tickets),
		style:     style,
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
		i.Ticket.Status +
		" " +
		"#" + strings.Join(i.Ticket.Tags, " #")
}

func (m *ListModel) Update(msg tea.KeyPressMsg) (ActiveModel, tea.Cmd) {
	if key.Matches(msg, keymaps.DefaultKeyMap.Select) &&
		!m.IsFiltering() {
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

func (m *ListModel) FindTicket(id string) (model.Ticket, bool) {
	t, ok := m.ticketIdx[id]
	return t, ok
}

func (m *ListModel) Title() string {
	return m.list.Title
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
	m.ticketIdx = buildTicketIndex(tickets)
	m.list.SetItems(items)
}

func buildTicketIndex(tickets []model.Ticket) map[string]model.Ticket {
	idx := make(map[string]model.Ticket, len(tickets))
	for _, t := range tickets {
		idx[t.ID] = t
	}
	return idx
}

func (m *ListModel) Initialize(tickets []model.Ticket) {
	m.SetItems(tickets)
	m.StopSpinner()
}

func (m *ListModel) StopSpinner() {
	m.list.StopSpinner()
}

func (m *ListModel) IsFiltering() bool {
	return m.list.FilterState() == list.Filtering
}

func (m *ListModel) SetTitle(title string) {
	m.list.Title = title
}

func (m *ListModel) HasTickets() bool {
	return len(m.ticketIdx) > 0
}

func (m *ListModel) SelectedTicket() (model.Ticket, bool) {
	if i, ok := m.list.SelectedItem().(TicketItem); ok {
		return i.Ticket, true
	}
	return model.Ticket{}, false
}

func (d ticketDelegate) Height() int {
	return 2
}

func (d ticketDelegate) Spacing() int {
	return 1
}

func (d ticketDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(TicketItem)
	if !ok {
		return
	}

	var sb strings.Builder

	styleStatusDot(i.Ticket.StatusCategory, &sb)
	stylePriorityDot(i.Ticket.Priority, &sb)

	if i.Ticket.IsEpic() {
		styleEpicBolt(&sb)
	}

	sb.WriteString(i.Ticket.ID)

	styleTags(i.Ticket.Tags, &sb)

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

// --------- Styles Helpers ---------

func styleStatusDot(statusCategory string, sb *strings.Builder) {
	statusC := statusColor(statusCategory)
	rendered := lipgloss.NewStyle().
		Foreground(statusC).
		Render("● ")

	sb.WriteString(rendered)
}

func stylePriorityDot(priority string, sb *strings.Builder) {
	if priority == "" {
		return
	}
	priorityC := priorityColor(priority)
	rendered := lipgloss.NewStyle().
		Foreground(priorityC).
		Render("▲ ")

	sb.WriteString(rendered)
}

func styleEpicBolt(sb *strings.Builder) {
	sb.WriteString(epicBoltStyle.Render("⚡"))
}

func styleTags(tags []string, sb *strings.Builder) {
	if len(tags) > 0 {
		sb.WriteString(" | ")
		for _, t := range tags {
			sb.WriteString(
				tagListStyle.
					Render("#" + t + " "),
			)
		}
	}
}
