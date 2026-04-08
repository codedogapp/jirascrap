package views

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

type Styles struct {
	App lipgloss.Style
}

func NewStyles() Styles {
	return Styles{
		App: lipgloss.NewStyle().
			Padding(1, 2),
	}
}

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

var (
	// Colours
	colPrimary   = lipgloss.Color("#7C3AED") // violet
	colSecondary = lipgloss.Color("#A78BFA")
	colMuted     = lipgloss.Color("#6B7280")
	blue         = lipgloss.Color("#3B82F6")
	green        = lipgloss.Color("#10B981")
	red          = lipgloss.Color("#EF4444")
	grey         = lipgloss.Color("#6B7280")
	greyFallback = lipgloss.Color("#6B7280")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colSecondary).
			Padding(0, 2)

	dimStyle = lipgloss.NewStyle().
			Foreground(colMuted).
			Padding(0, 2)

	tagStyle = lipgloss.NewStyle().
			Background(colPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2).
			MarginRight(1)
)

func statusColor(status string) color.Color {
	s := strings.ToLower(status)
	switch {
	case contains(s, "progress", "review", "reviewing", "development", "testing", "active", "open"):
		return blue
	case contains(s, "done", "closed", "resolved", "complete", "finished", "released", "deployed"):
		return green
	case contains(s, "block", "impediment", "on hold", "waiting", "rejected", "cancelled"):
		return red
	case contains(s, "todo", "to do", "backlog", "new", "open", "planned"):
		return grey
	default:
		return greyFallback
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
	priorityColors := map[string]color.Color{
		"Highest": lipgloss.Color("#EF4444"),
		"High":    lipgloss.Color("#F97316"),
		"Medium":  lipgloss.Color("#F59E0B"),
		"Low":     lipgloss.Color("#10B981"),
		"Lowest":  lipgloss.Color("#6B7280"),
	}

	if c, ok := priorityColors[priority]; ok {
		return c
	}

	return colMuted
}
