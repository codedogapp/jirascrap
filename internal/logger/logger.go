package logger

import (
	"slices"
	"sync"
)

// LogPersister is the storage backend for log entries.
// Implemented by store.SqliteLogStore — defined here as interface to avoid circular deps.
type LogPersister interface {
	InsertLog(level, message string) error
}

type Logger struct {
	mu        sync.Mutex
	logs      []LogEntry
	persister LogPersister
	minLevel  Level
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

// Log is the default global logger instance.
var Log = &Logger{minLevel: DEBUG}

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

// SetPersister sets the database backend for log persistence.
func (l *Logger) SetPersister(p LogPersister) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.persister = p
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
	if level < l.minLevel {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = append(l.logs, LogEntry{Level: level, Message: msg})
	if len(l.logs) > maxLogs {
		l.logs = l.logs[len(l.logs)-maxLogs:]
	}

	if l.persister != nil {
		_ = l.persister.InsertLog(level.String(), msg)
	}
}

func (l *Logger) Logs() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.logs)
}
