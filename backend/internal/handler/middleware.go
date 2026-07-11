package handler

import (
	"net/http"

	"github.com/farrellm/nisaba/internal/auth"
)

// RequireUser rejects requests without a valid session (401) and stores the
// logged-in user's id in the request context for handlers to read via
// auth.UserIDFrom.
func RequireUser(sess *auth.Sessions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := sess.UserID(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "Not logged in")
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithUserID(r.Context(), id)))
		})
	}
}
