package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type stateHandler interface {
	handleKey(m *AppModel, msg tea.KeyMsg) (tea.Model, tea.Cmd)
	view(m *AppModel) string
}

type (
	listHandler    struct{}
	taggingHandler struct{}
	detailHandler  struct{}
)

func handleQuit(m *AppModel, msg tea.KeyMsg) tea.Cmd {
	s := msg.String()

	if s == "ctrl+c" {
		return tea.Quit
	}

	if s == "q" && m.state != taggingView {
		return tea.Quit
	}

	return nil
}

func (h listHandler) handleKey(m *AppModel, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (h taggingHandler) handleKey(m *AppModel, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "esc":
		m.state = detailView
		return m, nil
	case "enter":
		tags := strings.Split(m.tagInput.Value(), ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		m.state = detailView
		return m, m.saveTagsCmd(m.selected.ID, tags)
	}

	m.tagInput, cmd = m.tagInput.Update(msg)
	return m, cmd
}

func (h detailHandler) handleKey(m *AppModel, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = listView
	case "t":
		m.state = taggingView
		m.tagInput.SetValue(strings.Join(m.selected.Tags, ", "))
		m.tagInput.Focus()
	}
	return m, nil
}

// TODO: Empty stub
func (h listHandler) view(m *AppModel) string {
	return ""
}

func (h detailHandler) view(m *AppModel) string {
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
		m.selected.ID,
		m.selected.Summary,
		m.selected.Reporter,
		m.selected.UpdatedAt.Format("Jan 02, 2006 15:04"),
		m.selected.CreatedAt.Format("Jan 02, 2006 15:04"),
		strings.Join(m.selected.Tags, ", "),
		m.selected.Markdown,
	)
}

func (h taggingHandler) view(m *AppModel) string {
	return fmt.Sprintf("Tagging %s:\n\n%s", m.selected.ID, m.tagInput.View())
}
