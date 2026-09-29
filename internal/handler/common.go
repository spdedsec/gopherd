package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/spdedsec/gopherd/internal/middleware"
	"github.com/spdedsec/gopherd/internal/repository"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decodeJSON(r *http.Request, max int64, dst any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, max+1))
	if err != nil {
		return errors.New("could not read request body")
	}
	if int64(len(body)) > max {
		return errors.New("request body too large")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid JSON body")
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func userID(r *http.Request) (string, bool) {
	u, ok := middleware.GetUser(r.Context())
	return u.ID, ok
}
func mapRepoError(err error) int {
	if errors.Is(err, repository.ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, repository.ErrConflict) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
