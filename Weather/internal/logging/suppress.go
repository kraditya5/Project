package logging

import (
	"os"
	"strings"
	"sync"
)

// LogSuppressor provides io.Writer interface for logging
// with lines suppression. For usage with log.Logger.
type LogSuppressor struct {
	filename   string
	suppress   []string
	linePrefix string

	logFile *os.File
	m       sync.Mutex
}

// NewLogSuppressor creates a new LogSuppressor for specified
// filename and lines to be suppressed.
//
// If filename is empty, log entries will be printed to stderr.
func NewLogSuppressor(filename string, suppress []string, linePrefix string) *LogSuppressor {
	return &LogSuppressor{
		filename:   filename,
		suppress:   suppress,
		linePrefix: linePrefix,
	}
}

// Open opens log file.
func (ls *LogSuppressor) Open() error {
	var err error

	if ls.filename == "" {
		return nil
	}

	//nolint:nosnakecase
	ls.logFile, err = os.OpenFile(ls.filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)

	return err
}

// Close closes log file.
func (ls *LogSuppressor) Close() error {
	if ls.filename == "" {
		return nil
	}

	return ls.logFile.Close()
}

// Write writes p to log, and returns number f bytes written.
// Implements io.Writer interface.
func (ls *LogSuppressor) Write(p []byte) (int, error) {
	var output string

	if ls.filename == "" {
		return os.Stdin.Write(p)
	}

	ls.m.Lock()
	defer ls.m.Unlock()

	lines := strings.Split(string(p), ls.linePrefix)
	for _, line := range lines {
		if (func(line string) bool {
			for _, suppress := range ls.suppress {
				if strings.Contains(line, suppress) {
					return true
				}
			}

			return false
		})(line) {
			continue
		}
		output += line
	}

	return ls.logFile.Write([]byte(output))
}
