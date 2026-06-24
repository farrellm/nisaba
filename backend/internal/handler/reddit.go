package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/store"
)

// redditClient is the shared HTTP client for outbound Reddit requests. It has an
// explicit timeout (unlike http.DefaultClient) so a slow upstream can't hang the
// handler indefinitely.
var redditClient = &http.Client{Timeout: 10 * time.Second}

// redditUserAgent identifies this app to Reddit. Reddit requires a descriptive
// User-Agent on every request, including the OAuth token exchange.
const redditUserAgent = "nisaba/1.0 (writing prompt importer)"

// redditAuth fetches and caches an application-only OAuth token (the
// client_credentials grant), which Reddit requires now that anonymous JSON
// access is blocked. The token is shared across requests and refreshed shortly
// before it expires.
type redditAuth struct {
	clientID     string
	clientSecret string

	mu      sync.Mutex
	token   string
	expires time.Time
}

// invalidate forces the next accessToken call to fetch a fresh token (e.g. after
// Reddit rejects the current one with 401).
func (a *redditAuth) invalidate() {
	a.mu.Lock()
	a.token = ""
	a.mu.Unlock()
}

func (a *redditAuth) accessToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.token != "" && time.Now().Before(a.expires) {
		return a.token, nil
	}

	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.reddit.com/api/v1/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(a.clientID, a.clientSecret)
	req.Header.Set("User-Agent", redditUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := redditClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reddit token request returned %d", resp.StatusCode)
	}

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("reddit token response missing access_token")
	}

	a.token = body.AccessToken
	// Refresh a minute before expiry to avoid racing the boundary.
	a.expires = time.Now().Add(time.Duration(body.ExpiresIn-60) * time.Second)
	return a.token, nil
}

// redditListing mirrors the shape of Reddit's /new listing response (only the
// fields we use). The OAuth endpoint returns the same structure as the old
// /new.json endpoint.
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

// NewRedditAuth creates a shared OAuth token holder for the Reddit handlers, so
// the listing and single-post endpoints reuse one cached application-only token.
func NewRedditAuth(clientID, clientSecret string) *redditAuth {
	return &redditAuth{clientID: clientID, clientSecret: clientSecret}
}

// configured reports whether Reddit credentials were supplied.
func (a *redditAuth) configured() bool {
	return a.clientID != "" && a.clientSecret != ""
}

// ListRedditPosts fetches the newest posts from the logged-in user's configured
// subreddit via Reddit's application-only OAuth API and returns their titles and
// permalink URLs. Anonymous access is blocked by Reddit, so clientID/secret from
// a registered app are required.
func ListRedditPosts(st *store.Store, sess *auth.Sessions, ra *redditAuth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		if !ra.configured() {
			writeError(w, http.StatusServiceUnavailable, "Reddit integration is not configured")
			return
		}

		user, err := st.GetUser(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		token, err := ra.accessToken(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "Could not authenticate with Reddit")
			return
		}

		// The subreddit is read server-side and escaped as defense in depth, even
		// though it is validated on save.
		endpoint := "https://oauth.reddit.com/r/" + url.PathEscape(user.Subreddit) + "/new?limit=25"
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not build request")
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", redditUserAgent)

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
		case http.StatusUnauthorized:
			// Cached token may be stale; drop it so the next attempt refreshes.
			ra.invalidate()
			writeError(w, http.StatusBadGateway, "Reddit rejected the request; try again")
			return
		default:
			writeError(w, http.StatusBadGateway, fmt.Sprintf("Reddit returned an unexpected status (%d)", resp.StatusCode))
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

// GetRedditPost fetches a single Reddit post by URL via Reddit's application-only
// OAuth API and returns its title and normalized permalink URL. It lets users
// import a specific post that isn't in the subreddit's newest listing.
func GetRedditPost(sess *auth.Sessions, ra *redditAuth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := sess.UserID(r); !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		if !ra.configured() {
			writeError(w, http.StatusServiceUnavailable, "Reddit integration is not configured")
			return
		}

		raw := strings.TrimSpace(r.URL.Query().Get("url"))
		if raw == "" {
			writeError(w, http.StatusBadRequest, "Missing url")
			return
		}
		// Tolerate URLs pasted without a scheme (e.g. "www.reddit.com/r/…"),
		// which url.Parse would otherwise treat as a path with no host.
		if !strings.Contains(raw, "://") {
			raw = "https://" + raw
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			writeError(w, http.StatusBadRequest, "Not a Reddit URL")
			return
		}
		host := strings.ToLower(parsed.Hostname())
		if host != "reddit.com" && !strings.HasSuffix(host, ".reddit.com") {
			writeError(w, http.StatusBadRequest, "Not a Reddit URL")
			return
		}

		importPath := path.Clean(parsed.Path)
		parsed.Path = importPath

		// Only a post permalink (…/comments/<id>/…) returns the 2-element array we
		// decode below. Reject subreddit/user/home URLs up front with a clear
		// message instead of letting the decode fail with a generic 502.
		if !strings.Contains(importPath, "/comments/") {
			writeError(w, http.StatusBadRequest, "Not a Reddit post URL")
			return
		}

		token, err := ra.accessToken(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "Could not authenticate with Reddit")
			return
		}

		// Use the escaped path so reserved characters survive the round-trip.
		cleanedPath := strings.TrimRight(parsed.EscapedPath(), "/")
		endpoint := "https://oauth.reddit.com" + cleanedPath + "?raw_json=1&limit=1"
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not build request")
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", redditUserAgent)

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
			writeError(w, http.StatusNotFound, "Post not found")
			return
		case http.StatusTooManyRequests:
			writeError(w, http.StatusTooManyRequests, "Reddit is rate-limiting requests; try again shortly")
			return
		case http.StatusUnauthorized:
			ra.invalidate()
			writeError(w, http.StatusBadGateway, "Reddit rejected the request; try again")
			return
		default:
			writeError(w, http.StatusBadGateway, fmt.Sprintf("Reddit returned an unexpected status (%d)", resp.StatusCode))
			return
		}

		// The comments endpoint returns a 2-element array: [post listing, comments].
		var listings []redditListing
		if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
			writeError(w, http.StatusBadGateway, "Unexpected response from Reddit")
			return
		}
		if len(listings) == 0 || len(listings[0].Data.Children) == 0 {
			writeError(w, http.StatusNotFound, "Post not found")
			return
		}

		data := listings[0].Data.Children[0].Data
		writeJSON(w, http.StatusOK, redditPost{
			Title: data.Title,
			URL:   "https://www.reddit.com" + data.Permalink,
		})
	}
}
