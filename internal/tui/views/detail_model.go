package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/model"
)

type DetailModel struct {
	ticket   model.Ticket
	viewport viewport.Model
}

type (
	GoToListMsg struct{}
	TaggingMsg  struct {
		Ticket model.Ticket
	}
)

func NewDetailModel(ticket model.Ticket, width int, height int) ActiveModel {
	footerHeight := 2
	vp := viewport.New(viewport.WithWidth(width-2), viewport.WithHeight(height-footerHeight-1))

	markdown := getContent(ticket)
	renderer, _ := glamour.NewTermRenderer(glamour.WithStandardStyle("dracula"), glamour.WithWordWrap(width-4))
	rendered, _ := renderer.Render(markdown)

	vp.SetContent(getHeader(ticket, width-4) + rendered)
	return &DetailModel{ticket: ticket, viewport: vp}
}

func (m *DetailModel) Update(msg tea.KeyPressMsg) (ActiveModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg {
			return GoToListMsg{}
		}
	case "t":
		return m, func() tea.Msg {
			return TaggingMsg{Ticket: m.ticket}
		}
	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
}

func getHeader(ticket model.Ticket, width int) string {
	var sb strings.Builder

	title := titleStyle.Render(ticket.ID) + ticket.Summary
	sb.WriteString(title + "\n\n")

	// Status / priority
	statusC := statusColor(ticket.StatusCategory)
	priorityC := priorityColor(ticket.Priority)
	sb.WriteString(lipgloss.NewStyle().Foreground(statusC).Bold(true).Padding(0, 2).Render("● " + ticket.Status))
	if ticket.Priority != "" {
		sb.WriteString("  ")
		sb.WriteString(lipgloss.NewStyle().Foreground(priorityC).Render("▲ " + ticket.Priority))
	}
	sb.WriteString("\n")

	if ticket.Reporter != "" {
		sb.WriteString(dimStyle.Render("Reporter: ") + ticket.Reporter + "\n")
	}
	sb.WriteString(dimStyle.Render(fmt.Sprintf("Updated:  %s", ticket.UpdatedAt.Format("2006-01-02 15:04"))) + "\n")
	sb.WriteString(dimStyle.Render(fmt.Sprintf("Created:  %s", ticket.CreatedAt.Format("2006-01-02 15:04"))) + "\n")

	// Tags
	sb.WriteString("\n")
	if len(ticket.Tags) > 0 {
		sb.WriteString(dimStyle.Render("Tags: "))
		for _, t := range ticket.Tags {
			sb.WriteString(tagStyle.Render(t) + " ")
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString(dimStyle.Render("Tags: —") + "\n")
	}

	sb.WriteString(dimStyle.Render(strings.Repeat("─", width-4)) + "\n")

	return "\n" + sb.String()
}

func (m *DetailModel) View() tea.View {
	return tea.NewView(m.viewport.View() + "\n" + getFooter())
}

func getContent(ticket model.Ticket) string {
	return ticket.Markdown
}

func getFooter() string {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4A4A4A"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3C3C3C"))

	var sb strings.Builder

	separator := " • "

	sb.WriteString(keyStyle.Render("↑/↓ "))
	sb.WriteString(descStyle.Render("scroll"))
	sb.WriteString(sepStyle.Render(separator))

	sb.WriteString(keyStyle.Render("esc "))
	sb.WriteString(descStyle.Render("back"))
	sb.WriteString(sepStyle.Render(separator))

	sb.WriteString(keyStyle.Render("t "))
	sb.WriteString(descStyle.Render("tag"))
	sb.WriteString(sepStyle.Render(separator))

	sb.WriteString(keyStyle.Render("q "))
	sb.WriteString(descStyle.Render("quit"))

	return lipgloss.NewStyle().
		MarginLeft(2).
		MarginTop(1).
		Render(sb.String())
}
