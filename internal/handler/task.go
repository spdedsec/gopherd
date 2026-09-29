package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/spdedsec/gopherd/internal/model"
	"github.com/spdedsec/gopherd/internal/service"
)

type TaskHandler struct {
	svc     *service.TaskService
	maxBody int64
}

func NewTaskHandler(s *service.TaskService, max int64) *TaskHandler {
	return &TaskHandler{svc: s, maxBody: max}
}

type taskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		writeJSON(w, 401, map[string]any{"error": "unauthorized"})
		return
	}
	limit := 50
	offset := 0
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && r.URL.Query().Get("limit") != "" {
		limit = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && r.URL.Query().Get("offset") != "" {
		offset = v
	}
	var completed *bool
	if v := r.URL.Query().Get("completed"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			writeJSON(w, 400, map[string]any{"error": "completed must be true or false"})
			return
		}
		completed = &b
	}
	items, err := h.svc.List(r.Context(), uid, completed, limit, offset)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "failed to list tasks"})
		return
	}
	if items == nil {
		items = []model.Task{}
	}
	writeJSON(w, 200, map[string]any{"items": items, "limit": limit, "offset": offset})
}
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		writeJSON(w, 401, map[string]any{"error": "unauthorized"})
		return
	}
	var in taskInput
	if err := decodeJSON(r, h.maxBody, &in); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	t, err := h.svc.Create(r.Context(), uid, in.Title, in.Description)
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 201, t)
}
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		writeJSON(w, 401, map[string]any{"error": "unauthorized"})
		return
	}
	t, err := h.svc.Get(r.Context(), uid, chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, mapRepoError(err), map[string]any{"error": "task not found"})
		return
	}
	writeJSON(w, 200, t)
}
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		writeJSON(w, 401, map[string]any{"error": "unauthorized"})
		return
	}
	var in taskInput
	if err := decodeJSON(r, h.maxBody, &in); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	t, err := h.svc.Update(r.Context(), uid, chi.URLParam(r, "id"), in.Title, in.Description, in.Completed)
	if err != nil {
		status := mapRepoError(err)
		if status == 500 {
			status = 400
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, t)
}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		writeJSON(w, 401, map[string]any{"error": "unauthorized"})
		return
	}
	err := h.svc.Delete(r.Context(), uid, chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, mapRepoError(err), map[string]any{"error": "task not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
