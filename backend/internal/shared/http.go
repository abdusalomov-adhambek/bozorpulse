package shared

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Decode decodes a strictly-typed JSON body.
func Decode(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return WithStatus(errors.New("invalid JSON body: "+err.Error()), http.StatusBadRequest)
	}
	return nil
}

// WriteError maps domain errors to HTTP status codes.
func WriteError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	var de DomainError
	if errors.As(err, &de) {
		status = de.Status
	} else {
		switch {
		case errors.Is(err, ErrNotFound):
			status = http.StatusNotFound
		case errors.Is(err, ErrUnauthorized):
			status = http.StatusUnauthorized
		case errors.Is(err, ErrForbidden), errors.Is(err, ErrNotApproved):
			status = http.StatusForbidden
		case errors.Is(err, ErrConflict):
			status = http.StatusConflict
		case errors.Is(err, ErrInvalid):
			status = http.StatusBadRequest
		}
	}
	WriteJSON(w, status, map[string]string{"error": err.Error()})
}

// PageParams parses limit/offset query parameters with sane defaults.
func PageParams(r *http.Request) (limit, offset int) {
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return
}

// List is the standard list envelope.
type List[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
