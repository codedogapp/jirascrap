package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/model"
)

type AppModel struct {
	jiraClient *jira.Client
	tickets    []model.Ticket
	cursor     int
	selected   *model.Ticket
	loading    bool
	err        error
}

func NewApp(client *jira.Client) AppModel {
	return AppModel{
		jiraClient: client,
		loading:    true,
	}
}

func (m AppModel) Init() tea.Cmd {
	return func() tea.Msg {
		tickets, err := m.jiraClient.FetchTickets()
		if err != nil {
			return errMsg{err}
		}
		return ticketsFetchedMsg(tickets)
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ticketsFetchedMsg:
		m.tickets = msg
		m.loading = false
		return m, nil

	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.tickets)-1 {
				m.cursor++
			}

		case "enter":
			if len(m.tickets) > 0 {
				m.selected = &m.tickets[m.cursor]
			}

		case "esc":
			m.selected = nil
		}
	}

	return m, nil
}

func (m AppModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\nPress 'q' to quit.", m.err)
	}

	if m.loading {
		return "\nFetching Tickets from Jira... \n"
	}

	if m.selected != nil {
		return fmt.Sprintf("\nTicket: %s\n\n%s\n\nPress 'esc' to return, 'q' to quit.\n", m.selected.ID, m.selected.Markdown)
	}

	var b strings.Builder

	b.WriteString("Your Jira Tickets:\n\n")

	for i, ticket := range m.tickets {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		fmt.Fprintf(&b, "%s %s\n", cursor, ticket.ID)
	}

	b.WriteString("\nPress j/k or up/down to move. Press Enter to select. Press 'q' to quit.\n")

	return b.String()
}

func Run(client *jira.Client) error {
	app := NewApp(client)
	p := tea.NewProgram(app)
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
