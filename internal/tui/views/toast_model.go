package views

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ToastModel struct {
	message        string
	version        int
	terminalWidth  int
	terminalHeight int
}

type ToastTimeoutMsg struct {
	version int
}

func NewToastModel(width, height int) *ToastModel {
	return &ToastModel{
		terminalWidth:  width,
		terminalHeight: height,
	}
}

const toastTimeout = 3 * time.Second

func (m *ToastModel) Show(msg string) tea.Cmd {
	m.message = msg
	m.version++
	v := m.version
	return tea.Tick(toastTimeout, func(time.Time) tea.Msg {
		return ToastTimeoutMsg{version: v}
	})
}

func (m *ToastModel) ShouldHide(msg ToastTimeoutMsg) bool {
	return msg.version == m.version
}

func (m *ToastModel) Hide() {
	m.message = ""
}

func (m *ToastModel) SetSize(width, height int) {
	m.terminalWidth = width
	m.terminalHeight = height
}

func (m *ToastModel) View() *lipgloss.Layer {
	if m.message == "" {
		return nil
	}

	w := 2
	toastStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Bold(true).
		Foreground(lipgloss.Color("#10B981")).
		Padding(0, w)

	content := toastStyle.Render(m.message)

	return lipgloss.NewLayer(content).
		X(m.terminalWidth - lipgloss.Width(content) - w).
		Y(2).
		Z(ZToast)
}
