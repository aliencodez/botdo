package store

import (
	"errors"

	"github.com/aliencodez/botdo/internal/model"
)

// ErrNotFound is returned when a task does not exist.
var ErrNotFound = errors.New("task not found")

// ErrProjectNotFound is returned when a project does not exist.
var ErrProjectNotFound = errors.New("project not found")

// Filter holds optional query filters for listing tasks.
type Filter struct {
	Status    string
	Agent     string
	ProjectID int64 // 0 = no filter
}

// Store defines the persistence interface for tasks and projects.
type Store interface {
	CreateTask(t *model.Task) error
	GetTask(id int64) (*model.Task, error)
	ListTasks(f Filter) ([]*model.Task, error)
	UpdateTask(t *model.Task) error
	DeleteTask(id int64) error

	CreateProject(p *model.Project) error
	GetProject(id int64) (*model.Project, error)
	ListProjects() ([]*model.Project, error)
	UpdateProject(p *model.Project) error
	DeleteProject(id int64) error
}
