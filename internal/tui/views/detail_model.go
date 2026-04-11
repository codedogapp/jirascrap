package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
)

type DetailModel struct {
	ticket          model.Ticket
	viewport        viewport.Model
	help            help.Model
	availableHeight int
	tagModel        *TagModel
}

type GoToListMsg struct{}

const (
	footerHeight     = 2
	separatorPadding = 4
)

func NewDetailModel(
	ticket model.Ticket,
	width int,
	height int,
	style Styles,
	allTags []string,
) ActiveModel {
	w, h := style.App.GetFrameSize()

	availableHeight := height - h - footerHeight
	contentWidth := width - w

	vp := viewport.New(
		viewport.WithWidth(contentWidth),
		viewport.WithHeight(availableHeight),
	)

	markdown := getContent(ticket, contentWidth)

	vp.SetContent(getMetaData(ticket, contentWidth) + markdown)

	help := help.New()
	help.SetWidth(contentWidth)

	return &DetailModel{
		ticket:          ticket,
		viewport:        vp,
		help:            help,
		availableHeight: availableHeight,
		tagModel:        NewTagModel(contentWidth, availableHeight, allTags),
	}
}

func (m *DetailModel) Update(msg tea.KeyPressMsg) (ActiveModel, tea.Cmd) {
	if m.tagModel.IsVisible() {
		switch {
		case key.Matches(msg, keymaps.DefaultKeyMap.GoBack):
			m.tagModel.Hide()
			return m, nil

		case key.Matches(msg, keymaps.DefaultKeyMap.Select):
			id := m.tagModel.TicketID()
			tags := m.tagModel.CurrentTags()
			m.tagModel.Hide()
			return m, func() tea.Msg {
				return TagsFilledMsg{ID: id, Tags: tags}
			}

		default:
			return m, m.tagModel.Update(msg)
		}
	}

	switch {
	case key.Matches(msg, keymaps.DefaultKeyMap.GoBack):
		return m, func() tea.Msg {
			return GoToListMsg{}
		}

	case key.Matches(msg, keymaps.DefaultKeyMap.ToggleTagging):
		return m, m.tagModel.Show(m.ticket)

	case key.Matches(msg, keymaps.DefaultKeyMap.ToggleHelp):
		m.help.ShowAll = !m.help.ShowAll
		if m.help.ShowAll {
			m.viewport.SetHeight(m.availableHeight - keymaps.DefaultKeyMap.GetFullHelpHeight())
		} else {
			m.viewport.SetHeight(m.availableHeight)
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
}

func getMetaData(ticket model.Ticket, width int) string {
	var sb strings.Builder

	title := titleStyle.Render(ticket.ID) + ticket.Summary
	sb.WriteString(title + "\n\n")

	styleStatus(ticket.StatusCategory, ticket.Status, &sb)
	stylePriority(ticket.Priority, &sb)

	sb.WriteString("\n")

	if ticket.Reporter != "" {
		sb.WriteString(
			dimStyle.Render("Reporter: ") + ticket.Reporter + "\n",
		)
	}

	sb.WriteString(
		dimStyle.Render(
			fmt.Sprintf(
				"Updated:  %s",
				ticket.UpdatedAt.Format("2006-01-02 15:04"),
			),
		),
	)

	sb.WriteString("\n")

	sb.WriteString(
		dimStyle.Render(
			fmt.Sprintf(
				"Created:  %s",
				ticket.CreatedAt.Format("2006-01-02 15:04"),
			),
		),
	)

	sb.WriteString("\n")
	sb.WriteString("\n")

	// Tags
	if len(ticket.Tags) > 0 {
		sb.WriteString(dimStyle.Render("Tags: "))
		for _, t := range ticket.Tags {
			sb.WriteString(tagStyle.Render(t) + " ")
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString(dimStyle.Render("Tags: —"))
		sb.WriteString("\n")
	}

	sb.WriteString(
		dimStyle.Render(
			strings.Repeat("─", width-separatorPadding),
		),
	)

	sb.WriteString("\n\n")

	return sb.String()
}

func (m *DetailModel) IsTagging() bool {
	return m.tagModel.IsVisible()
}

func (m *DetailModel) UpdateMsg(msg tea.Msg) tea.Cmd {
	if m.tagModel.IsVisible() {
		return m.tagModel.UpdateMsg(msg)
	}
	return nil
}

func (m *DetailModel) View() tea.View {
	helpView := styleHelp(m.help.View(keymaps.DefaultKeyMap))
	base := m.viewport.View() + "\n" + helpView

	if overlay := m.tagModel.View(); overlay != nil {
		return tea.NewView(
			lipgloss.NewCompositor(
				lipgloss.NewLayer(base),
				overlay,
			).Render(),
		)
	}

	return tea.NewView(base)
}

func getContent(ticket model.Ticket, width int) string {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dracula"),
		glamour.WithWordWrap(width),
	)

	rendered, _ := renderer.Render(ticket.Markdown)

	return rendered
}

// STYLES
func styleStatus(statusCategory string, status string, sb *strings.Builder) {
	statusC := statusColor(statusCategory)
	rendered := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		Foreground(statusC).
		Render("● " + status)

	sb.WriteString(rendered)
}

func stylePriority(priority string, sb *strings.Builder) {
	priorityC := priorityColor(priority)
	if priority != "" {
		sb.WriteString("  ")
		sb.WriteString(
			lipgloss.NewStyle().
				Foreground(priorityC).
				Render("▲ " + priority),
		)
	}
}

func styleHelp(help string) string {
	return lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Render(help)
}
