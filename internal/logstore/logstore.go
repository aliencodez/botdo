package logstore

import (
	"context"
	"io"

	"github.com/aliencodez/botdo/internal/agent"
)

// LogStore manages per-task log output.
type LogStore interface {
	Writer(taskID int64) agent.LineWriter
	Read(taskID int64) ([]string, error)
	Tail(ctx context.Context, taskID int64, w io.Writer) error
}
