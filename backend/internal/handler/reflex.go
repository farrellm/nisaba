package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/store"
)

// ListReflexDocuments returns every legacy (reflex.db) document as a summary,
// newest first, for the read-only "Anansi" browser. It requires a logged-in
// session but is not scoped to the caller — the legacy users are unrelated to
// current accounts and the data is shared and read-only.
func ListReflexDocuments(rs *store.ReflexStore, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := sess.UserID(r); !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		docs, err := rs.ListReflexDocuments(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load documents")
			return
		}
		writeJSON(w, http.StatusOK, docs)
	}
}

// GetReflexDocument returns a single fully-populated legacy document read-only,
// or 404 if it does not exist. Requires a logged-in session.
func GetReflexDocument(rs *store.ReflexStore, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := sess.UserID(r); !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid document id")
			return
		}
		doc, err := rs.GetReflexDocument(r.Context(), id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "Document not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Could not load document")
			return
		}
		writeJSON(w, http.StatusOK, doc)
	}
}
