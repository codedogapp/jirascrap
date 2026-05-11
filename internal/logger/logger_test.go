package logger

import (
	"fmt"
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
	for i := 0; i < 150; i++ {
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

	for i := 0; i < 100; i++ {
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
