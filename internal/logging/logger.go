package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger wraps slog.Logger with hourly file rotation.
type Logger struct {
	slog  *slog.Logger
	w     *rotatingWriter
}

// rotatingWriter writes to a log file that rotates every hour.
type rotatingWriter struct {
	mu          sync.Mutex
	serviceName string
	logDir      string
	currentHour string
	file        *os.File
}

// New creates a Logger that writes JSON to log/<serviceName>-<YYYY-MM-DD-HH>.log,
// rotating to a new file each hour.
func New(serviceName string) (*Logger, error) {
	logDir := "log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	w := &rotatingWriter{
		serviceName: serviceName,
		logDir:      logDir,
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return &Logger{
		slog: slog.New(handler),
		w:    w,
	}, nil
}

// Info logs an info-level message with the given attributes.
func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

// Close closes the current log file.
func (l *Logger) Close() error {
	l.w.mu.Lock()
	defer l.w.mu.Unlock()
	if l.w.file != nil {
		return l.w.file.Close()
	}
	return nil
}

func (w *rotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	hour := time.Now().Format("2006-01-02-15")
	if hour != w.currentHour {
		if w.file != nil {
			w.file.Close()
		}
		filename := filepath.Join(w.logDir, w.serviceName+"-"+hour+".log")
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return 0, err
		}
		w.file = f
		w.currentHour = hour
	}

	return w.file.Write(p)
}
