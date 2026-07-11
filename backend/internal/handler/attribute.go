package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/farrellm/nisaba/internal/auth"
)

// AttributeStore is the consumer-side view of the data layer the attribute
// handler uses.
type AttributeStore interface {
	ListAttributeValues(ctx context.Context, userID int64, key string) ([]string, error)
}

// ListAttributeValues returns the logged-in user's distinct past values for the
// attribute key given in ?key=, alphabetically sorted. Used to populate
// autocomplete suggestions (e.g. author names) when editing a block.
func ListAttributeValues(st AttributeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFrom(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		key := strings.TrimSpace(r.URL.Query().Get("key"))
		if key == "" {
			writeError(w, http.StatusBadRequest, "key is required")
			return
		}

		values, err := st.ListAttributeValues(r.Context(), userID, key)
		if err != nil {
			internalError(w, r, "Could not load attribute values", err)
			return
		}
		writeJSON(w, http.StatusOK, values)
	}
}
