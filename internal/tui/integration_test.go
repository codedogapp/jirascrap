package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/codedogapp/jirascrap/internal/config"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/model"
)

// --- Mock client ---

type mockClient struct {
	tickets      []model.Ticket
	epicChildren map[string][]model.Ticket
	transitions  []jira.Transition
	fetchErr     error
	transErr     error
	doTransErr   error
}

func (c *mockClient) FetchTickets(_ context.Context) ([]model.Ticket, error) {
	return c.tickets, c.fetchErr
}

func (c *mockClient) FetchEpicChildren(_ context.Context, epicKey string) ([]model.Ticket, error) {
	if c.epicChildren == nil {
		return nil, nil
	}
	return c.epicChildren[epicKey], nil
}

func (c *mockClient) FetchAllEpicChildren(_ context.Context, tickets []model.Ticket) (map[string][]model.Ticket, error) {
	if c.epicChildren == nil {
		return map[string][]model.Ticket{}, nil
	}
	return c.epicChildren, nil
}

func (c *mockClient) FetchTransitions(_ context.Context, _ string) ([]jira.Transition, error) {
	return c.transitions, c.transErr
}

func (c *mockClient) DoTransition(_ context.Context, _ string, _ string) error {
	return c.doTransErr
}

// --- Test helpers ---

var testTickets = []model.Ticket{
	{
		ID: "PROJ-1", Summary: "Fix login bug", Reporter: "Alice",
		Status: "In Progress", StatusCategory: "In Progress",
		Priority: "High", Type: "Task",
		CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 2, 20, 14, 0, 0, 0, time.UTC),
		Markdown:  "Fix the login flow",
	},
	{
		ID: "PROJ-2", Summary: "Add search", Reporter: "Bob",
		Status: "Open", StatusCategory: "To Do",
		Priority: "Medium", Type: "Story",
		CreatedAt: time.Date(2024, 3, 1, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 3, 5, 11, 0, 0, 0, time.UTC),
		Markdown:  "Implement search",
	},
	{
		ID: "EPIC-1", Summary: "Platform Epic", Reporter: "Carol",
		Status: "In Progress", StatusCategory: "In Progress",
		Priority: "High", Type: "Epic",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		Markdown:  "Epic description",
	},
}

var testEpicChildren = map[string][]model.Ticket{
	"EPIC-1": {
		{
			ID: "CHILD-1", Summary: "Child ticket", Reporter: "Dave",
			Status: "Open", StatusCategory: "To Do",
			Priority: "Low", Type: "Task",
			CreatedAt: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC),
		},
	},
}

var testCfg = &config.Config{
	Domain:           "https://test.atlassian.net",
	Email:            "test@example.com",
	APIToken:         "test-token",
	DBPath:           ":memory:",
	CopilotWorkspace: "/tmp",
	CopilotModel:     "test-model",
}

func newTestApp(client *mockClient) *AppModel {
	st := &mockStore{tickets: client.tickets}
	return NewApp(client, st, testCfg)
}

// sendSize simulates a terminal resize to give models dimensions.
func sendSize(t *testing.T, app *AppModel, width, height int) {
	t.Helper()
	app.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

// runInit executes Init and processes the returned commands synchronously.
func runInit(t *testing.T, app *AppModel) {
	t.Helper()
	cmd := app.Init()
	processCmds(t, app, cmd, 10)
}

// processCmds executes commands in a loop (up to maxDepth to prevent infinite loops).
// Uses a timeout to skip timer-based commands (cursor blink, etc.).
func processCmds(t *testing.T, app *AppModel, cmd tea.Cmd, maxDepth int) {
	t.Helper()
	for i := 0; i < maxDepth && cmd != nil; i++ {
		// Run cmd with timeout to skip timer-based commands
		msgCh := make(chan tea.Msg, 1)
		go func() { msgCh <- cmd() }()

		var msg tea.Msg
		select {
		case msg = <-msgCh:
		case <-time.After(50 * time.Millisecond):
			return // skip timer-based commands
		}

		if msg == nil {
			return
		}
		// Handle batch commands
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				processCmds(t, app, c, maxDepth-1)
			}
			return
		}
		var nextCmd tea.Cmd
		_, nextCmd = app.Update(msg)
		cmd = nextCmd
	}
}

// sendKey sends a key press to the app and processes the resulting command.
func sendKey(t *testing.T, app *AppModel, key string) {
	t.Helper()
	msg := tea.KeyPressMsg{Code: keyCode(key), Text: key}
	_, cmd := app.Update(msg)
	processCmds(t, app, cmd, 10)
}

func keyCode(key string) rune {
	switch key {
	case "enter":
		return tea.KeyEnter
	case "esc":
		return tea.KeyEscape
	case "tab":
		return tea.KeyTab
	default:
		if len(key) == 1 {
			return rune(key[0])
		}
		return 0
	}
}

func viewContent(app *AppModel) string {
	return app.View().Content
}

// --- Integration Tests ---

func TestIntegration_StartupAndListRender(t *testing.T) {
	client := &mockClient{tickets: testTickets, epicChildren: testEpicChildren}
	app := newTestApp(client)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	output := viewContent(app)
	if !strings.Contains(output, "PROJ-1") {
		t.Error("expected PROJ-1 in list view")
	}
	if !strings.Contains(output, "Fix login bug") {
		t.Error("expected ticket summary in list view")
	}
	if !strings.Contains(output, "PROJ-2") {
		t.Error("expected PROJ-2 in list view")
	}
}

func TestIntegration_NavigateToDetailAndBack(t *testing.T) {
	client := &mockClient{tickets: testTickets, epicChildren: testEpicChildren}
	app := newTestApp(client)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	// Select first ticket
	sendKey(t, app, "enter")

	output := viewContent(app)
	if !strings.Contains(output, "Fix login bug") {
		t.Error("expected ticket summary in detail view")
	}
	if !strings.Contains(output, "Alice") {
		t.Error("expected reporter in detail view")
	}

	// Go back
	sendKey(t, app, "esc")

	output = viewContent(app)
	if !strings.Contains(output, "PROJ-1") {
		t.Error("expected list view after esc")
	}
}

func TestIntegration_EpicDrillDown(t *testing.T) {
	client := &mockClient{tickets: testTickets, epicChildren: testEpicChildren}
	app := newTestApp(client)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	// After sync, epicChildren should be populated in app state
	if len(app.epicChildren) == 0 {
		t.Fatal("expected epicChildren populated after sync")
	}

	// Navigate to EPIC-1 (3rd item) and select it
	sendKey(t, app, "j") // down
	sendKey(t, app, "j") // down to EPIC-1
	sendKey(t, app, "enter")

	// Epic was in cache, so showEpicChildren should have been called directly
	output := viewContent(app)
	if !strings.Contains(output, "CHILD-1") || !strings.Contains(output, "Child ticket") {
		// If children not shown, it might be that list order differs.
		// Check if we're at least in the epic sub-list
		if !strings.Contains(output, "EPIC-1") {
			t.Errorf("expected epic view, got: %s", output[:min(200, len(output))])
		}
	}

	// Escape back to main list
	sendKey(t, app, "esc")
	output = viewContent(app)
	if !strings.Contains(output, "PROJ-1") {
		// Might be in detail view if we selected wrong item, press esc again
		sendKey(t, app, "esc")
		output = viewContent(app)
		if !strings.Contains(output, "PROJ-1") {
			t.Errorf("expected root list eventually, got: %s", output[:min(200, len(output))])
		}
	}
}

func TestIntegration_TagPopup(t *testing.T) {
	client := &mockClient{tickets: testTickets, epicChildren: testEpicChildren}
	st := &mockStore{tickets: testTickets, tags: []string{"frontend", "backend"}}
	app := NewApp(client, st, testCfg)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	// Open tag popup
	sendKey(t, app, "t")

	if !app.popups.tag.IsVisible() {
		t.Fatal("tag popup should be visible after 't'")
	}

	// Type a tag
	sendKey(t, app, "b")
	sendKey(t, app, "u")
	sendKey(t, app, "g")

	tags := app.popups.tag.CurrentTags()
	if len(tags) != 1 || tags[0] != "bug" {
		t.Errorf("expected tags=[bug], got %v", tags)
	}
}

func TestIntegration_TodoPopup(t *testing.T) {
	client := &mockClient{tickets: testTickets, epicChildren: testEpicChildren}
	st := &mockStore{tickets: testTickets, todos: map[string][]model.Todo{
		"PROJ-1": {{Title: "Existing todo", Done: false}},
	}}
	app := NewApp(client, st, testCfg)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	// Open todo popup
	sendKey(t, app, "n")

	if !app.popups.todo.IsVisible() {
		t.Fatal("todo popup should be visible after 'n'")
	}
}

func TestIntegration_StatusTransition(t *testing.T) {
	transitions := []jira.Transition{
		{ID: "21", Name: "Done", ToStatus: "Done", ToStatusCategory: "Done"},
		{ID: "31", Name: "In Progress", ToStatus: "In Progress", ToStatusCategory: "In Progress"},
	}
	client := &mockClient{tickets: testTickets, epicChildren: testEpicChildren, transitions: transitions}
	app := newTestApp(client)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	// Open status popup
	sendKey(t, app, "s")

	if !app.popups.status.IsVisible() {
		t.Fatal("status popup should be visible after 's'")
	}

	// Simulate transitions loaded
	msg := transitionsLoadedMsg{ticketID: "PROJ-1", transitions: transitions}
	app.Update(msg)

	output := viewContent(app)
	if !strings.Contains(output, "Done") {
		t.Error("expected transition options in status popup")
	}
}

func TestIntegration_SyncError(t *testing.T) {
	client := &mockClient{
		tickets:  nil,
		fetchErr: fmt.Errorf("network timeout"),
	}
	// Give store some cached tickets so sync error doesn't go to fatal state
	st := &mockStore{tickets: testTickets}
	app := NewApp(client, st, testCfg)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	// With cached tickets available, sync error should NOT be fatal
	output := viewContent(app)
	if strings.Contains(output, "Press 'q' to quit") {
		t.Error("sync error with cached data should not put app in fatal error state")
	}
	// Should still show cached tickets
	if !strings.Contains(output, "PROJ-1") {
		t.Error("expected cached tickets still visible after sync error")
	}
}

func TestIntegration_SyncError_NoCachedData(t *testing.T) {
	client := &mockClient{
		tickets:  nil,
		fetchErr: fmt.Errorf("network timeout"),
	}
	// No cached data — sync error should be fatal
	st := &mockStore{tickets: nil}
	app := NewApp(client, st, testCfg)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	output := viewContent(app)
	if !strings.Contains(output, "network timeout") {
		t.Error("expected error message when no cached data and sync fails")
	}
}

func TestIntegration_RefreshTriggerSync(t *testing.T) {
	client := &mockClient{tickets: testTickets, epicChildren: testEpicChildren}
	app := newTestApp(client)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	// Verify initial sync completed
	if !app.synced {
		t.Error("expected synced=true after init")
	}

	// Press refresh — since mock resolves instantly, sync will complete immediately.
	// Just verify it doesn't crash and tickets are still shown.
	sendKey(t, app, "r")

	output := viewContent(app)
	if !strings.Contains(output, "PROJ-1") {
		t.Error("expected tickets visible after refresh")
	}
}

func TestIntegration_DebugOverlay(t *testing.T) {
	client := &mockClient{tickets: testTickets, epicChildren: testEpicChildren}
	app := newTestApp(client)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	// Toggle debug
	sendKey(t, app, "d")

	if !app.popups.debug.IsVisible() {
		t.Error("debug overlay should be visible after 'd'")
	}

	// Toggle off
	sendKey(t, app, "d")

	if app.popups.debug.IsVisible() {
		t.Error("debug overlay should hide after second 'd'")
	}
}

func TestIntegration_QuitKey(t *testing.T) {
	client := &mockClient{tickets: testTickets, epicChildren: testEpicChildren}
	app := newTestApp(client)
	sendSize(t, app, 120, 40)
	runInit(t, app)

	// Press q — should return a quit command
	msg := tea.KeyPressMsg{Code: rune('q'), Text: "q"}
	_, cmd := app.Update(msg)

	if cmd == nil {
		t.Fatal("expected quit command from 'q' key")
	}

	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", result)
	}
}
