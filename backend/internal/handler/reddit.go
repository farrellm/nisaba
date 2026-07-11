package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/reddit"
	"github.com/farrellm/nisaba/internal/store"
)

// writeRedditError maps a reddit.Client error onto the HTTP response.
// notFoundMsg names the missing resource (subreddit vs post) for 404s.
func writeRedditError(w http.ResponseWriter, r *http.Request, err error, notFoundMsg string) {
	var statusErr *reddit.StatusError
	var submitErr *reddit.SubmitError
	switch {
	case errors.Is(err, reddit.ErrInvalidURL):
		writeError(w, http.StatusBadRequest, "Not a Reddit URL")
	case errors.Is(err, reddit.ErrShareResolve):
		writeError(w, http.StatusBadGateway, "Could not resolve Reddit share link")
	case errors.Is(err, reddit.ErrAuth):
		writeError(w, http.StatusBadGateway, "Could not authenticate with Reddit")
	case errors.Is(err, reddit.ErrUnreachable):
		writeError(w, http.StatusBadGateway, "Could not reach Reddit")
	case errors.Is(err, reddit.ErrNotFound):
		writeError(w, http.StatusNotFound, notFoundMsg)
	case errors.Is(err, reddit.ErrRateLimited):
		writeError(w, http.StatusTooManyRequests, "Reddit is rate-limiting requests; try again shortly")
	case errors.Is(err, reddit.ErrRejected):
		writeError(w, http.StatusBadGateway, "Reddit rejected the request; try again")
	case errors.Is(err, reddit.ErrBadResponse):
		writeError(w, http.StatusBadGateway, "Unexpected response from Reddit")
	case errors.As(err, &statusErr):
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Reddit returned an unexpected status (%d)", statusErr.Code))
	case errors.As(err, &submitErr):
		writeError(w, http.StatusBadRequest, submitErr.Message)
	default:
		// Only request construction can land here; it never should.
		internalError(w, r, "Could not build request", err)
	}
}

// ListRedditPosts fetches the newest posts from the logged-in user's configured
// subreddit via Reddit's application-only OAuth API and returns their titles and
// permalink URLs. Anonymous access is blocked by Reddit, so clientID/secret from
// a registered app are required.
func ListRedditPosts(st *store.Store, rc *reddit.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := auth.UserIDFrom(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		if !rc.Configured() {
			writeError(w, http.StatusServiceUnavailable, "Reddit integration is not configured")
			return
		}

		user, err := st.GetUser(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		posts, err := rc.NewestPosts(r.Context(), user.Subreddit)
		if err != nil {
			writeRedditError(w, r, err, "Subreddit not found")
			return
		}
		writeJSON(w, http.StatusOK, posts)
	}
}

// GetRedditPost fetches a single Reddit post by URL via Reddit's application-only
// OAuth API and returns its title and normalized permalink URL. It lets users
// import a specific post that isn't in the subreddit's newest listing.
func GetRedditPost(rc *reddit.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserIDFrom(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		if !rc.Configured() {
			writeError(w, http.StatusServiceUnavailable, "Reddit integration is not configured")
			return
		}

		raw := r.URL.Query().Get("url")
		if strings.TrimSpace(raw) == "" {
			writeError(w, http.StatusBadRequest, "Missing url")
			return
		}

		post, err := rc.FetchPost(r.Context(), raw)
		if err != nil {
			writeRedditError(w, r, err, "Post not found")
			return
		}
		writeJSON(w, http.StatusOK, post)
	}
}

// SubmitRedditPost publishes a self (text) post to the document owner's
// configured subreddit as the script-app account (the password grant) and
// records the resulting permalink on the document. Reading uses an
// application-only token, which has no account identity and cannot submit, so
// this needs REDDIT_USERNAME/PASSWORD in addition to the app credentials and
// reports 503 when they're absent. It is nested under /documents/{id} so the
// returned post URL is saved against that document.
func SubmitRedditPost(st *store.Store, rc *reddit.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st)
		if !ok {
			return
		}
		if !rc.CanSubmit() {
			writeError(w, http.StatusServiceUnavailable, "Reddit posting is not configured")
			return
		}

		var body struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		title := strings.TrimSpace(body.Title)
		if title == "" {
			writeError(w, http.StatusBadRequest, "Missing title")
			return
		}

		user, err := st.GetUser(r.Context(), doc.UserID)
		if err != nil {
			internalError(w, r, "Could not load user", err)
			return
		}

		postURL, err := rc.SubmitSelfPost(r.Context(), user.Subreddit, title, body.Body)
		if err != nil {
			writeRedditError(w, r, err, "Post not found")
			return
		}

		if postURL != "" {
			if err := st.AddDocumentPost(r.Context(), doc.ID, postURL); err != nil {
				internalError(w, r, "Posted, but could not save the post URL", err)
				return
			}
		}

		updated, err := st.GetDocument(r.Context(), doc.ID)
		if err != nil {
			internalError(w, r, "Could not load document", err)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}
