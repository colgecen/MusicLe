// Package logging provides a small append-only activity log. It is deliberately
// dependency-free so every package can use it without creating import cycles.
package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// maxSize is the size at which the current log is rotated to <path>.1.
const maxSize = 2 << 20 // 2 MiB

var (
	mu   sync.Mutex
	file *os.File
	out  io.Writer
)

// SetFile starts logging to path. An empty path or "off" disables logging.
// "-" writes to stderr. The previous log file, if any, is closed first.
func SetFile(path string) error {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		_ = file.Close()
		file = nil
	}
	out = nil
	if path == "" || path == "off" {
		return nil
	}
	if path == "-" {
		out = os.Stderr
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if fi, err := os.Stat(path); err == nil && fi.Size() > maxSize {
		_ = os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	file = f
	out = f
	return nil
}

// Printf writes one timestamped line when logging is enabled.
func Printf(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if out == nil {
		return
	}
	fmt.Fprintf(out, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

// Close flushes and closes the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		_ = file.Close()
		file = nil
	}
	out = nil
}
