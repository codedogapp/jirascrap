package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	tickets  []string
	cursor   int
	selected string
}

func initModel() model {
	return model{
		tickets: []string{"Proj-123: FIX ME", "Proj-124: UPDATE ME"},
		cursor:  0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
			m.selected = m.tickets[m.cursor]
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.selected != "" {
		return fmt.Sprintf("\nYou selected %s\n Press 'q' to quit.\n", m.selected)
	}

	var b strings.Builder

	b.WriteString("Your Jira Tickets:\n\n")

	for i, ticket := range m.tickets {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		fmt.Fprintf(&b, "%s %s\n", cursor, ticket)
	}

	b.WriteString("\nPress j/k or up/down to move. Press Enter to select. Press 'q' to quit.\n")

	return b.String()
}

func main() {
	p := tea.NewProgram(initModel())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
