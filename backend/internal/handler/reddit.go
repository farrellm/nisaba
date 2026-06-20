package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/store"
)

// redditClient is the shared HTTP client for outbound Reddit requests. It has an
// explicit timeout (unlike http.DefaultClient) so a slow upstream can't hang the
// handler indefinitely.
var redditClient = &http.Client{Timeout: 10 * time.Second}

// redditListing mirrors the shape of Reddit's /new.json response (only the fields
// we use).
type redditListing struct {
	Data struct {
		Children []struct {
			Data struct {
				Title     string `json:"title"`
				Permalink string `json:"permalink"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

// redditPost is the trimmed post we return to the frontend.
type redditPost struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// ListRedditPosts fetches the newest posts from the logged-in user's configured
// subreddit and returns their titles and permalink URLs. Reddit is the app's only
// outbound HTTP dependency, so failures are mapped to clear status codes.
func ListRedditPosts(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		user, err := st.GetUser(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		// The subreddit is read server-side and escaped as defense in depth, even
		// though it is validated on save.
		endpoint := "https://www.reddit.com/r/" + url.PathEscape(user.Subreddit) + "/new.json?limit=25"
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not build request")
			return
		}
		// Reddit rate-limits or rejects requests with a generic/empty User-Agent.
		req.Header.Set("User-Agent", "nisaba/1.0 (writing prompt importer)")

		resp, err := redditClient.Do(req)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Could not reach Reddit")
			return
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			// fall through to decode
		case http.StatusNotFound:
			writeError(w, http.StatusNotFound, "Subreddit not found")
			return
		case http.StatusTooManyRequests:
			writeError(w, http.StatusTooManyRequests, "Reddit is rate-limiting requests; try again shortly")
			return
		default:
			writeError(w, http.StatusBadGateway, "Could not reach Reddit")
			return
		}

		var listing redditListing
		if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
			writeError(w, http.StatusBadGateway, "Unexpected response from Reddit")
			return
		}

		posts := make([]redditPost, 0, len(listing.Data.Children))
		for _, c := range listing.Data.Children {
			posts = append(posts, redditPost{
				Title: c.Data.Title,
				URL:   "https://www.reddit.com" + c.Data.Permalink,
			})
		}
		writeJSON(w, http.StatusOK, posts)
	}
}
