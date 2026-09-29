package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/spdedsec/gopherd/internal/repository"

	"github.com/spdedsec/gopherd/internal/middleware"
	"github.com/spdedsec/gopherd/internal/service"
)

type AuthHandler struct {
	svc     *service.AuthService
	maxBody int64
}

func NewAuthHandler(s *service.AuthService, max int64) *AuthHandler {
	return &AuthHandler{svc: s, maxBody: max}
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type authResponse struct {
	Token string `json:"token"`
	User  any    `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if err := decodeJSON(r, h.maxBody, &in); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	u, t, err := h.svc.Register(r.Context(), in.Email, in.Password)
	if err != nil {
		status := 500
		if errors.Is(err, service.ErrInvalidCredentials) {
			status = 400
		}
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "password") {
			status = 400
		}
		if errors.Is(err, repository.ErrConflict) {
			status = 409
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 201, authResponse{Token: t, User: u})
}
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if err := decodeJSON(r, h.maxBody, &in); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	u, t, err := h.svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		writeJSON(w, 401, map[string]any{"error": "invalid email or password"})
		return
	}
	writeJSON(w, 200, authResponse{Token: t, User: u})
}
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := middleware.Bearer(r)
	if token == "" {
		writeJSON(w, 401, map[string]any{"error": "missing bearer token"})
		return
	}
	if err := h.svc.Logout(r.Context(), token); err != nil {
		writeJSON(w, 500, map[string]any{"error": "logout failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	u, ok := middleware.GetUser(r.Context())
	if !ok {
		writeJSON(w, 401, map[string]any{"error": "unauthorized"})
		return
	}
	writeJSON(w, 200, u)
}
