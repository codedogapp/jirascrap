package views

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
)

type TodosChangedMsg struct {
	TicketID string
	Todos    []model.Todo
}

type TodoItem struct {
	model.Todo
}

func (i TodoItem) FilterValue() string { return i.Title }

type TodoModel struct {
	list          list.Model
	help          help.Model
	textInput     textinput.Model
	ticketID      string
	adding        bool
	visible       bool
	contentWidth  int
	contentHeight int
}

const todoHeightRatio = 2

func NewTodoModel(contentWidth, contentHeight int, ticketID string, todos []model.Todo) *TodoModel {
	w := contentWidth / RatioWidth
	h := contentHeight / todoHeightRatio

	items := make([]list.Item, len(todos))
	for i, t := range todos {
		items[i] = TodoItem{t}
	}

	delegate := todoDelegate{}
	l := list.New(items, delegate, w, h)
	l.Title = ""
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)

	ti := textinput.New()
	ti.Placeholder = "New todo..."
	ti.SetWidth(w - 4)

	helpModel := help.New()
	helpModel.SetWidth(w)

	return &TodoModel{
		list:          l,
		help:          helpModel,
		textInput:     ti,
		ticketID:      ticketID,
		contentWidth:  contentWidth,
		contentHeight: contentHeight,
	}
}

func (m *TodoModel) Show() {
	m.visible = true
}

func (m *TodoModel) Hide() {
	m.visible = false
	m.adding = false
}

func (m *TodoModel) IsVisible() bool {
	return m.visible
}

func (m *TodoModel) IsAdding() bool {
	return m.adding
}

func (m *TodoModel) Update(msg tea.KeyPressMsg) tea.Cmd {
	if m.adding {
		return m.updateAdding(msg)
	}
	return m.updateNormal(msg)
}

func (m *TodoModel) updateAdding(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keymaps.DefaultTodoKeyMap.Confirm):
		value := strings.TrimSpace(m.textInput.Value())

		if value != "" {
			items := m.list.Items()
			items = append(items, TodoItem{model.Todo{Title: value}})
			m.list.SetItems(items)
		}

		m.textInput.SetValue("")
		m.adding = false

		if value == "" {
			return nil
		}

		return m.todosChangedCmd()
	case key.Matches(msg, keymaps.DefaultTodoKeyMap.Cancel):
		m.textInput.SetValue("")
		m.adding = false
		return nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return cmd
}

func (m *TodoModel) updateNormal(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keymaps.DefaultTodoKeyMap.Add):
		m.adding = true
		m.textInput.SetValue("")
		return m.textInput.Focus()
	case key.Matches(msg, keymaps.DefaultTodoKeyMap.Toggle):
		if i, ok := m.list.SelectedItem().(TodoItem); ok {
			idx := m.list.Index()
			i.Done = !i.Done
			items := m.list.Items()
			items[idx] = i
			m.list.SetItems(items)
		}
		return m.todosChangedCmd()
	case key.Matches(msg, keymaps.DefaultTodoKeyMap.Delete):
		idx := m.list.Index()
		items := m.list.Items()
		if idx >= 0 && idx < len(items) {
			items = append(items[:idx], items[idx+1:]...)
			m.list.SetItems(items)
		}
		return m.todosChangedCmd()
	case key.Matches(msg, keymaps.DefaultKeyMap.Quit):
		return nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

func (m *TodoModel) todosChangedCmd() tea.Cmd {
	items := m.list.Items()
	todos := make([]model.Todo, 0, len(items))
	for _, item := range items {
		if t, ok := item.(TodoItem); ok {
			todos = append(todos, t.Todo)
		}
	}
	ticketID := m.ticketID
	return func() tea.Msg {
		return TodosChangedMsg{TicketID: ticketID, Todos: todos}
	}
}

func (m *TodoModel) UpdateMsg(msg tea.Msg) tea.Cmd {
	if m.adding {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return cmd
	}
	return nil
}

func (m *TodoModel) SetSize(contentWidth, contentHeight int) {
	m.contentWidth = contentWidth
	m.contentHeight = contentHeight
	w := contentWidth / RatioWidth
	h := contentHeight / todoHeightRatio
	m.list.SetSize(w, h)
	m.textInput.SetWidth(w - 4)
}

func (m *TodoModel) View() *lipgloss.Layer {
	if !m.visible {
		return nil
	}

	content := m.list.View()
	if m.adding {
		content += "\n" + m.textInput.View()
	}
	content += "\n" + m.help.View(keymaps.DefaultTodoKeyMap)

	contentStyled := lipgloss.NewStyle().Padding(1, 1).Render(content)

	popupView := tagViewPopUp.Width(m.contentWidth).Render(contentStyled)
	popupWidth := lipgloss.Width(popupView)

	dashCount := popupWidth - lipgloss.Width("╭─ TODO ╮")
	dashCount = max(dashCount, 0)

	dashes := strings.Repeat("─", dashCount)
	topLine := topBorder.Render("╭─ ") +
		popUpTitle.Render("TODO") +
		topBorder.Render(" "+dashes+"╮")

	full := topLine + "\n" + popupView

	return lipgloss.NewLayer(full).
		X((m.contentWidth - popupWidth) / 2).
		Y((m.contentHeight - m.contentHeight/todoHeightRatio) / 2).
		Z(1)
}

type todoDelegate struct{}

func (d todoDelegate) Height() int {
	return 1
}

func (d todoDelegate) Spacing() int {
	return 0
}

func (d todoDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d todoDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(TodoItem)
	if !ok {
		return
	}

	checkbox := "[ ] "
	style := lipgloss.NewStyle()
	if i.Done {
		checkbox = "[x] "
		style = style.Strikethrough(true).Foreground(grey)
	}

	cursor := "  "
	if index == m.Index() {
		cursor = "> "
		if !i.Done {
			style = style.Foreground(colSecondary).Bold(true)
		}
	}

	fmt.Fprintf(w, "%s", style.Render(cursor+checkbox+i.Title))
}
