package handler

import (
	"net/http"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/store"
)

// ListReflexDocuments returns every legacy (reflex.db) document as a summary,
// newest first, for the read-only "Anansi" browser. It requires a logged-in
// session but is not scoped to the caller — the legacy users are unrelated to
// current accounts and the data is shared and read-only.
func ListReflexDocuments(rs *store.ReflexStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserIDFrom(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		docs, err := rs.ListReflexDocuments(r.Context())
		if err != nil {
			internalError(w, r, "Could not load documents", err)
			return
		}
		writeJSON(w, http.StatusOK, docs)
	}
}

// GetReflexDocument returns a single fully-populated legacy document read-only,
// or 404 if it does not exist. Requires a logged-in session.
func GetReflexDocument(rs *store.ReflexStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserIDFrom(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid document id")
			return
		}
		doc, err := rs.GetReflexDocument(r.Context(), id)
		if err != nil {
			notFoundOr500(w, r, err, "Document not found", "Could not load document")
			return
		}
		writeJSON(w, http.StatusOK, doc)
	}
}
