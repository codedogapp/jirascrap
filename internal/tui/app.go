package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/store"
)

type AppModel struct {
	// Dependencies
	jiraClient *jira.Client
	store      store.MetaStore

	// Data
	tickets  []model.Ticket
	selected *model.Ticket

	// TUI Elements
	tagInput textinput.Model

	// State
	state   sessionState
	cursor  int
	loading bool
	err     error
}

type sessionState int

const (
	listView sessionState = iota
	detailView
	taggingView
)

func NewApp(client *jira.Client, s store.MetaStore) *AppModel {
	ti := textinput.New()
	ti.Placeholder = "tag1, tag2..."
	ti.Focus()

	return &AppModel{
		jiraClient: client,
		store:      s,
		tagInput:   ti,
		state:      listView,
		loading:    true,
	}
}

func (m *AppModel) Init() tea.Cmd {
	return func() tea.Msg {
		tickets, err := m.jiraClient.FetchTickets()
		if err != nil {
			return errMsg{err}
		}

		localData, err := m.store.GetAllMeta()
		if err != nil {
			return errMsg{err}
		}

		for i, t := range tickets {
			meta, exists := localData[t.ID]
			if exists {
				tickets[i].Tags = meta.Tags
			}
		}

		return ticketsFetchedMsg(tickets)
	}
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ticketsFetchedMsg:
		m.tickets = msg
		m.loading = false
		return m, nil

	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil

	case tagSavedMsg:
		for i, t := range m.tickets {
			if t.ID == msg.id {
				m.tickets[i].Tags = msg.tags
				if m.selected != nil && m.selected.ID == msg.id {
					m.selected.Tags = msg.tags
				}
			}
		}
		return m, nil

	case tea.KeyMsg:
		cmd := handleQuit(m, msg)
		if cmd != nil {
			return m, cmd
		}
		return m.getHandler().handleKey(m, msg)
	}

	return m, nil
}

func (m *AppModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\nPress 'q' to quit.", m.err)
	}

	if m.loading {
		return "\nFetching Tickets from Jira... \n"
	}

	return m.getHandler().view(m)
}

func (m *AppModel) getHandler() stateHandler {
	switch m.state {
	case detailView:
		return detailHandler{}
	case taggingView:
		return taggingHandler{}
	default:
		return listHandler{}
	}
}

func (m *AppModel) saveTagsCmd(id string, tags []string) tea.Cmd {
	return func() tea.Msg {
		err := m.store.SaveMeta(id, tags, "")
		if err != nil {
			return errMsg{err}
		}

		return tagSavedMsg{id: id, tags: tags}
	}
}

func Run(client *jira.Client, s store.MetaStore) error {
	app := NewApp(client, s)
	p := tea.NewProgram(app)
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
