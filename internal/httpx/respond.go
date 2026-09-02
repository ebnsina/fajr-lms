// Package httpx holds shared HTTP plumbing: JSON encoding, errors, middleware.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// Error is the single error shape every endpoint returns.
type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func Errorf(status int, code, msg string) *Error {
	return &Error{Status: status, Code: code, Message: msg}
}

var (
	ErrNotFound     = Errorf(http.StatusNotFound, "not_found", "Resource not found.")
	ErrUnauthorized = Errorf(http.StatusUnauthorized, "unauthorized", "Authentication required.")
	ErrForbidden    = Errorf(http.StatusForbidden, "forbidden", "You do not have access to this resource.")
	ErrInternal     = Errorf(http.StatusInternalServerError, "internal", "Something went wrong on our side.")
)

// Handler is an http.Handler that returns an error instead of writing one.
type Handler func(http.ResponseWriter, *http.Request) error

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		WriteError(w, r, err)
	}
}

// JSON writes v with the given status, logging any encode failure.
func JSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return nil
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("httpx: encode response", "error", err)
		return nil // headers already sent; nothing left to say to the client
	}
	return nil
}

// NoContent ends a request with 204 and no body.
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// WriteError maps any error onto the Error shape, hiding internals from clients.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var e *Error
	if !errors.As(err, &e) {
		slog.ErrorContext(r.Context(), "unhandled error", "error", err, "path", r.URL.Path)
		e = ErrInternal
	}
	if e.Status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "server error", "code", e.Code, "error", err, "path", r.URL.Path)
	}
	_ = JSON(w, e.Status, map[string]any{"error": e})
}

// DecodeJSON reads a JSON body, rejecting unknown fields and oversized payloads.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" && !hasJSONType(ct) {
		return Errorf(http.StatusUnsupportedMediaType, "unsupported_media_type", "Send application/json.")
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()

	// An absent body means no fields were supplied; required-field checks still run.
	if err := dec.Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return Errorf(http.StatusRequestEntityTooLarge, "body_too_large", "Request body must be under 1 MB.")
		}
		return Errorf(http.StatusBadRequest, "invalid_json", "Request body is not valid JSON.")
	}
	if dec.More() {
		return Errorf(http.StatusBadRequest, "invalid_json", "Request body must contain a single JSON object.")
	}
	return nil
}

func hasJSONType(ct string) bool {
	for i := 0; i < len(ct); i++ {
		if ct[i] == ';' {
			ct = ct[:i]
			break
		}
	}
	return ct == "application/json"
}
