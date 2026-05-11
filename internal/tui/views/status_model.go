package views

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
)

type StatusTransitionMsg struct {
	TicketID   string
	Transition jira.Transition
}

type TransitionItem struct {
	jira.Transition
}

func (i TransitionItem) FilterValue() string { return i.Name }

type StatusModel struct {
	ticket        model.Ticket
	list          list.Model
	visible       bool
	loading       bool
	contentWidth  int
	contentHeight int
}

func NewStatusModel(contentWidth, contentHeight int) *StatusModel {
	l := newTransitionList(contentWidth, 0)
	return &StatusModel{
		list:          l,
		contentWidth:  contentWidth,
		contentHeight: contentHeight,
	}
}

func newTransitionList(width, itemCount int) list.Model {
	w := width / RatioWidth
	if w < 30 {
		w = 30
	}
	delegate := transitionDelegate{}
	h := statusListHeightForItems(itemCount)
	l := list.New(nil, delegate, w, h)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	return l
}

func statusListHeightForItems(n int) int {
	if n <= 0 {
		return 1
	}
	return n
}

func (m *StatusModel) Show(ticket model.Ticket) {
	m.ticket = ticket
	m.list.SetItems(nil)
	m.visible = true
	m.loading = true
}

func (m *StatusModel) Hide() {
	m.visible = false
	m.loading = false
}

func (m *StatusModel) IsVisible() bool {
	return m.visible
}

func (m *StatusModel) SetTransitions(transitions []jira.Transition) {
	items := make([]list.Item, len(transitions))
	for i, t := range transitions {
		items[i] = TransitionItem{t}
	}
	m.list.SetHeight(statusListHeightForItems(len(transitions)))
	m.list.SetItems(items)
	m.loading = false
}

func (m *StatusModel) SetSize(contentWidth, contentHeight int) {
	m.contentWidth = contentWidth
	m.contentHeight = contentHeight
	w := contentWidth / RatioWidth
	if w < 30 {
		w = 30
	}
	m.list.SetSize(w, statusListHeightForItems(len(m.list.Items())))
}

func (m *StatusModel) Update(msg tea.KeyPressMsg) tea.Cmd {
	if m.loading {
		return nil
	}

	if key.Matches(msg, keymaps.DefaultStatusKeyMap.Confirm) {
		if i, ok := m.list.SelectedItem().(TransitionItem); ok {
			ticketID := m.ticket.ID
			transition := i.Transition
			m.Hide()
			return func() tea.Msg {
				return StatusTransitionMsg{
					TicketID:   ticketID,
					Transition: transition,
				}
			}
		}
		return nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

func (m *StatusModel) View() *lipgloss.Layer {
	if !m.visible {
		return nil
	}

	var content string
	if m.loading {
		content = lipgloss.NewStyle().
			Foreground(grey).
			Italic(true).
			Padding(1, 1).
			Render("loading…")
	} else if len(m.list.Items()) == 0 {
		content = lipgloss.NewStyle().
			Foreground(grey).
			Italic(true).
			Padding(1, 1).
			Render("no transitions")
	} else {
		content = m.list.View()
	}

	currentStatus := lipgloss.NewStyle().
		Bold(true).
		Foreground(statusColor(m.ticket.StatusCategory)).
		Render("● " + m.ticket.Status)

	header := currentStatus + " " + lipgloss.NewStyle().
		Foreground(grey).
		Render("→")

	contentStyled := lipgloss.NewStyle().Padding(1, 1).Render(header + "\n" + content)

	popupView := tagViewPopUp.Render(contentStyled)
	popupWidth := lipgloss.Width(popupView)

	dashCount := popupWidth - lipgloss.Width("╭─ STATUS ╮")
	if dashCount < 0 {
		dashCount = 0
	}
	dashes := strings.Repeat("─", dashCount)
	topLine := topBorder.Render("╭─ ") +
		popUpTitle.Render("STATUS") +
		topBorder.Render(" "+dashes+"╮")

	full := topLine + "\n" + popupView

	return lipgloss.NewLayer(full).
		X(2).
		Y(3).
		Z(2)
}

type transitionDelegate struct{}

func (d transitionDelegate) Height() int {
	return 1
}
func (d transitionDelegate) Spacing() int {
	return 0
}

func (d transitionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d transitionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(TransitionItem)
	if !ok {
		return
	}

	dot := lipgloss.NewStyle().
		Foreground(statusColor(i.ToStatusCategory)).
		Render("● ")

	name := i.Name
	if i.Name != i.ToStatus {
		name = fmt.Sprintf("%s → %s", i.Name, i.ToStatus)
	}

	cursor := "  "
	style := lipgloss.NewStyle()
	if index == m.Index() {
		cursor = "▶ "
		style = style.Bold(true).Foreground(colPrimary)
	}

	fmt.Fprintf(w, "%s", style.Render(cursor+dot+name))
}

// StatusKeyMap is used internally; the actual key routing happens in app.go.
var _ = keymaps.DefaultStatusKeyMap
