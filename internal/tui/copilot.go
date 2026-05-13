package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/logger"
	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/tmux"
	"github.com/codedogapp/jirascrap/internal/tui/keymaps"
)

// copilotSession is always valid — "copilot" passes sanitizeName.
var copilotSession, _ = tmux.NewSession("copilot")

func (m *AppModel) handleSendToCopilot(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !key.Matches(msg, keymaps.DefaultKeyMap.SendToCopilot) {
		return false, nil
	}

	ticket, ok := m.withSelectedTicket()
	if !ok {
		return false, nil
	}

	todos, err := m.todoStore.GetTodos(ticket.ID)
	if err != nil {
		logger.Log.Warn(fmt.Sprintf("failed to load todos for copilot: %v", err))
	}

	return true, m.sendToCopilotCmd(ticket, todos)
}

func (m *AppModel) sendToCopilotCmd(ticket model.Ticket, todos []model.Todo) tea.Cmd {
	return func() tea.Msg {
		workspace := m.config.CopilotWorkspace
		copilotModel := m.config.CopilotModel

		// Write prompt file
		promptPath, err := writePromptFile(ticket, todos)
		if err != nil {
			return copilotLaunchedMsg{ticketID: ticket.ID, err: err}
		}

		// Ensure tmux session
		if err := os.MkdirAll(workspace, 0750); err != nil {
			return copilotLaunchedMsg{ticketID: ticket.ID, err: fmt.Errorf("creating workspace: %w", err)}
		}
		if err := copilotSession.Ensure(workspace); err != nil {
			return copilotLaunchedMsg{ticketID: ticket.ID, err: fmt.Errorf("tmux session: %w", err)}
		}

		// Find or create window for this ticket
		windowID, err := resolveWindow(ticket.ID, workspace)
		if err != nil {
			return copilotLaunchedMsg{ticketID: ticket.ID, err: fmt.Errorf("tmux window: %w", err)}
		}

		// Skip if copilot already running
		cmd, err := copilotSession.WindowCommand(windowID)
		if err != nil {
			logger.Log.Warn(fmt.Sprintf("failed to check window command: %v", err))
		} else if strings.Contains(cmd, "copilot") {
			return copilotLaunchedMsg{ticketID: ticket.ID}
		}

		// Launch copilot
		cdCmd := fmt.Sprintf("cd %s", shellQuote(workspace))
		if err := copilotSession.SendKeys(windowID, cdCmd); err != nil {
			return copilotLaunchedMsg{ticketID: ticket.ID, err: fmt.Errorf("cd to workspace: %w", err)}
		}

		copilotCmd := fmt.Sprintf(
			"copilot --plan --model %s --allow-all-paths -i \"$(cat %s)\"",
			shellQuote(copilotModel), shellQuote(promptPath),
		)
		if err := copilotSession.SendKeys(windowID, copilotCmd); err != nil {
			return copilotLaunchedMsg{ticketID: ticket.ID, err: fmt.Errorf("launching copilot: %w", err)}
		}

		return copilotLaunchedMsg{ticketID: ticket.ID}
	}
}

func resolveWindow(ticketID, workspace string) (string, error) {
	if id, ok, err := copilotSession.FindWindow(ticketID); err != nil {
		return "", fmt.Errorf("find window %q: %w", ticketID, err)
	} else if ok {
		return id, nil
	}

	// Claim first unused window if session was just created
	if id, name, ok, err := copilotSession.FirstWindow(); err != nil {
		return "", fmt.Errorf("first window: %w", err)
	} else if ok && !strings.Contains(name, "-") {
		_ = copilotSession.RenameWindow(id, ticketID)
		return id, nil
	}

	return copilotSession.NewWindow(ticketID, workspace)
}

func writePromptFile(ticket model.Ticket, todos []model.Todo) (string, error) {
	dir := filepath.Join(os.TempDir(), "jirascrap")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", fmt.Errorf("creating prompt dir: %w", err)
	}

	sanitizedID := strings.ReplaceAll(strings.ToLower(ticket.ID), "/", "-")
	path := filepath.Join(dir, fmt.Sprintf("%s.md", sanitizedID))

	if err := os.WriteFile(path, []byte(buildCopilotPrompt(ticket, todos)), 0600); err != nil {
		return "", fmt.Errorf("writing prompt: %w", err)
	}

	return path, nil
}

func buildCopilotPrompt(ticket model.Ticket, todos []model.Todo) string {
	var b strings.Builder

	b.WriteString("use caveman\n\n")
	b.WriteString("I'm working on this Jira ticket:\n\n")
	b.WriteString(fmt.Sprintf("# %s: %s\n\n", ticket.ID, ticket.Summary))
	b.WriteString(fmt.Sprintf("- **Status:** %s\n", ticket.Status))
	b.WriteString(fmt.Sprintf("- **Priority:** %s\n", ticket.Priority))
	b.WriteString(fmt.Sprintf("- **Type:** %s\n", ticket.Type))
	b.WriteString(fmt.Sprintf("- **Reporter:** %s\n", ticket.Reporter))

	if ticket.EpicID != nil {
		b.WriteString(fmt.Sprintf("- **Epic:** %s\n", *ticket.EpicID))
	}

	if len(ticket.Tags) > 0 {
		b.WriteString(fmt.Sprintf("- **Tags:** %s\n", strings.Join(ticket.Tags, ", ")))
	}

	if len(todos) > 0 {
		b.WriteString("\n## Todos\n")
		for _, todo := range todos {
			check := " "
			if todo.Done {
				check = "x"
			}
			b.WriteString(fmt.Sprintf("- [%s] %s\n", check, todo.Title))
		}
	}

	if ticket.Markdown != "" {
		b.WriteString("\n## Description\n")
		b.WriteString(ticket.Markdown)
		b.WriteString("\n")
	}

	b.WriteString("\n---\nPlan the implementation for this ticket. Break it into clear steps.\n")

	return b.String()
}

func (m *AppModel) handleCopilotLaunched(msg copilotLaunchedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m, m.popups.toast.Show(fmt.Sprintf("✗ %s", msg.err))
	}

	return m, m.popups.toast.Show(
		fmt.Sprintf("✓ Copilot launched for %s — tmux attach -t %s", msg.ticketID, copilotSession.Name),
	)
}

// shellQuote wraps a string in single quotes, escaping embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
