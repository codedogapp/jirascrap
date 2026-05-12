package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// Interface defines the logging contract. Consumers should depend on this
// rather than the concrete Logger type to enable testing and DI.
type Interface interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Debug(msg string)
	Logs() []LogEntry
}

type Logger struct {
	mu       sync.Mutex
	logs     []LogEntry
	writer   io.Writer
	minLevel Level
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

// maxLogFiles is the number of session log files to keep.
const maxLogFiles = 10

// Log is the default global logger instance. Prefer injecting Interface where possible.
var Log Interface = &Logger{minLevel: DEBUG}

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

// SetOutput sets a writer for file-based logging. All log entries will be
// written to this writer in addition to the in-memory buffer.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer = w
}

// Close flushes and closes the underlying writer if it implements io.Closer.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if c, ok := l.writer.(io.Closer); ok {
		return c.Close()
	}
	return nil
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

	if l.writer != nil {
		ts := time.Now().Format("2006-01-02 15:04:05.000")
		fmt.Fprintf(l.writer, "%s [%-5s] %s\n", ts, level, msg)
	}
}

func (l *Logger) Logs() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.logs)
}

// OpenSessionLog creates a log directory and opens a timestamped log file.
// It also prunes old log files beyond maxLogFiles. Returns the file (caller
// must close) and the path for display purposes.
func OpenSessionLog(dir string) (*os.File, string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", fmt.Errorf("create log dir: %w", err)
	}

	pruneOldLogs(dir)

	name := fmt.Sprintf("session-%s.log", time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, name)

	f, err := os.Create(path)
	if err != nil {
		return nil, "", fmt.Errorf("create log file: %w", err)
	}

	return f, path, nil
}

func pruneOldLogs(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var logFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "session-") && strings.HasSuffix(e.Name(), ".log") {
			logFiles = append(logFiles, e.Name())
		}
	}

	if len(logFiles) <= maxLogFiles {
		return
	}

	sort.Strings(logFiles)
	for _, name := range logFiles[:len(logFiles)-maxLogFiles] {
		os.Remove(filepath.Join(dir, name))
	}
}
