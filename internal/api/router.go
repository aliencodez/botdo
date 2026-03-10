package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/aliencodez/botdo/internal/logstore"
	"github.com/aliencodez/botdo/internal/store"
)

// NewRouter builds and returns the application HTTP router.
// ls may be nil (logs endpoint returns empty).
// The fs parameter should serve the embedded web/ directory.
func NewRouter(s store.Store, ls logstore.LogStore, fs http.FileSystem) http.Handler {
	h := &handler{store: s, logs: ls}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// REST API
	r.Route("/api", func(r chi.Router) {
		r.Get("/tasks", h.listTasks)
		r.Post("/tasks", h.createTask)
		r.Get("/tasks/{id}", h.getTask)
		r.Put("/tasks/{id}", h.updateTask)
		r.Delete("/tasks/{id}", h.deleteTask)

		r.Post("/tasks/{id}/claim", h.claimTask)
		r.Post("/tasks/{id}/complete", h.completeTask)
		r.Post("/tasks/{id}/retry", h.retryTask)
		r.Get("/tasks/{id}/logs", h.getTaskLogs)

		r.Get("/agents/{agent}/tasks", h.agentTasks)

		r.Get("/projects", h.listProjects)
		r.Post("/projects", h.createProject)
		r.Get("/projects/{id}", h.getProject)
		r.Put("/projects/{id}", h.updateProject)
		r.Delete("/projects/{id}", h.deleteProject)
	})

	// Embedded web UI — serve everything else from the static file system
	r.Handle("/*", http.FileServer(fs))

	return r
}
