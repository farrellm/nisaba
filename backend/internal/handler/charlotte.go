package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/store"
)

// ListCharlotteDocuments returns every document from the legacy file-based app as a
// summary for the read-only "Charlotte" browser. Like Anansi it requires a logged-in
// session but is not scoped to the caller — the legacy data is shared and read-only.
func ListCharlotteDocuments(cs *store.CharlotteStore, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := sess.UserID(r); !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		docs, err := cs.ListCharlotteDocuments(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load documents")
			return
		}
		writeJSON(w, http.StatusOK, docs)
	}
}

// GetCharlotteDocument returns a single legacy document fully populated read-only, 404
// if the id is out of range, or 500 if the legacy tool cannot parse it. Requires a
// logged-in session.
func GetCharlotteDocument(cs *store.CharlotteStore, sess *auth.Sessions) http.HandlerFunc {
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
		doc, err := cs.GetCharlotteDocument(r.Context(), id)
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
