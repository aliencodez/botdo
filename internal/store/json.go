package store

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aliencodez/botdo/internal/model"
)

// JSONStore implements Store using a JSON file on disk.
// All writes are protected by a mutex and flushed atomically.
type JSONStore struct {
	mu   sync.Mutex
	path string
	data *storeData
}

type storeData struct {
	NextID        int64            `json:"next_id"`
	NextProjectID int64            `json:"next_project_id"`
	Tasks         []*model.Task    `json:"tasks"`
	Projects      []*model.Project `json:"projects"`
}

// Compile-time check that *JSONStore implements Store.
var _ Store = (*JSONStore)(nil)

// NewJSONStore opens (or creates) a JSON file store at path.
func NewJSONStore(path string) (Store, error) {
	s := &JSONStore{path: path, data: &storeData{NextID: 1, NextProjectID: 1}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *JSONStore) load() error {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil // start fresh
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, s.data)
}

func (s *JSONStore) flush() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *JSONStore) CreateTask(t *model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	t.ID = s.data.NextID
	s.data.NextID++
	if t.Status == "" {
		t.Status = model.StatusPending
	}
	t.CreatedAt = now
	t.UpdatedAt = now
	s.data.Tasks = append(s.data.Tasks, t)
	return s.flush()
}

func (s *JSONStore) GetTask(id int64) (*model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.data.Tasks {
		if t.ID == id {
			cp := *t
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (s *JSONStore) ListTasks(f Filter) ([]*model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []*model.Task
	for _, t := range s.data.Tasks {
		if f.Status != "" && string(t.Status) != f.Status {
			continue
		}
		if f.Agent != "" && !strings.EqualFold(string(t.Agent), f.Agent) {
			continue
		}
		if f.ProjectID != 0 {
			if t.ProjectID == nil || *t.ProjectID != f.ProjectID {
				continue
			}
		}
		cp := *t
		result = append(result, &cp)
	}
	return result, nil
}

func (s *JSONStore) UpdateTask(t *model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t.UpdatedAt = time.Now().UTC()
	for i, existing := range s.data.Tasks {
		if existing.ID == t.ID {
			s.data.Tasks[i] = t
			return s.flush()
		}
	}
	return ErrNotFound
}

func (s *JSONStore) DeleteTask(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.data.Tasks {
		if t.ID == id {
			s.data.Tasks = append(s.data.Tasks[:i], s.data.Tasks[i+1:]...)
			return s.flush()
		}
	}
	return ErrNotFound
}

func (s *JSONStore) CreateProject(p *model.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	p.ID = s.data.NextProjectID
	s.data.NextProjectID++
	p.CreatedAt = now
	p.UpdatedAt = now
	s.data.Projects = append(s.data.Projects, p)
	return s.flush()
}

func (s *JSONStore) GetProject(id int64) (*model.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range s.data.Projects {
		if p.ID == id {
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrProjectNotFound
}

func (s *JSONStore) ListProjects() ([]*model.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]*model.Project, len(s.data.Projects))
	for i, p := range s.data.Projects {
		cp := *p
		result[i] = &cp
	}
	return result, nil
}

func (s *JSONStore) UpdateProject(p *model.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p.UpdatedAt = time.Now().UTC()
	for i, existing := range s.data.Projects {
		if existing.ID == p.ID {
			s.data.Projects[i] = p
			return s.flush()
		}
	}
	return ErrProjectNotFound
}

func (s *JSONStore) DeleteProject(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, p := range s.data.Projects {
		if p.ID == id {
			s.data.Projects = append(s.data.Projects[:i], s.data.Projects[i+1:]...)
			now := time.Now().UTC()
			for _, task := range s.data.Tasks {
				if task.ProjectID != nil && *task.ProjectID == id {
					task.ProjectID = nil
					task.UpdatedAt = now
				}
			}
			return s.flush()
		}
	}
	return ErrProjectNotFound
}
