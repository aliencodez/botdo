package model

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

type AgentType string

const (
	AgentNone       AgentType = ""
	AgentClaudeCode AgentType = "claude-code"
	// AgentOpenCode AgentType = "opencode"  // to be added later
)

// Permissions controls which file-system tools the agent is allowed to use.
type Permissions struct {
	AllowWrite bool `json:"allow_write"`
	AllowEdit  bool `json:"allow_edit"`
}

type Task struct {
	ID          int64       `json:"id"`
	ProjectID   *int64      `json:"project_id,omitempty"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      Status      `json:"status"`
	Agent       AgentType   `json:"agent"`
	Permissions Permissions `json:"permissions"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}
