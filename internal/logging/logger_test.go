package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetEnabledDoesNotTruncateLogFileWhenStateIsUnchanged(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	resetLoggerStateForTest(t)

	if err := Init(true); err != nil {
		t.Fatalf("Init(true) returned error: %v", err)
	}

	logger := Get()
	logger.Info("first entry")

	path, err := logPathForTest()
	if err != nil {
		t.Fatalf("log path lookup failed: %v", err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read log file before SetEnabled: %v", err)
	}
	if !strings.Contains(string(before), "first entry") {
		t.Fatalf("expected initial log entry to be present, got %q", string(before))
	}

	if err := SetEnabled(true); err != nil {
		t.Fatalf("SetEnabled(true) returned error: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read log file after SetEnabled: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected log file to stay untouched on true->true, before=%q after=%q", string(before), string(after))
	}
}

func resetLoggerStateForTest(t *testing.T) {
	t.Helper()

	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()

		if logFd != nil {
			_ = logFd.Close()
			logFd = nil
		}
		logger = nil
		enabled = false
	})
}

func logPathForTest() (string, error) {
	dir, err := logDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, logFile), nil
}
