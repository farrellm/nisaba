package handler

import (
	"net/http"

	"github.com/farrellm/nisaba/internal/llm"
)

// ListModels returns the fixed, cross-provider list of selectable models.
// Distinct from ListModes (GET /api/modes), which returns the writing modes.
func ListModels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, llm.Models())
	}
}
