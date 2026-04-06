package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/store"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

// TODO: define global styling
// TODO: define key bindings
type AppModel struct {
	// Dependencies
	jiraClient *jira.Client
	store      store.MetaStore

	// State
	list        *views.ListModel
	activeModel views.ActiveModel
	debugModel  *views.DebugModel
	err         error

	// Size
	width  int
	height int
}

func NewApp(client *jira.Client, s store.MetaStore) *AppModel {
	listModel := views.NewListModel([]model.Ticket{})
	debugModel := views.NewDebugModel(0, 0)
	return &AppModel{
		jiraClient:  client,
		store:       s,
		list:        listModel,
		activeModel: listModel,
		debugModel:  debugModel,
	}
}

func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.list.StartSpinner(),
		func() tea.Msg {
			tickets, err := m.jiraClient.FetchTickets()
			if err != nil {
				return views.ErrMsg{Err: err}
			}

			localData, err := m.store.GetAllMeta()
			if err != nil {
				return views.ErrMsg{Err: err}
			}

			for i, t := range tickets {
				meta, exists := localData[t.ID]
				if exists {
					tickets[i].Tags = meta.Tags
				}
			}

			return ticketsFetchedMsg(tickets)
		},
	)
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case ticketsFetchedMsg:
		return m.handleTicketsFetched(msg)

	case views.ErrMsg:
		return m.handleError(msg)

	case views.SelectTicketMsg:
		return m.handleSelectTicket(msg)

	case views.GoToListMsg:
		return m.handleGoToList(msg)

	case views.TaggingMsg:
		return m.handleTaggingMsg(msg)

	case views.TagsCancelledMsg:
		return m.handleTagsCancelled(msg)

	case views.TagsFilledMsg:
		return m.handleTagFilled(msg)

	case tagSavedMsg:
		return m.handleTagSaved(msg)

	case tea.KeyPressMsg:
		cmd := handleQuit(m, msg)
		if cmd != nil {
			return m, cmd
		}
		if consumed, cmd := handleDebug(m, msg); consumed {
			return m, cmd
		}
		m.activeModel, cmd = m.activeModel.Update(msg)
		return m, cmd

	default:
		if mu, ok := m.activeModel.(views.MsgUpdater); ok {
			return m, mu.UpdateMsg(msg)
		}
	}

	return m, nil
}

func (m *AppModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("\nError: %v\n\nPress 'q' to quit.", m.err))
	}

	base := m.activeModel.View()

	debug := m.debugModel.View()

	if debug != nil {
		return tea.NewView(
			lipgloss.NewCompositor(
				lipgloss.NewLayer(base.Content),
				debug,
			).Render(),
		)
	}

	return tea.NewView(lipgloss.NewStyle().Padding(1, 2).Render(base.Content))
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
