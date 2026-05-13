package tmux

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// validName matches allowed tmux session/window name characters.
var validName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// sanitizeName validates a tmux session or window name.
func sanitizeName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("tmux name cannot be empty")
	}
	if !validName.MatchString(name) {
		return "", fmt.Errorf(
			"tmux name %q contains invalid characters (allowed: alphanumeric, dot, dash, underscore)",
			name,
		)
	}
	return name, nil
}

// Session manages a named tmux session.
type Session struct {
	Name string
}

// NewSession returns a Session handle for the given name.
func NewSession(name string) (*Session, error) {
	safe, err := sanitizeName(name)
	if err != nil {
		return nil, err
	}
	return &Session{Name: safe}, nil
}

// Ensure creates the session if it doesn't exist, starting in the given directory.
func (s *Session) Ensure(dir string) error {
	exists, err := s.Exists()
	if err != nil {
		return fmt.Errorf("check session %q: %w", s.Name, err)
	}
	if exists {
		return nil
	}
	return run("new-session", "-d", "-s", s.Name, "-c", dir)
}

// Exists checks whether the session is alive.
func (s *Session) Exists() (bool, error) {
	err := run("has-session", "-t", s.Name)
	if err == nil {
		return true, nil
	}

	// tmux returns exit code 1 when session doesn't exist — expected
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// FindWindow returns the window ID for a window with the given name.
func (s *Session) FindWindow(name string) (string, bool, error) {
	out, err := output("list-windows", "-t", s.Name, "-F", "#{window_id}\t#{window_name}")
	if err != nil {
		return "", false, fmt.Errorf("list-windows: %w", err)
	}

	for _, line := range lines(out) {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 && parts[1] == name {
			return parts[0], true, nil
		}
	}

	return "", false, nil
}

// FirstWindow returns the ID and name of the first (and only) window, if there's exactly one.
func (s *Session) FirstWindow() (id string, name string, ok bool, err error) {
	out, err := output("list-windows", "-t", s.Name, "-F", "#{window_id}\t#{window_name}")
	if err != nil {
		return "", "", false, fmt.Errorf("list-windows: %w", err)
	}

	l := lines(out)
	if len(l) != 1 {
		return "", "", false, nil
	}

	parts := strings.SplitN(l[0], "\t", 2)
	if len(parts) != 2 {
		return "", "", false, nil
	}

	return parts[0], parts[1], true, nil
}

// RenameWindow renames the window with the given ID.
func (s *Session) RenameWindow(windowID, name string) error {
	return run("rename-window", "-t", windowID, name)
}

// NewWindow creates a new window in the session and returns its ID.
func (s *Session) NewWindow(name, dir string) (string, error) {
	safe, err := sanitizeName(name)
	if err != nil {
		return "", err
	}

	out, err := output(
		"new-window",
		"-t",
		s.Name,
		"-n",
		safe,
		"-c",
		dir,
		"-P",
		"-F",
		"#{window_id}",
	)

	if err != nil {
		return "", fmt.Errorf("new-window: %w", err)
	}

	return strings.TrimSpace(out), nil
}

// SendKeys sends a command string to the target, followed by Enter.
func (s *Session) SendKeys(target, cmd string) error {
	return run("send-keys", "-t", target, cmd, "Enter")
}

// WindowCommand returns the current command running in the given window's active pane.
func (s *Session) WindowCommand(windowID string) (string, error) {
	out, err := output("list-panes", "-t", windowID, "-F", "#{pane_current_command}")
	if err != nil {
		return "", fmt.Errorf("list-panes: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func run(args ...string) error {
	return exec.Command("tmux", args...).Run() // #nosec G204 -- args are hardcoded tmux subcommands
}

func output(args ...string) (string, error) {
	out, err := exec.Command("tmux", args...).Output() // #nosec G204 -- args are hardcoded tmux subcommands
	return string(out), err
}

func lines(s string) []string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}
