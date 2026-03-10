package logstore

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aliencodez/botdo/internal/agent"
)

// FileLogStore stores one log file per task under logDir.
type FileLogStore struct {
	logDir string
}

// NewFileLogStore creates a FileLogStore that writes logs to logDir.
// The directory is created if it does not exist.
func NewFileLogStore(logDir string) (*FileLogStore, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("logstore: mkdir %s: %w", logDir, err)
	}
	return &FileLogStore{logDir: logDir}, nil
}

func (s *FileLogStore) logPath(taskID int64) string {
	return filepath.Join(s.logDir, fmt.Sprintf("%d.log", taskID))
}

// Writer returns a LineWriter that appends lines to <logDir>/<taskID>.log.
func (s *FileLogStore) Writer(taskID int64) agent.LineWriter {
	return &fileWriter{path: s.logPath(taskID)}
}

// Read returns all log lines for a task.
func (s *FileLogStore) Read(taskID int64) ([]string, error) {
	f, err := os.Open(s.logPath(taskID))
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

// Tail streams new bytes written to the log file until ctx is cancelled.
func (s *FileLogStore) Tail(ctx context.Context, taskID int64, w io.Writer) error {
	path := s.logPath(taskID)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		// Wait for file to appear.
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(100 * time.Millisecond):
			}
			f, err = os.Open(path)
			if err == nil {
				break
			}
		}
	} else if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return nil
			}
			if flusher, ok := w.(interface{ Flush() }); ok {
				flusher.Flush()
			}
		}
		if err == io.EOF {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(100 * time.Millisecond):
			}
			continue
		}
		if err != nil {
			return err
		}
	}
}

// fileWriter appends lines to a log file.
type fileWriter struct {
	path string
}

func (fw *fileWriter) WriteLine(line string) {
	f, err := os.OpenFile(fw.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, line)
}
