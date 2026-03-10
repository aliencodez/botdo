package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/aliencodez/botdo/internal/logstore"
	"github.com/aliencodez/botdo/internal/model"
	"github.com/aliencodez/botdo/internal/store"
)

type handler struct {
	store store.Store
	logs  logstore.LogStore
}

func (h *handler) listTasks(w http.ResponseWriter, r *http.Request) {
	f := store.Filter{
		Status: r.URL.Query().Get("status"),
		Agent:  r.URL.Query().Get("agent"),
	}
	if pid := r.URL.Query().Get("project_id"); pid != "" {
		if v, err := strconv.ParseInt(pid, 10, 64); err == nil {
			f.ProjectID = v
		}
	}
	tasks, err := h.store.ListTasks(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tasks == nil {
		tasks = []*model.Task{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *handler) createTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string           `json:"title"`
		Description string           `json:"description"`
		Agent       model.AgentType  `json:"agent"`
		ProjectID   *int64           `json:"project_id"`
		Permissions model.Permissions `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	t := &model.Task{
		Title:       body.Title,
		Description: body.Description,
		Agent:       body.Agent,
		ProjectID:   body.ProjectID,
		Permissions: body.Permissions,
		Status:      model.StatusPending,
	}
	if err := h.store.CreateTask(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *handler) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.store.GetTask(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *handler) updateTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.store.GetTask(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var body struct {
		Title       *string           `json:"title"`
		Description *string           `json:"description"`
		Status      *model.Status     `json:"status"`
		Agent       *model.AgentType  `json:"agent"`
		ProjectID   *int64            `json:"project_id"`
		Permissions *model.Permissions `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Title != nil {
		t.Title = *body.Title
	}
	if body.Description != nil {
		t.Description = *body.Description
	}
	if body.Status != nil {
		t.Status = *body.Status
	}
	if body.Agent != nil {
		t.Agent = *body.Agent
	}
	if body.ProjectID != nil {
		t.ProjectID = body.ProjectID
	}
	if body.Permissions != nil {
		t.Permissions = *body.Permissions
	}
	if err := h.store.UpdateTask(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteTask(id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Project handlers ──────────────────────────────────────────────────────

func (h *handler) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.store.ListProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []*model.Project{}
	}
	writeJSON(w, http.StatusOK, projects)
}

func (h *handler) createProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		WorkDir string `json:"work_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	p := &model.Project{
		Name:    body.Name,
		WorkDir: body.WorkDir,
	}
	if err := h.store.CreateProject(p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *handler) getProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.store.GetProject(id)
	if errors.Is(err, store.ErrProjectNotFound) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *handler) updateProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.store.GetProject(id)
	if errors.Is(err, store.ErrProjectNotFound) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var body struct {
		Name    *string `json:"name"`
		WorkDir *string `json:"work_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Name != nil {
		p.Name = *body.Name
	}
	if body.WorkDir != nil {
		p.WorkDir = *body.WorkDir
	}
	if err := h.store.UpdateProject(p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteProject(id); errors.Is(err, store.ErrProjectNotFound) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// agentTasks returns pending + in_progress tasks for a specific agent (polling endpoint).
func (h *handler) agentTasks(w http.ResponseWriter, r *http.Request) {
	agentName := chi.URLParam(r, "agent")
	all, err := h.store.ListTasks(store.Filter{Agent: agentName})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var result []*model.Task
	for _, t := range all {
		if t.Status == model.StatusPending || t.Status == model.StatusInProgress {
			result = append(result, t)
		}
	}
	if result == nil {
		result = []*model.Task{}
	}
	writeJSON(w, http.StatusOK, result)
}

// claimTask sets a task's status to in_progress (agent picks it up).
func (h *handler) claimTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.store.GetTask(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t.Status != model.StatusPending {
		writeError(w, http.StatusConflict, "task is not pending")
		return
	}
	t.Status = model.StatusInProgress
	if err := h.store.UpdateTask(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// retryTask resets a failed task back to pending so it can be re-dispatched.
func (h *handler) retryTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.store.GetTask(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t.Status != model.StatusFailed {
		writeError(w, http.StatusConflict, "task is not failed")
		return
	}
	t.Status = model.StatusPending
	if err := h.store.UpdateTask(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// completeTask lets an agent mark a task done or failed.
func (h *handler) completeTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.store.GetTask(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var body struct {
		Status model.Status `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Status != model.StatusDone && body.Status != model.StatusFailed {
		writeError(w, http.StatusBadRequest, "status must be 'done' or 'failed'")
		return
	}
	t.Status = body.Status
	if err := h.store.UpdateTask(t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// getTaskLogs returns log lines for a task.
// With ?stream=1, streams as SSE. Without, returns JSON.
func (h *handler) getTaskLogs(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if h.logs == nil {
		writeJSON(w, http.StatusOK, map[string][]string{"lines": {}})
		return
	}
	if r.URL.Query().Get("stream") == "1" {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// Wrap the writer so Tail can flush and format SSE lines.
		sw := &sseWriter{w: w}
		if err := h.logs.Tail(r.Context(), id, sw); err != nil {
			return
		}
		return
	}
	lines, err := h.logs.Read(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if lines == nil {
		lines = []string{}
	}
	writeJSON(w, http.StatusOK, map[string][]string{"lines": lines})
}

// sseWriter wraps http.ResponseWriter to emit SSE-formatted data events
// line-by-line. It buffers incomplete lines and flushes after each newline.
type sseWriter struct {
	w   http.ResponseWriter
	buf []byte
}

func (s *sseWriter) Write(p []byte) (int, error) {
	s.buf = append(s.buf, p...)
	for {
		idx := -1
		for i, b := range s.buf {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx == -1 {
			break
		}
		line := string(s.buf[:idx])
		s.buf = s.buf[idx+1:]
		fmt.Fprintf(s.w, "data: %s\n\n", line)
		if f, ok := s.w.(http.Flusher); ok {
			f.Flush()
		}
	}
	return len(p), nil
}

func (s *sseWriter) Flush() {
	if f, ok := s.w.(http.Flusher); ok {
		f.Flush()
	}
}

// helpers

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
