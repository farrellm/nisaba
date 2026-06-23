package handler

import (
	"net/http"
	"strings"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/llm"
	"github.com/farrellm/nisaba/internal/store"
)

// ListLabels returns the logged-in user's label names, ordered by name. Labels
// are a user-global taxonomy; this feeds the edit-labels dialog's pool of
// existing labels to apply to a document.
func ListLabels(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		labels, err := st.ListLabels(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load labels")
			return
		}

		names := make([]string, 0, len(labels))
		for _, l := range labels {
			names = append(names, l.Name)
		}
		writeJSON(w, http.StatusOK, names)
	}
}

// SuggestDocumentLabels suggests descriptive labels for a document by analyzing
// its "story" attribute with a fixed model (llm.SuggestLabels). It is read-only:
// it returns candidates for the caller to review and apply itself via
// PUT /api/documents/{id} — it does not attach anything.
func SuggestDocumentLabels(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
		if !ok {
			return
		}

		story := strings.TrimSpace(doc.Attributes["story"])
		if story == "" {
			writeError(w, http.StatusBadRequest, "Document has no story to label yet")
			return
		}

		labels, err := llm.SuggestLabels(r.Context(), story)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Model request failed")
			return
		}
		writeJSON(w, http.StatusOK, labels)
	}
}
