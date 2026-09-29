package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct{ db *pgxpool.Pool }

func NewHealthHandler(db *pgxpool.Pool) *HealthHandler { return &HealthHandler{db: db} }
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.db.Ping(ctx); err != nil {
		writeJSON(w, 503, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ready"})
}
