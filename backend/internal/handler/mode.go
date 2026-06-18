package handler

import (
	"net/http"

	"github.com/farrellm/nisaba/internal/mode"
)

// ListModes returns the fixed set of writing modes (name, label, keys, output).
// The mustache templates are intentionally omitted from the response — prompt
// assembly happens server-side.
func ListModes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mode.All())
	}
}
