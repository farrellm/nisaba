package handler

import (
	"net/http"

	"github.com/farrellm/nisaba/internal/auth"
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
