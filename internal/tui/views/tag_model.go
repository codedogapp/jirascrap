package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/model"
)

type TagModel struct {
	tagInput      textinput.Model
	ticket        model.Ticket
	visible       bool
	contentWidth  int
	contentHeight int
	allTags       []string
	suggestions   []string
	selectedIdx   int
}

type TagsFilledMsg struct {
	ID   string
	Tags []string
}

func NewTagModel(contentWidth, contentHeight int, allTags []string) *TagModel {
	ti := textinput.New()
	ti.Placeholder = "tag1, tag2, ..."
	ti.SetWidth(contentWidth / RatioWidth)
	return &TagModel{
		tagInput:      ti,
		contentWidth:  contentWidth,
		contentHeight: contentHeight,
		allTags:       allTags,
		selectedIdx:   -1,
	}
}

func (m *TagModel) Show(ticket model.Ticket) tea.Cmd {
	m.ticket = ticket
	m.visible = true
	m.suggestions = nil
	m.selectedIdx = -1
	m.tagInput.SetValue(strings.Join(ticket.Tags, ", "))
	return m.tagInput.Focus()
}

func (m *TagModel) Hide() {
	m.visible = false
}

func (m *TagModel) IsVisible() bool {
	return m.visible
}

func (m *TagModel) CurrentTags() []string {
	return trimTags(m.tagInput.Value())
}

func (m *TagModel) TicketID() string {
	return m.ticket.ID
}

func (m *TagModel) Update(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "tab":
		if len(m.suggestions) > 0 {
			if m.selectedIdx < 0 {
				m.selectedIdx = 0
			}
			m.completeSuggestion()
		}
		return nil
	case "down":
		if len(m.suggestions) > 0 && m.selectedIdx < len(m.suggestions)-1 {
			m.selectedIdx++
		}
		return nil
	case "up":
		if len(m.suggestions) > 0 && m.selectedIdx > 0 {
			m.selectedIdx--
		}
		return nil
	}

	var cmd tea.Cmd
	m.tagInput, cmd = m.tagInput.Update(msg)
	m.updateSuggestions()
	return cmd
}

func (m *TagModel) currentWord() string {
	val := m.tagInput.Value()
	lastComma := strings.LastIndex(val, ",")
	if lastComma >= 0 {
		return strings.TrimSpace(val[lastComma+1:])
	}
	return strings.TrimSpace(val)
}

func (m *TagModel) updateSuggestions() {
	word := m.currentWord()
	if word == "" {
		m.suggestions = nil
		m.selectedIdx = -1
		return
	}

	// collect already-used tags to exclude from suggestions
	val := m.tagInput.Value()
	parts := strings.Split(val, ",")
	used := make(map[string]bool, len(parts))
	for _, p := range parts[:len(parts)-1] {
		used[strings.TrimSpace(p)] = true
	}

	lower := strings.ToLower(word)
	m.suggestions = nil
	for _, t := range m.allTags {
		if used[t] {
			continue
		}
		if strings.HasPrefix(strings.ToLower(t), lower) && strings.ToLower(t) != lower {
			m.suggestions = append(m.suggestions, t)
			if len(m.suggestions) == 5 {
				break
			}
		}
	}

	if m.selectedIdx >= len(m.suggestions) {
		m.selectedIdx = -1
	}
}

func (m *TagModel) completeSuggestion() {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.suggestions) {
		return
	}
	suggestion := m.suggestions[m.selectedIdx]
	val := m.tagInput.Value()
	lastComma := strings.LastIndex(val, ",")
	var newVal string
	if lastComma >= 0 {
		newVal = val[:lastComma+1] + " " + suggestion
	} else {
		newVal = suggestion
	}
	m.tagInput.SetValue(newVal + ", ")
	m.tagInput.CursorEnd()
	m.suggestions = nil
	m.selectedIdx = -1
}

func (m *TagModel) UpdateMsg(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.tagInput, cmd = m.tagInput.Update(msg)
	return cmd
}

func (m *TagModel) SetSize(contentWidth, contentHeight int) {
	m.contentWidth = contentWidth
	m.contentHeight = contentHeight
	m.tagInput.SetWidth(contentWidth / RatioWidth)
}

func (m *TagModel) View() *lipgloss.Layer {
	if !m.visible {
		return nil
	}

	popupContent := lipgloss.NewStyle().
		Bold(true).
		Render(
			fmt.Sprintf(
				"Tagging %s:\n\n%s",
				m.ticket.ID,
				m.tagInput.View(),
			),
		)

	if len(m.suggestions) > 0 {
		popupContent += "\n\n" + m.renderSuggestions()
	}

	popupContentStyled := lipgloss.NewStyle().Padding(1, 1).Render(popupContent)

	popupView := tagViewPopUp.Render(popupContentStyled)

	overlayWidth := m.contentWidth / RatioWidth
	dashes := strings.Repeat("─", overlayWidth+1)
	topLine := topBorder.Render("╭─ ") +
		popUpTitle.Render("TAG") +
		topBorder.Render(" "+dashes+"╮")

	content := topLine + "\n" + popupView

	return lipgloss.NewLayer(content).
		X((m.contentWidth - overlayWidth) / 2).
		Y((m.contentHeight - RatioHeight) / 2).
		Z(1)
}

func (m *TagModel) renderSuggestions() string {
	var sb strings.Builder
	for i, s := range m.suggestions {
		if i == m.selectedIdx {
			sb.WriteString(lipgloss.NewStyle().Foreground(colPrimary).Bold(true).Render("▶ " + s))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(grey).Render("  " + s))
		}
		if i < len(m.suggestions)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func trimTags(tags string) []string {
	splitTags := strings.Split(tags, ",")
	var trimmedTags []string
	for _, tag := range splitTags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			trimmedTags = append(trimmedTags, trimmed)
		}
	}
	return trimmedTags
}
