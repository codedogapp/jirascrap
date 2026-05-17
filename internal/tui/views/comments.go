package views

import (
	"fmt"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/codedogapp/jirascrap/internal/model"
)

func (m *DetailModel) renderComments(width int) string {
	if m.commentsLoading {
		return commentsLoadingView()
	}

	if m.commentsError != nil {
		return commentsErrorView(m.commentsError)
	}

	if len(m.comments) == 0 {
		return commentsEmptyView(width)
	}

	return commentsView(m.comments, m.commentsTotal, width)
}

func commentsLoadingView() string {
	return dimStyle.Render("Loading comments...")
}

func commentsErrorView(err error) string {
	return dimStyle.Render(fmt.Sprintf("⚠ Failed to load comments: %v", err))
}

func commentsEmptyView(width int) string {
	var sb strings.Builder
	sb.WriteString(dimStyle.Render(strings.Repeat("─", width-separatorPadding)))
	sb.WriteString("\n\n")
	sb.WriteString(dimStyle.Render("💬 No comments"))
	sb.WriteString("\n")
	return sb.String()
}

func commentsView(comments []model.Comment, total int, width int) string {
	var sb strings.Builder

	sb.WriteString(commentsSeparator(width))
	sb.WriteString(commentsHeader(len(comments), total))

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dracula"),
		glamour.WithWordWrap(width),
	)

	for i, c := range comments {
		sb.WriteString(renderComment(c, renderer))
		if i < len(comments)-1 {
			sb.WriteString("\n" + dimStyle.Render("──────────────") + "\n\n")
		}
	}

	return paddingStyle.Render(sb.String())
}

func commentsSeparator(width int) string {
	return dimStyle.Render(strings.Repeat("─", width-separatorPadding)) + "\n\n"
}

func commentsHeader(shown, total int) string {
	if total > shown {
		return fmt.Sprintf("💬 Comments (%d of %d)\n\n", shown, total)
	}
	return fmt.Sprintf("💬 Comments (%d)\n\n", total)
}

func renderComment(c model.Comment, renderer *glamour.TermRenderer) string {
	var sb strings.Builder

	author := lipgloss.NewStyle().Bold(true).Render(c.Author)
	ts := dimStyle.Render(c.CreatedAt.Format("2006-01-02 15:04"))
	sb.WriteString(author + " · " + ts + "\n\n")

	if c.Markdown == "" {
		return sb.String()
	}

	if renderer != nil {
		if rendered, err := renderer.Render(c.Markdown); err == nil {
			sb.WriteString(rendered)
			return sb.String()
		}
	}
	sb.WriteString(c.Markdown + "\n")

	return sb.String()
}
