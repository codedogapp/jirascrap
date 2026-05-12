package logger

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func newTestLogger() *Logger {
	return &Logger{minLevel: DEBUG}
}

func TestLogger_BasicLevels(t *testing.T) {
	l := newTestLogger()
	l.Info("info msg")
	l.Error("error msg")
	l.Debug("debug msg")
	l.Warn("warn msg")

	logs := l.Logs()
	if len(logs) != 4 {
		t.Fatalf("expected 4 logs, got %d", len(logs))
	}

	expected := []struct {
		level Level
		msg   string
	}{
		{INFO, "info msg"},
		{ERROR, "error msg"},
		{DEBUG, "debug msg"},
		{WARN, "warn msg"},
	}

	for i, want := range expected {
		if logs[i].Level != want.level {
			t.Errorf("log[%d] level: got %v, want %v", i, logs[i].Level, want.level)
		}
		if logs[i].Message != want.msg {
			t.Errorf("log[%d] message: got %q, want %q", i, logs[i].Message, want.msg)
		}
	}
}

func TestLogger_MaxLogs(t *testing.T) {
	l := newTestLogger()
	for i := range 150 {
		l.Info(fmt.Sprintf("msg-%d", i))
	}

	logs := l.Logs()
	if len(logs) != maxLogs {
		t.Fatalf("expected %d logs, got %d", maxLogs, len(logs))
	}

	// Oldest kept should be msg-50
	if logs[0].Message != "msg-50" {
		t.Errorf("oldest log: got %q, want %q", logs[0].Message, "msg-50")
	}
	if logs[maxLogs-1].Message != "msg-149" {
		t.Errorf("newest log: got %q, want %q", logs[maxLogs-1].Message, "msg-149")
	}
}

func TestLogger_EmptyLogs(t *testing.T) {
	l := newTestLogger()
	logs := l.Logs()
	if logs != nil {
		t.Errorf("expected nil logs, got %v", logs)
	}
}

func TestLogger_LogsReturnsCopy(t *testing.T) {
	l := newTestLogger()
	l.Info("original")

	logs := l.Logs()
	logs[0].Message = "mutated"

	fresh := l.Logs()
	if fresh[0].Message != "original" {
		t.Errorf("Logs() returned reference, not copy: got %q", fresh[0].Message)
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{Level(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		got := tt.level.String()
		if got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestLogger_Concurrent(t *testing.T) {
	l := newTestLogger()
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			l.Info(fmt.Sprintf("concurrent-%d", n))
		}(i)
	}

	wg.Wait()

	logs := l.Logs()
	if len(logs) != 100 {
		t.Errorf("expected 100 logs, got %d", len(logs))
	}
}

func TestLogger_WritesToFile(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger()
	l.SetOutput(&buf)

	l.Warn("disk full")
	l.Error("crash")

	output := buf.String()
	if !strings.Contains(output, "[WARN ] disk full") {
		t.Errorf("missing WARN in file output: %s", output)
	}
	if !strings.Contains(output, "[ERROR] crash") {
		t.Errorf("missing ERROR in file output: %s", output)
	}

	// Should have timestamps
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if len(line) < 23 {
			t.Errorf("line too short for timestamp: %q", line)
		}
	}
}

func TestOpenSessionLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")

	f, path, err := OpenSessionLog(logDir)
	if err != nil {
		t.Fatalf("OpenSessionLog: %v", err)
	}
	defer f.Close()

	if !strings.HasPrefix(path, logDir) {
		t.Errorf("path %q not in dir %q", path, logDir)
	}
	if !strings.Contains(path, "session-") {
		t.Errorf("path missing session- prefix: %q", path)
	}

	// Write and verify
	fmt.Fprintln(f, "test line")
	f.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "test line") {
		t.Errorf("log file missing content")
	}
}

func TestPruneOldLogs(t *testing.T) {
	dir := t.TempDir()

	// Create 15 fake log files
	for i := range 15 {
		name := fmt.Sprintf("session-20260101-%06d.log", i)
		os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644)
	}

	// Also create a non-log file that should be ignored
	os.WriteFile(filepath.Join(dir, "other.txt"), []byte("x"), 0o644)

	pruneOldLogs(dir)

	entries, _ := os.ReadDir(dir)
	logCount := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "session-") {
			logCount++
		}
	}

	if logCount != maxLogFiles {
		t.Errorf("expected %d log files after prune, got %d", maxLogFiles, logCount)
	}

	// other.txt should still exist
	if _, err := os.Stat(filepath.Join(dir, "other.txt")); err != nil {
		t.Errorf("non-log file was deleted: %v", err)
	}
}
