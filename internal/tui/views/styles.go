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

var priorityColors = map[string]color.Color{
	"Highest": lipgloss.Color("#EF4444"),
	"High":    lipgloss.Color("#F97316"),
	"Medium":  lipgloss.Color("#F59E0B"),
	"Low":     lipgloss.Color("#10B981"),
	"Lowest":  lipgloss.Color("#6B7280"),
}

var (
	// Colours
	colPrimary   = lipgloss.Color("#7C3AED") // violet
	colSecondary = lipgloss.Color("#A78BFA")
	grey         = lipgloss.Color("#6B7280")
	blue         = lipgloss.Color("#3B82F6")
	green        = lipgloss.Color("#10B981")
	red          = lipgloss.Color("#EF4444")

	// Z-index layers for popup overlays.
	ZPopup  = 1  // standard popups (tag, todo)
	ZStatus = 2  // status popup (above other popups)
	ZToast  = 10 // toast notifications (topmost)

	// Popup sizing ratios
	RatioWidth      = 2
	RatioHeight     = 3
	PopupWidthScale = 0.7 // fraction of content width for popup overlays
	todoHeightRatio = 2

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colSecondary).
			Padding(0, 2)

	dimStyle = lipgloss.NewStyle().
			Foreground(grey).
			Padding(0, 0, 0, 2)

	tagStyle = lipgloss.NewStyle().
			Background(colPrimary).
			Foreground(lipgloss.White).
			Padding(0, 2).
			MarginRight(1)

	popUpTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colSecondary)

	tagViewPopUp = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderTop(false).
			BorderForeground(colSecondary).
			Padding(0, 1)

	paddingStyle = lipgloss.NewStyle().
			Padding(0, 2)

	topBorder = lipgloss.NewStyle().
			Foreground(colSecondary)

	tagListStyle = lipgloss.NewStyle().Foreground(colSecondary)

	epicBoltStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F5A623"))

	boldStyle = lipgloss.NewStyle().Bold(true)

	suggestionSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212"))

	commentInputBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))

	toastStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Bold(true).
			Foreground(green).
			Padding(0, 2)
)

func statusColor(status string) color.Color {
	s := strings.ToLower(status)
	switch {
	case contains(s, "progress", "review", "reviewing", "development", "testing", "active"):
		return blue

	case contains(s, "done", "closed", "resolved", "complete", "finished", "released", "deployed"):
		return green

	case contains(s, "block", "impediment", "on hold", "waiting", "rejected", "cancelled"):
		return red

	case contains(s, "todo", "to do", "backlog", "new", "open", "planned"):
		return grey

	default:
		return grey
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

	return grey
}

// RenderPopupLayer builds a styled popup overlay with a titled border.
// content is the inner rendered content; title is shown in the top border.
// An optional minWidth forces the popup to be at least that wide.
func RenderPopupLayer(content, title string, x, y, z int, minWidth ...int) *lipgloss.Layer {
	contentStyled := lipgloss.NewStyle().Padding(1, 1).Render(content)
	popup := tagViewPopUp
	if len(minWidth) > 0 && minWidth[0] > 0 {
		popup = popup.Width(minWidth[0])
	}
	popupView := popup.Render(contentStyled)
	popupWidth := lipgloss.Width(popupView)

	titleLabel := "╭─ " + title + " ╮"
	dashCount := popupWidth - lipgloss.Width(titleLabel)
	if dashCount < 0 {
		dashCount = 0
	}
	dashes := strings.Repeat("─", dashCount)
	topLine := topBorder.Render("╭─ ") +
		popUpTitle.Render(title) +
		topBorder.Render(" "+dashes+"╮")

	full := topLine + "\n" + popupView
	return lipgloss.NewLayer(full).X(x).Y(y).Z(z)
}
