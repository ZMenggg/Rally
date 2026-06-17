package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// Level represents a log level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a level string.
func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// LogEntry represents a single log entry for the Web UI.
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// Logger is Rally's internal logger.
type Logger struct {
	level     Level
	backend   *log.Logger
	buf       []LogEntry
	mu        sync.RWMutex
	maxBuf    int
	listeners map[int]func(LogEntry)
	nextID    int
}

// Global singleton
var (
	global   *Logger
	globalMu sync.Mutex
)

// Init initializes the global logger.
func Init(levelStr, outputPath string) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	level := ParseLevel(levelStr)

	var w io.Writer = os.Stderr
	if outputPath != "" {
		f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		w = f
	}

	global = &Logger{
		level:     level,
		backend:   log.New(w, "", 0),
		buf:       make([]LogEntry, 0, 1000),
		maxBuf:    1000,
		listeners: make(map[int]func(LogEntry)),
	}
	return nil
}

// G returns the global logger.
func G() *Logger {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global == nil {
		// Auto-init with defaults
		global = &Logger{
			level:     LevelInfo,
			backend:   log.New(os.Stderr, "", 0),
			buf:       make([]LogEntry, 0, 1000),
			maxBuf:    1000,
			listeners: make(map[int]func(LogEntry)),
		}
	}
	return global
}

// SetLevel updates the log level at runtime.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Subscribe adds a listener that receives new log entries in real-time.
// Returns an unsubscribe function.
func (l *Logger) Subscribe(fn func(LogEntry)) func() {
	l.mu.Lock()
	defer l.mu.Unlock()
	id := l.nextID
	l.nextID++
	l.listeners[id] = fn
	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		delete(l.listeners, id)
	}
}

// Recent returns recent log entries.
func (l *Logger) Recent(n int) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if n <= 0 || n > len(l.buf) {
		n = len(l.buf)
	}
	out := make([]LogEntry, n)
	copy(out, l.buf[len(l.buf)-n:])
	return out
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	msg := fmt.Sprintf(format, args...)
	now := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] [%s] %s", now, level.String(), msg)

	l.mu.Lock()
	l.backend.Println(line)
	entry := LogEntry{Time: now, Level: level.String(), Message: msg}
	l.buf = append(l.buf, entry)
	if len(l.buf) > l.maxBuf {
		l.buf = l.buf[len(l.buf)-l.maxBuf:]
	}
	// Copy listeners map under lock
	listeners := make(map[int]func(LogEntry), len(l.listeners))
	for id, fn := range l.listeners {
		listeners[id] = fn
	}
	l.mu.Unlock()

	// Notify listeners outside lock
	for _, fn := range listeners {
		fn(entry)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) { l.log(LevelDebug, format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.log(LevelInfo, format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.log(LevelWarn, format, args...) }
func (l *Logger) Error(format string, args ...interface{}) { l.log(LevelError, format, args...) }

// Package-level convenience functions
func Debug(format string, args ...interface{}) { G().Debug(format, args...) }
func Info(format string, args ...interface{})  { G().Info(format, args...) }
func Warn(format string, args ...interface{})  { G().Warn(format, args...) }
func Error(format string, args ...interface{}) { G().Error(format, args...) }
