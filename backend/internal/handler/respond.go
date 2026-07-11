package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/farrellm/nisaba/internal/store"
)

// writeJSON sends v as a JSON body with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeError sends a JSON error body with the given status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

// internalError logs err and sends a 500 with msg. Use it wherever an
// unexpected error is in hand, so no 500 goes unlogged.
func internalError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	slog.Error(msg, "err", err, "method", r.Method, "path", r.URL.Path)
	writeError(w, http.StatusInternalServerError, msg)
}

// notFoundOr500 maps a read/write failure to 404 when the resource is missing
// (store.ErrNotFound) and to a logged 500 otherwise.
func notFoundOr500(w http.ResponseWriter, r *http.Request, err error, notFoundMsg, failMsg string) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, notFoundMsg)
		return
	}
	internalError(w, r, failMsg, err)
}

// decodeJSON decodes the request body into dst.
func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// pathID parses the named chi URL parameter as an int64 id.
func pathID(r *http.Request, param string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, param), 10, 64)
}
