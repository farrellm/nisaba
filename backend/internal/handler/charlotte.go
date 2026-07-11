package handler

import (
	"net/http"

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
			internalError(w, r, "Could not load documents", err)
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
		id, err := pathID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid document id")
			return
		}
		doc, err := cs.GetCharlotteDocument(r.Context(), id)
		if err != nil {
			notFoundOr500(w, r, err, "Document not found", "Could not load document")
			return
		}
		writeJSON(w, http.StatusOK, doc)
	}
}
