package logging

import (
	"afk-hero/internal/appmeta"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
)

const (
	logFile     = "app.log"
	logFilePrev = "app.log.prev"
)

var (
	mu      sync.Mutex
	logger  *slog.Logger
	logFd   *os.File
	enabled bool
)

// Init initializes the logging system.
// If loggingEnabled is true, logs go to a file. Otherwise logging is a no-op.
//
// When a previous app.log exists, it is rotated to app.log.prev before the
// new session truncates the primary file. This preserves one session of
// history so post-mortem analysis after a crash is still possible.
func Init(loggingEnabled bool) error {
	mu.Lock()
	defer mu.Unlock()

	// Close previous file if open
	if logFd != nil {
		_ = logFd.Close()
		logFd = nil
	}

	enabled = loggingEnabled

	if !loggingEnabled {
		logger = slog.New(slog.NewTextHandler(nopWriter{}, &slog.HandlerOptions{Level: slog.LevelError + 1}))
		return nil
	}

	dir, err := logDir()
	if err != nil {
		return fmt.Errorf("log dir: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	path := filepath.Join(dir, logFile)
	rotateLogFile(path, filepath.Join(dir, logFilePrev))

	// Truncate on each app start (previous content is preserved via rotation above).
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	logFd = f
	logger = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return nil
}

// rotateLogFile moves the current log file aside as `app.log.prev` so the
// previous session's trace survives a restart. Failures are silent: rotation
// is best-effort and must never prevent the new session from logging.
func rotateLogFile(current, previous string) {
	info, err := os.Stat(current)
	if err != nil {
		return
	}
	if info.Size() == 0 {
		return
	}

	_ = os.Remove(previous)
	_ = os.Rename(current, previous)
}

// SetEnabled changes the logging state at runtime.
func SetEnabled(on bool) error {
	mu.Lock()
	alreadyInitialized := logger != nil && enabled == on && (!on || logFd != nil)
	mu.Unlock()

	if alreadyInitialized {
		return nil
	}

	return Init(on)
}

// Get returns the current logger instance.
func Get() *slog.Logger {
	mu.Lock()
	defer mu.Unlock()
	if logger == nil {
		return slog.New(slog.NewTextHandler(nopWriter{}, &slog.HandlerOptions{Level: slog.LevelError + 1}))
	}
	return logger
}

// Close cleans up the log file handle.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFd != nil {
		_ = logFd.Close()
		logFd = nil
	}
}

// Enabled returns whether logging is currently active.
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

// RecoverPanic logs a recovered panic together with the current stack trace.
func RecoverPanic(component string, attrs ...any) {
	recovered := recover()
	if recovered == nil {
		return
	}

	fields := make([]any, 0, len(attrs)+6)
	fields = append(fields,
		"component", component,
		"panic", fmt.Sprint(recovered),
		"stack", string(debug.Stack()),
	)
	fields = append(fields, attrs...)

	Get().Error("panic recovered", fields...)
}

func logDir() (string, error) {
	userConfig, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userConfig, appmeta.DirName, "logs"), nil
}

// nopWriter discards all writes.
type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }
