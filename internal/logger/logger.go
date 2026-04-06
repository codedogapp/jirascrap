package logger

import (
	"sync"
)

type Logger struct {
	mu   sync.Mutex
	logs []LogEntry
}

type LogEntry struct {
	Level   Level
	Message string
}

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

const maxLogs = 100

var Log = &Logger{}

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (l *Logger) Info(msg string) {
	l.add(INFO, msg)
}

func (l *Logger) Error(msg string) {
	l.add(ERROR, msg)
}

func (l *Logger) Debug(msg string) {
	l.add(DEBUG, msg)
}

func (l *Logger) Warn(msg string) {
	l.add(WARN, msg)
}

func (l *Logger) add(level Level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log := LogEntry{Level: level, Message: msg}
	l.logs = append(l.logs, log)
	if len(l.logs) > maxLogs {
		l.logs = l.logs[len(l.logs)-maxLogs:]
	}
}

func (l *Logger) Logs() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logs
}
