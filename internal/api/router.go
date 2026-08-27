package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/aliencodez/botdo/internal/logstore"
	"github.com/aliencodez/botdo/internal/store"
)

// Config controls optional hosted-mode features. APIKey protects all mutable
// workspace data; CheckoutURL is exposed to the UI as the upgrade destination.
type Config struct {
	APIKey      string
	CheckoutURL string
}

// NewRouter builds and returns the application HTTP router.
// ls may be nil (logs endpoint returns empty).
// The fs parameter should serve the embedded web/ directory.
func NewRouter(s store.Store, ls logstore.LogStore, fs http.FileSystem, configs ...Config) http.Handler {
	h := &handler{store: s, logs: ls}
	var cfg Config
	if len(configs) > 0 {
		cfg = configs[0]
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/api/config", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"auth_required": cfg.APIKey != "",
			"checkout_url":  cfg.CheckoutURL,
		})
	})
	r.Post("/api/session", func(w http.ResponseWriter, r *http.Request) {
		if cfg.APIKey == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !validAPIKey(body.Token, cfg.APIKey) {
			writeError(w, http.StatusUnauthorized, "valid workspace token required")
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "botdo_session",
			Value:    body.Token,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   60 * 60 * 24 * 30,
		})
		w.WriteHeader(http.StatusNoContent)
	})

	// REST API
	r.Route("/api", func(r chi.Router) {
		if cfg.APIKey != "" {
			r.Use(requireAPIKey(cfg.APIKey))
		}
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

func requireAPIKey(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := strings.TrimSpace(r.Header.Get("X-Botdo-Token"))
			if provided == "" {
				provided = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			}
			if provided == "" {
				if cookie, err := r.Cookie("botdo_session"); err == nil {
					provided = cookie.Value
				}
			}
			if !validAPIKey(provided, expected) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "valid workspace token required"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func validAPIKey(provided, expected string) bool {
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
