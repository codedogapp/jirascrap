package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/codedogapp/jirascrap/internal/logger"
)

type DebugModel struct {
	visible        bool
	viewport       viewport.Model
	terminalWidth  int
	terminalHeight int
}

const (
	RatioWidth       = 2
	RatioHeight      = 3
	PopupWidthScale  = 0.7 // fraction of content width for popup overlays
	todoHeightRatio  = 2
)

func NewDebugModel(width int, height int) *DebugModel {
	modelWidth := width / RatioWidth
	modelHeight := height / RatioHeight

	vp := viewport.New(
		viewport.WithWidth(modelWidth),
		viewport.WithHeight(modelHeight),
	)
	vp.SetContent("")

	return &DebugModel{
		visible:        false,
		viewport:       vp,
		terminalWidth:  width,
		terminalHeight: height,
	}
}

func (m *DebugModel) Show() {
	m.visible = true
	m.viewport.SetContent(formatLogs())
}

func (m *DebugModel) Hide() {
	m.visible = false
}

func (m *DebugModel) View() *lipgloss.Layer {
	if !m.visible {
		return nil
	}

	debugView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		Render(m.viewport.View())

	overlayWidth := m.viewport.Width()
	overlayHeight := m.viewport.Height()
	title := "─ DEBUG "
	top := title + strings.Repeat("─", overlayWidth-len(title)+2)

	content := "╭" + top + "╮\n" + debugView

	overlay := lipgloss.NewLayer(content).
		X((m.terminalWidth - overlayWidth) / 2).
		Y((m.terminalHeight - overlayHeight) / 2).
		Z(ZPopup)

	return overlay
}

func (m *DebugModel) Update(msg tea.KeyPressMsg) tea.Cmd {
	m.viewport.SetContent(formatLogs())
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

func (m *DebugModel) SetSize(width int, height int) {
	m.viewport.SetWidth(width / RatioWidth)
	m.viewport.SetHeight(height / RatioHeight)
	m.terminalWidth = width
	m.terminalHeight = height
}

func (m *DebugModel) IsVisible() bool {
	return m.visible
}

func formatLogs() string {
	var sb strings.Builder
	entries := logger.Log.Logs()
	for i, entry := range entries {
		log := fmt.Sprintf(
			"[%s] %s",
			entry.Level.String(),
			entry.Message,
		)

		styledLog := lipgloss.NewStyle().
			Foreground(getColor(entry.Level)).
			Render(log)
		sb.WriteString(styledLog)

		if i != len(entries)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func getColor(level logger.Level) ansi.BasicColor {
	switch level {
	case logger.DEBUG:
		return lipgloss.White
	case logger.INFO:
		return lipgloss.Blue
	case logger.WARN:
		return lipgloss.Yellow
	case logger.ERROR:
		return lipgloss.Red
	default:
		return lipgloss.White
	}
}
