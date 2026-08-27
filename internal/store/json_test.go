package store

import (
	"testing"

	"github.com/aliencodez/botdo/internal/model"
)

func TestDeleteProjectUnassignsTasks(t *testing.T) {
	s, err := NewJSONStore(t.TempDir() + "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	project := &model.Project{Name: "Customer app"}
	if err := s.CreateProject(project); err != nil {
		t.Fatal(err)
	}
	task := &model.Task{Title: "Fix checkout", ProjectID: &project.ID}
	if err := s.CreateTask(task); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteProject(project.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetTask(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectID != nil {
		t.Fatalf("task ProjectID = %d, want nil", *got.ProjectID)
	}
	if !got.UpdatedAt.After(task.CreatedAt) {
		t.Fatalf("task UpdatedAt = %v, want after %v", got.UpdatedAt, task.CreatedAt)
	}
}
