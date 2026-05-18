package views

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/model"
)

const (
	commentInputHeight   = 4
	suggestionMaxVisible = 5
	maxCommentLength     = 5000
)

var (
	keyDown  = key.NewBinding(key.WithKeys("down"))
	keyUp    = key.NewBinding(key.WithKeys("up"))
	keyTab   = key.NewBinding(key.WithKeys("tab", "enter"))
	keyEsc   = key.NewBinding(key.WithKeys("esc"))
	keyEnter = key.NewBinding(key.WithKeys("enter"))
)

type CommentSubmitMsg struct {
	TicketID string
	Text     string
	Mentions map[string]string
}

type CommentCancelMsg struct{}

type UserSearchRequestMsg struct {
	Query string
}

type CommentInputModel struct {
	textarea    textarea.Model
	visible     bool
	suggestions []model.User
	showSuggest bool
	selectedIdx int
	mentions    map[string]string // displayName -> accountId
	ticketID    string
	width       int
	atQuery     string // current @query being typed
}

func NewCommentInput(width int) *CommentInputModel {
	ta := textarea.New()
	ta.Placeholder = "Write a comment... (@ to mention)"
	ta.ShowLineNumbers = false
	ta.SetWidth(width)
	ta.SetHeight(commentInputHeight)
	ta.MaxHeight = commentInputHeight
	ta.CharLimit = maxCommentLength

	// Override keymap: shift+enter for newline, enter handled externally
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("shift+enter"))

	return &CommentInputModel{
		textarea: ta,
		mentions: make(map[string]string),
		width:    width,
	}
}

func (m *CommentInputModel) Show(ticketID string) tea.Cmd {
	m.visible = true
	m.ticketID = ticketID
	m.textarea.Reset()
	m.mentions = make(map[string]string)
	m.suggestions = nil
	m.showSuggest = false
	m.atQuery = ""
	return m.textarea.Focus()
}

func (m *CommentInputModel) Hide() {
	m.visible = false
	m.textarea.Blur()
	m.suggestions = nil
	m.showSuggest = false
	m.atQuery = ""
}

func (m *CommentInputModel) Visible() bool {
	return m.visible
}

func (m *CommentInputModel) Height() int {
	if !m.visible {
		return 0
	}
	h := commentInputHeight + 2 // border padding
	if m.showSuggest && len(m.suggestions) > 0 {
		h += min(len(m.suggestions), suggestionMaxVisible)
	}
	return h
}

func (m *CommentInputModel) SetWidth(w int) {
	m.width = w
	m.textarea.SetWidth(w)
}

func (m *CommentInputModel) SetSuggestions(users []model.User) {
	m.suggestions = users
	m.showSuggest = len(users) > 0
	m.selectedIdx = 0
}

func (m *CommentInputModel) Update(msg tea.Msg) (*CommentInputModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
}

func (m *CommentInputModel) handleKey(msg tea.KeyPressMsg) (*CommentInputModel, tea.Cmd) {
	// Handle suggestion navigation
	if m.showSuggest && len(m.suggestions) > 0 {
		switch {
		case key.Matches(msg, keyDown):
			m.selectedIdx = (m.selectedIdx + 1) % len(m.suggestions)
			return m, nil
		case key.Matches(msg, keyUp):
			m.selectedIdx--
			if m.selectedIdx < 0 {
				m.selectedIdx = len(m.suggestions) - 1
			}
			return m, nil
		case key.Matches(msg, keyTab):
			m.acceptSuggestion()
			return m, nil
		case key.Matches(msg, keyEsc):
			m.showSuggest = false
			m.suggestions = nil
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, keyEsc):
		m.Hide()
		return m, func() tea.Msg { return CommentCancelMsg{} }

	case key.Matches(msg, keyEnter):
		return m, m.submit()

	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		// Check for @ trigger after text update
		searchCmd := m.checkAtTrigger()
		if searchCmd != nil {
			return m, tea.Batch(cmd, searchCmd)
		}
		return m, cmd
	}
}

func (m *CommentInputModel) submit() tea.Cmd {
	text := strings.TrimSpace(m.textarea.Value())
	if text == "" {
		return nil
	}
	ticketID := m.ticketID
	mentions := make(map[string]string, len(m.mentions))
	for k, v := range m.mentions {
		mentions[k] = v
	}
	// Don't Hide() here — keep input visible until POST succeeds.
	return func() tea.Msg {
		return CommentSubmitMsg{
			TicketID: ticketID,
			Text:     text,
			Mentions: mentions,
		}
	}
}

func (m *CommentInputModel) acceptSuggestion() {
	if m.selectedIdx >= len(m.suggestions) {
		return
	}

	user := m.suggestions[m.selectedIdx]
	m.mentions[user.DisplayName] = user.AccountID

	// Replace the @query with @DisplayName
	val := m.textarea.Value()
	atIdx := strings.LastIndex(val, "@"+m.atQuery)
	if atIdx != -1 {
		before := val[:atIdx]
		after := val[atIdx+1+len(m.atQuery):]
		m.textarea.SetValue(before + "@" + user.DisplayName + " " + after)
	}

	m.showSuggest = false
	m.suggestions = nil
	m.atQuery = ""
}

func (m *CommentInputModel) checkAtTrigger() tea.Cmd {
	val := m.textarea.Value()
	// Find the last @ that isn't followed by a space
	lastAt := strings.LastIndex(val, "@")
	if lastAt == -1 {
		m.showSuggest = false
		return nil
	}

	afterAt := val[lastAt+1:]
	// If there's a space after @, no active query
	if strings.Contains(afterAt, " ") || strings.Contains(afterAt, "\n") {
		m.showSuggest = false
		return nil
	}

	query := afterAt
	if query == m.atQuery {
		return nil // no change
	}
	m.atQuery = query

	if len(query) < 1 {
		m.showSuggest = false
		return nil
	}

	q := query
	return func() tea.Msg {
		return UserSearchRequestMsg{Query: q}
	}
}

func (m *CommentInputModel) View() string {
	if !m.visible {
		return ""
	}

	var sb strings.Builder

	// Suggestion dropdown (rendered above input)
	if m.showSuggest && len(m.suggestions) > 0 {
		for i, u := range m.suggestions {
			if i >= suggestionMaxVisible {
				break
			}
			line := "  " + u.DisplayName
			if i == m.selectedIdx {
				line = suggestionSelectedStyle.Render("▸ " + u.DisplayName)
			}
			sb.WriteString(line + "\n")
		}
	}

	border := commentInputBorder.
		Width(m.width - 2)

	sb.WriteString(border.Render(m.textarea.View()))

	return sb.String()
}
