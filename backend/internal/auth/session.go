// Package auth manages the signed session cookie that tracks the logged-in user.
// Only the user id is stored in the session; it is signed (not secret) and the
// cookie is HttpOnly so browser JavaScript cannot read or forge it.
package auth

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	sessionName  = "nisaba_session"
	userIDKey    = "uid"
	maxAgeSecond = 60 * 60 * 24 * 7 // one week
)

// Sessions wraps a cookie-backed session store with helpers for the user id.
type Sessions struct {
	store *sessions.CookieStore
}

// NewSessions builds a session manager. Pass secure=true in production so the
// cookie is only sent over HTTPS.
func NewSessions(secret string, secure bool) *Sessions {
	store := sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   maxAgeSecond,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
	return &Sessions{store: store}
}

// Save writes the user id into the session cookie.
func (s *Sessions) Save(w http.ResponseWriter, r *http.Request, userID int64) error {
	sess, _ := s.store.Get(r, sessionName)
	sess.Values[userIDKey] = userID
	return sess.Save(r, w)
}

// UserID returns the logged-in user id, or false if there is no valid session.
func (s *Sessions) UserID(r *http.Request) (int64, bool) {
	sess, err := s.store.Get(r, sessionName)
	if err != nil {
		return 0, false
	}
	id, ok := sess.Values[userIDKey].(int64)
	return id, ok
}

// Clear expires the session cookie, logging the user out.
func (s *Sessions) Clear(w http.ResponseWriter, r *http.Request) error {
	sess, _ := s.store.Get(r, sessionName)
	sess.Options.MaxAge = -1
	delete(sess.Values, userIDKey)
	return sess.Save(r, w)
}
