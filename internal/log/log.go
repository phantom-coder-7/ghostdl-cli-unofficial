package log

import (
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	loggerMu      sync.RWMutex
	defaultLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
)

// Default returns the package-level slog logger.
func Default() *slog.Logger {
	loggerMu.RLock()
	l := defaultLogger
	loggerMu.RUnlock()
	return l
}

// Setup configures debug logging. When verbose is false and logFile is empty,
// debug output is discarded. logFile implies verbose. Returns a cleanup function
// that closes any opened log file; call it when the command finishes.
func Setup(verbose bool, logFile string, errOut io.Writer) (func(), error) {
	if logFile != "" {
		verbose = true
	}

	if !verbose {
		loggerMu.Lock()
		defaultLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
		loggerMu.Unlock()
		return func() {}, nil
	}

	writers := []io.Writer{errOut}
	var file *os.File
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		file = f
		writers = append(writers, file)
	}

	w := io.MultiWriter(writers...)
	loggerMu.Lock()
	defaultLogger = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
	loggerMu.Unlock()

	cleanup := func() {
		if file != nil {
			_ = file.Close()
		}
	}
	return cleanup, nil
}

// SetDefault replaces the package-level logger (for tests).
func SetDefault(l *slog.Logger) {
	loggerMu.Lock()
	defaultLogger = l
	loggerMu.Unlock()
}
