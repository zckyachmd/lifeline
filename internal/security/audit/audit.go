package audit

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Logger writes append-only audit log.
type Logger struct {
	path string
	mu   sync.Mutex
}

// New creates audit logger.
func New(path string) *Logger {
	return &Logger{path: path}
}

// Write records an audit event.
func (l *Logger) Write(userID int64, command string, status string, meta map[string]string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = os.MkdirAll(filepathDir(l.path), 0o755)
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().UTC().Format(time.RFC3339)
	metaStr := ""
	for k, v := range meta {
		metaStr += fmt.Sprintf(" %s=%s", k, v)
	}
	line := fmt.Sprintf("%s user=%d cmd=%s status=%s%s\n", ts, userID, command, status, metaStr)
	_, _ = f.WriteString(line)
}

// filepathDir is isolated for testability.
func filepathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
