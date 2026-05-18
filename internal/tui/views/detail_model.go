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
	helpModel       help.Model
	availableHeight int

	comments        []model.Comment
	commentsTotal   int
	commentsLoading bool
	commentsError   error

	commentInput *CommentInputModel
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
) ActiveModel {
	w, h := style.App.GetFrameSize()

	availableHeight := height - h - footerHeight
	contentWidth := width - w

	vp := viewport.New(
		viewport.WithWidth(contentWidth),
		viewport.WithHeight(availableHeight),
	)

	markdown := getContent(ticket, contentWidth)

	vp.SetContent(getMetaData(ticket, contentWidth) + markdown + "\n" + commentsLoadingView())

	helpModel := help.New()
	helpModel.SetWidth(contentWidth)

	return &DetailModel{
		ticket:          ticket,
		viewport:        vp,
		helpModel:       helpModel,
		availableHeight: availableHeight,
		commentsLoading: true,
		commentInput:    NewCommentInput(contentWidth),
	}
}

func (m *DetailModel) Update(msg tea.KeyPressMsg) (ActiveModel, tea.Cmd) {
	if m.commentInput.Visible() {
		var cmd tea.Cmd
		m.commentInput, cmd = m.commentInput.handleKey(msg)
		m.AdjustViewportHeight()
		return m, cmd
	}

	switch {
	case key.Matches(msg, keymaps.DefaultKeyMap.GoBack):
		return m, func() tea.Msg {
			return GoToListMsg{}
		}

	case key.Matches(msg, keymaps.DefaultKeyMap.AddComment):
		cmd := m.commentInput.Show(m.ticket.ID)
		m.AdjustViewportHeight()
		return m, cmd

	case key.Matches(msg, keymaps.DefaultKeyMap.ToggleHelp):
		m.helpModel.ShowAll = !m.helpModel.ShowAll
		m.AdjustViewportHeight()
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
				ticket.UpdatedAt.Format(TimestampFmt),
			),
		),
	)

	sb.WriteString("\n")

	sb.WriteString(
		dimStyle.Render(
			fmt.Sprintf(
				"Created:  %s",
				ticket.CreatedAt.Format(TimestampFmt),
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

func (m *DetailModel) UpdateTags(ticket model.Ticket) {
	m.ticket = ticket
	m.refreshContent()
}

func (m *DetailModel) Ticket() model.Ticket {
	return m.ticket
}

func (m *DetailModel) SetComments(comments []model.Comment, total int) {
	m.comments = comments
	m.commentsTotal = total
	m.commentsLoading = false
	m.commentsError = nil
	m.refreshContent()
}

func (m *DetailModel) SetCommentsError(err error) {
	m.commentsError = err
	m.commentsLoading = false
	m.refreshContent()
}

func (m *DetailModel) CommentInput() *CommentInputModel {
	return m.commentInput
}

func (m *DetailModel) AdjustViewportHeight() {
	h := m.availableHeight
	if m.helpModel.ShowAll {
		h -= keymaps.DefaultKeyMap.GetFullHelpHeight()
	}
	h -= m.commentInput.Height()
	m.viewport.SetHeight(h)
}

// UpdateMsg handles non-key messages (cursor blink, etc.) for the comment input.
func (m *DetailModel) UpdateMsg(msg tea.Msg) tea.Cmd {
	if !m.commentInput.Visible() {
		return nil
	}
	var cmd tea.Cmd
	m.commentInput, cmd = m.commentInput.Update(msg)
	return cmd
}

func (m *DetailModel) refreshContent() {
	contentWidth := m.viewport.Width()
	markdown := getContent(m.ticket, contentWidth)
	commentsSection := m.renderComments(contentWidth)
	m.viewport.SetContent(getMetaData(m.ticket, contentWidth) + markdown + "\n" + commentsSection)
}

func (m *DetailModel) View() tea.View {
	helpView := styleHelp(m.helpModel.View(keymaps.DefaultKeyMap))
	content := m.viewport.View() + "\n"
	if m.commentInput.Visible() {
		content += m.commentInput.View() + "\n"
	}
	content += helpView
	return tea.NewView(content)
}

func getContent(ticket model.Ticket, width int) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dracula"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return fmt.Sprintf("[render error: %v]\n\n%s", err, ticket.Markdown)
	}

	rendered, err := renderer.Render(ticket.Markdown)
	if err != nil {
		return fmt.Sprintf("[render error: %v]\n\n%s", err, ticket.Markdown)
	}

	return rendered
}

// --------- Styles Helpers ---------

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
		sb.WriteString(" ")
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
