package views

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	// Colours
	colPrimary   = lipgloss.Color("#7C3AED") // violet
	colSecondary = lipgloss.Color("#A78BFA")
	colMuted     = lipgloss.Color("#6B7280")
	colBg        = lipgloss.Color("#1E1E2E")
	colHighlight = lipgloss.Color("#312E81")
	colSuccess   = lipgloss.Color("#10B981")
	colWarning   = lipgloss.Color("#F59E0B")
	colError     = lipgloss.Color("#EF4444")
	colBorder    = lipgloss.Color("#374151")
	colText      = lipgloss.Color("#E5E7EB")

	// Priority colours
	priorityColors = map[string]color.Color{
		"Highest": lipgloss.Color("#EF4444"),
		"High":    lipgloss.Color("#F97316"),
		"Medium":  lipgloss.Color("#F59E0B"),
		"Low":     lipgloss.Color("#10B981"),
		"Lowest":  lipgloss.Color("#6B7280"),
	}

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(0, 1)

	panelFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colPrimary).
				Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colSecondary).
			Padding(0, 2)

	selectedItemStyle = lipgloss.NewStyle().
				Background(colHighlight)

	dimStyle = lipgloss.NewStyle().Foreground(colMuted).Padding(0, 2)

	keyStyle = lipgloss.NewStyle().
			Background(colPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#111827")).
			Foreground(colMuted).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(colError).
			Bold(true)

	tagStyle = lipgloss.NewStyle().
			Background(colPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2).
			MarginRight(1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colPrimary).
			Padding(1, 2).
			Width(60)
)

func statusColor(status string) color.Color {
	s := strings.ToLower(status)
	switch {
	case contains(s, "progress", "review", "reviewing", "development", "testing", "active", "open"):
		return lipgloss.Color("#3B82F6") // blue
	case contains(s, "done", "closed", "resolved", "complete", "finished", "released", "deployed"):
		return lipgloss.Color("#10B981") // green
	case contains(s, "block", "impediment", "on hold", "waiting", "rejected", "cancelled"):
		return lipgloss.Color("#EF4444") // red
	case contains(s, "todo", "to do", "backlog", "new", "open", "planned"):
		return lipgloss.Color("#6B7280") // grey
	default:
		return lipgloss.Color("#6B7280") // grey fallback
	}
}

// contains reports whether s contains any of the given substrings.
func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func priorityColor(priority string) color.Color {
	if c, ok := priorityColors[priority]; ok {
		return c
	}
	return colMuted
}
