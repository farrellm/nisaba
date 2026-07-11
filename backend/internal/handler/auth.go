package handler

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/store"
)

const minPasswordLen = 8

const defaultSubreddit = "WritingPrompts"

// subredditPattern matches Reddit's allowed subreddit names (3-21 chars of
// letters, digits, and underscores).
var subredditPattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,21}$`)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// isUniqueViolation reports whether err is a Postgres unique-constraint error.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Register creates a new user, logs them in, and returns the user.
func Register(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var c credentials
		if err := decodeJSON(r, &c); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		c.Username = strings.TrimSpace(c.Username)
		if c.Username == "" {
			writeError(w, http.StatusBadRequest, "Username is required")
			return
		}
		if len(c.Password) < minPasswordLen {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Password must be at least %d characters", minPasswordLen))
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
		if err != nil {
			internalError(w, r, "Could not create account", err)
			return
		}

		user, err := st.CreateUser(r.Context(), c.Username, string(hash))
		if err != nil {
			if isUniqueViolation(err) {
				writeError(w, http.StatusConflict, "That username is taken")
				return
			}
			internalError(w, r, "Could not create account", err)
			return
		}

		if err := sess.Save(w, r, user.ID); err != nil {
			internalError(w, r, "Could not start session", err)
			return
		}
		writeJSON(w, http.StatusCreated, user)
	}
}

// Login verifies credentials, starts a session, and returns the user.
func Login(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var c credentials
		if err := decodeJSON(r, &c); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		c.Username = strings.TrimSpace(c.Username)

		id, hash, err := st.GetCredentialsByUsername(r.Context(), c.Username)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(c.Password)) != nil {
			// Generic message: don't reveal whether the username exists.
			writeError(w, http.StatusUnauthorized, "Incorrect username or password")
			return
		}

		if err := sess.Save(w, r, id); err != nil {
			internalError(w, r, "Could not start session", err)
			return
		}

		user, err := st.GetUser(r.Context(), id)
		if err != nil {
			internalError(w, r, "Could not load account", err)
			return
		}
		writeJSON(w, http.StatusOK, user)
	}
}

// Logout clears the session cookie.
func Logout(sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := sess.Clear(w, r); err != nil {
			internalError(w, r, "Could not log out", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// UpdateMe updates the logged-in user's settings and returns the refreshed user.
// Body fields are optional pointers, each applied only when present, so a caller
// can update one setting (e.g. the streaming toggle in the app menu) without
// clobbering the others.
func UpdateMe(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		var body struct {
			Subreddit        *string `json:"subreddit"`
			StreamingEnabled *bool   `json:"streamingEnabled"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate up front so a bad subreddit doesn't half-apply a multi-field
		// update.
		var subreddit string
		if body.Subreddit != nil {
			subreddit = strings.TrimSpace(*body.Subreddit)
			if subreddit == "" {
				subreddit = defaultSubreddit
			} else if !subredditPattern.MatchString(subreddit) {
				writeError(w, http.StatusBadRequest, "Invalid subreddit")
				return
			}
		}

		notFound := func(err error) bool {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusUnauthorized, "Not logged in")
				return true
			}
			if err != nil {
				internalError(w, r, "Could not update settings", err)
				return true
			}
			return false
		}

		if body.Subreddit != nil {
			if _, err := st.UpdateUserSubreddit(r.Context(), id, subreddit); notFound(err) {
				return
			}
		}
		if body.StreamingEnabled != nil {
			if _, err := st.UpdateUserStreamingEnabled(r.Context(), id, *body.StreamingEnabled); notFound(err) {
				return
			}
		}

		user, err := st.GetUser(r.Context(), id)
		if notFound(err) {
			return
		}
		writeJSON(w, http.StatusOK, user)
	}
}

// Me returns the currently logged-in user, or 401 if there is no valid session.
func Me(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		user, err := st.GetUser(r.Context(), id)
		if err != nil {
			// Session points at a user that no longer exists — treat as logged out.
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		writeJSON(w, http.StatusOK, user)
	}
}
