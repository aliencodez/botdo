package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/aliencodez/botdo/internal/store"
)

// NewRouter builds and returns the application HTTP router.
// The fs parameter should serve the embedded web/ directory.
func NewRouter(s store.Store, fs http.FileSystem) http.Handler {
	h := &handler{store: s}

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

		r.Get("/agents/{agent}/tasks", h.agentTasks)
	})

	// Embedded web UI — serve everything else from the static file system
	r.Handle("/*", http.FileServer(fs))

	return r
}
