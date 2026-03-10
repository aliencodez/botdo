package store

import (
	"errors"

	"github.com/aliencodez/botdo/internal/model"
)

// ErrNotFound is returned when a task does not exist.
var ErrNotFound = errors.New("task not found")

// Filter holds optional query filters for listing tasks.
type Filter struct {
	Status string
	Agent  string
}

// Store defines the persistence interface for tasks.
type Store interface {
	CreateTask(t *model.Task) error
	GetTask(id int64) (*model.Task, error)
	ListTasks(f Filter) ([]*model.Task, error)
	UpdateTask(t *model.Task) error
	DeleteTask(id int64) error
}
