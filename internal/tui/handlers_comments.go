package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/logger"
	"github.com/codedogapp/jirascrap/internal/tui/views"
)

const maxComments = 20

func (m *AppModel) updateCommentMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case commentsLoadedMsg:
		return m.handleCommentsLoaded(msg)

	case commentsErrorMsg:
		return m.handleCommentsError(msg)

	case views.CommentSubmitMsg:
		return m.handleCommentSubmit(msg)

	case views.CommentCancelMsg:
		return m, nil

	case commentPostSuccessMsg:
		return m.handleCommentPostSuccess(msg)

	case commentPostErrorMsg:
		return m.handleCommentPostError(msg)

	case views.UserSearchRequestMsg:
		return m.handleUserSearchRequest(msg)

	case userSearchResultMsg:
		return m.handleUserSearchResult(msg)

	default:
		return m, nil
	}
}

func (m *AppModel) fetchCommentsCmd(ticketID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		comments, total, err := m.jiraClient.FetchComments(ctx, ticketID, maxComments)
		if err != nil {
			return commentsErrorMsg{ticketID: ticketID, err: err}
		}
		return commentsLoadedMsg{ticketID: ticketID, comments: comments, total: total}
	}
}

func (m *AppModel) handleCommentsLoaded(msg commentsLoadedMsg) (tea.Model, tea.Cmd) {
	dm, ok := m.activeDetailModel()
	if !ok || dm.Ticket().ID != msg.ticketID {
		return m, nil
	}
	dm.SetComments(msg.comments, msg.total)
	return m, nil
}

func (m *AppModel) handleCommentsError(msg commentsErrorMsg) (tea.Model, tea.Cmd) {
	dm, ok := m.activeDetailModel()
	if !ok || dm.Ticket().ID != msg.ticketID {
		return m, nil
	}
	dm.SetCommentsError(msg.err)
	return m, nil
}

func (m *AppModel) handleCommentSubmit(msg views.CommentSubmitMsg) (tea.Model, tea.Cmd) {
	return m, tea.Batch(
		m.postCommentCmd(msg.TicketID, msg.Text, msg.Mentions),
		m.popups.toast.Show("Posting comment..."),
	)
}

func (m *AppModel) postCommentCmd(ticketID, text string, mentions map[string]string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()

		body := jira.BuildCommentADF(text, mentions)
		if err := m.jiraClient.PostComment(ctx, ticketID, body); err != nil {
			return commentPostErrorMsg{err: err}
		}
		return commentPostSuccessMsg{ticketID: ticketID}
	}
}

func (m *AppModel) handleCommentPostSuccess(msg commentPostSuccessMsg) (tea.Model, tea.Cmd) {
	dm, ok := m.activeDetailModel()
	if ok {
		dm.CommentInput().Hide()
		dm.AdjustViewportHeight()
	}
	return m, tea.Batch(
		m.popups.toast.Show("✓ Comment added"),
		m.fetchCommentsCmd(msg.ticketID),
	)
}

func (m *AppModel) handleCommentPostError(msg commentPostErrorMsg) (tea.Model, tea.Cmd) {
	// Keep comment input visible so user doesn't lose their text
	return m, m.popups.toast.Show("⚠ " + msg.err.Error())
}

func (m *AppModel) handleUserSearchResult(msg userSearchResultMsg) (tea.Model, tea.Cmd) {
	dm, ok := m.activeDetailModel()
	if !ok {
		return m, nil
	}
	dm.CommentInput().SetSuggestions(msg.users)
	return m, nil
}

func (m *AppModel) handleUserSearchRequest(msg views.UserSearchRequestMsg) (tea.Model, tea.Cmd) {
	return m, m.searchUsersCmd(msg.Query)
}

func (m *AppModel) searchUsersCmd(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		defer cancel()
		users, err := m.jiraClient.SearchUsers(ctx, query)
		if err != nil {
			logger.Log.Warn(fmt.Sprintf("user search error: %v", err))
			return userSearchResultMsg{users: nil}
		}
		return userSearchResultMsg{users: users}
	}
}
