package logger

import (
	"fmt"
	"sync"
	"testing"
)

func newTestLogger() *Logger {
	return &Logger{minLevel: DEBUG}
}

// mockPersister records InsertLog calls for testing.
type mockPersister struct {
	mu      sync.Mutex
	entries []struct{ level, message string }
}

func (m *mockPersister) InsertLog(level, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, struct{ level, message string }{level, message})
	return nil
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

func TestLogger_PersistsToDB(t *testing.T) {
	l := newTestLogger()
	p := &mockPersister{}
	l.SetPersister(p)

	l.Warn("disk full")
	l.Error("crash")

	if len(p.entries) != 2 {
		t.Fatalf("expected 2 persisted entries, got %d", len(p.entries))
	}

	if p.entries[0].level != "WARN" || p.entries[0].message != "disk full" {
		t.Errorf("entry[0]: got %+v", p.entries[0])
	}
	if p.entries[1].level != "ERROR" || p.entries[1].message != "crash" {
		t.Errorf("entry[1]: got %+v", p.entries[1])
	}
}

func TestLogger_NoPersisterNoPanic(t *testing.T) {
	l := newTestLogger()
	// Should not panic without a persister
	l.Info("no persister")
	l.Error("still fine")

	if len(l.Logs()) != 2 {
		t.Errorf("expected 2 logs in memory")
	}
}
