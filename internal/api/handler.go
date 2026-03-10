package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/aliencodez/botdo/internal/model"
	"github.com/aliencodez/botdo/internal/store"
)

type handler struct {
	store store.Store
}

func (h *handler) listTasks(w http.ResponseWriter, r *http.Request) {
	f := store.Filter{
		Status: r.URL.Query().Get("status"),
		Agent:  r.URL.Query().Get("agent"),
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
		Title       string          `json:"title"`
		Description string          `json:"description"`
		Agent       model.AgentType `json:"agent"`
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
		Title       *string          `json:"title"`
		Description *string          `json:"description"`
		Status      *model.Status    `json:"status"`
		Agent       *model.AgentType `json:"agent"`
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
