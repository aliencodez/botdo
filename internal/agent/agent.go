package agent

import (
	"context"

	"github.com/aliencodez/botdo/internal/model"
)

// LineWriter receives log lines from an agent execution.
type LineWriter interface {
	WriteLine(line string)
}

// Adapter executes a task on behalf of a specific agent type.
type Adapter interface {
	AgentType() string
	Run(ctx context.Context, taskID int64, prompt, workDir string, perms model.Permissions, out LineWriter) error
}
