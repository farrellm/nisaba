package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	// username/password authenticate a script-app account for submitting posts
	// (the password grant). They are optional: reading uses the app-only token
	// above, submitting needs a user identity.
	username string
	password string

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
				Author    string `json:"author"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

// redditPost is the trimmed post we return to the frontend.
type redditPost struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Author string `json:"author"`
}

// NewRedditAuth creates a shared OAuth token holder for the Reddit handlers, so
// the listing and single-post endpoints reuse one cached application-only token.
// username/password are optional and only used to submit posts.
func NewRedditAuth(clientID, clientSecret, username, password string) *redditAuth {
	return &redditAuth{
		clientID:     clientID,
		clientSecret: clientSecret,
		username:     username,
		password:     password,
	}
}

// configured reports whether Reddit credentials were supplied.
func (a *redditAuth) configured() bool {
	return a.clientID != "" && a.clientSecret != ""
}

// canSubmit reports whether the script-app account credentials needed to submit
// a post (in addition to the app credentials) were supplied.
func (a *redditAuth) canSubmit() bool {
	return a.configured() && a.username != "" && a.password != ""
}

// userAccessToken fetches a fresh user-context OAuth token via the password
// grant, authenticating as the configured script-app account. Unlike the
// application-only token it is not cached: submissions are infrequent, and a
// per-submit fetch avoids tangling with the cached app token (a.token).
func (a *redditAuth) userAccessToken(ctx context.Context) (string, error) {
	form := url.Values{
		"grant_type": {"password"},
		"username":   {a.username},
		"password":   {a.password},
	}
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
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("reddit token response missing access_token")
	}
	return body.AccessToken, nil
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
				Title:  c.Data.Title,
				URL:    "https://www.reddit.com" + c.Data.Permalink,
				Author: c.Data.Author,
			})
		}
		writeJSON(w, http.StatusOK, posts)
	}
}

// redditPostPath validates a user-supplied Reddit post URL and returns the safe,
// traversal-free path to forward to oauth.reddit.com. ok is false for any
// non-Reddit host, non-permalink path, or path containing dot-segments (which
// would otherwise let "../" traverse to another oauth.reddit.com endpoint once
// the upstream resolves it — SSRF). The dot-segment check runs on the decoded
// path: EscapedPath() is always a consistent escaping of Path, so a decoded path
// free of "."/".." segments cannot produce traversal in either literal ("../")
// or percent-encoded ("%2e%2e") form.
func redditPostPath(raw string) (path string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	// Tolerate URLs pasted without a scheme (e.g. "www.reddit.com/r/…"),
	// which url.Parse would otherwise treat as a path with no host.
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", false
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "reddit.com" && !strings.HasSuffix(host, ".reddit.com") {
		return "", false
	}
	// Only a post permalink (…/comments/<id>/…) returns the 2-element array the
	// caller decodes. Reject subreddit/user/home URLs up front.
	if !strings.Contains(parsed.Path, "/comments/") {
		return "", false
	}
	for _, seg := range strings.Split(parsed.Path, "/") {
		if seg == "." || seg == ".." {
			return "", false
		}
	}

	// Reject paths with encoded dots or slashes which might bypass traversal checks
	// depending on how the upstream server normalizes the path.
	// Check both Path (for double-encoding resolved to single-encoding) and
	// RawPath (for single-encoding preserved raw).
	lowerPath := strings.ToLower(parsed.Path)
	lowerRawPath := strings.ToLower(parsed.RawPath)
	if strings.Contains(lowerPath, "%2e") || strings.Contains(lowerPath, "%2f") ||
		strings.Contains(lowerRawPath, "%2e") || strings.Contains(lowerRawPath, "%2f") {
		return "", false
	}

	// Use the escaped path so reserved characters survive the round-trip.
	return strings.TrimRight(parsed.EscapedPath(), "/"), true
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

		raw := r.URL.Query().Get("url")
		if strings.TrimSpace(raw) == "" {
			writeError(w, http.StatusBadRequest, "Missing url")
			return
		}
		path, ok := redditPostPath(raw)
		if !ok {
			writeError(w, http.StatusBadRequest, "Not a Reddit URL")
			return
		}

		token, err := ra.accessToken(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "Could not authenticate with Reddit")
			return
		}

		endpoint := "https://oauth.reddit.com" + path + "?raw_json=1&limit=1"
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
			Title:  data.Title,
			URL:    "https://www.reddit.com" + data.Permalink,
			Author: data.Author,
		})
	}
}

// SubmitRedditPost publishes a self (text) post to the document owner's
// configured subreddit as the script-app account (the password grant) and
// records the resulting permalink on the document. Reading uses an
// application-only token, which has no account identity and cannot submit, so
// this needs REDDIT_USERNAME/PASSWORD in addition to the app credentials and
// reports 503 when they're absent. It is nested under /documents/{id} so the
// returned post URL is saved against that document.
func SubmitRedditPost(st *store.Store, sess *auth.Sessions, ra *redditAuth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
		if !ok {
			return
		}
		if !ra.canSubmit() {
			writeError(w, http.StatusServiceUnavailable, "Reddit posting is not configured")
			return
		}

		var body struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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
			writeError(w, http.StatusInternalServerError, "Could not load user")
			return
		}

		token, err := ra.userAccessToken(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, "Could not authenticate with Reddit")
			return
		}

		form := url.Values{
			"sr":       {user.Subreddit},
			"kind":     {"self"},
			"title":    {title},
			"text":     {body.Body},
			"api_type": {"json"},
		}
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
			"https://oauth.reddit.com/api/submit", strings.NewReader(form.Encode()))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not build request")
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", redditUserAgent)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := redditClient.Do(req)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Could not reach Reddit")
			return
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			// fall through to decode; Reddit reports logical errors in the body.
		case http.StatusTooManyRequests:
			writeError(w, http.StatusTooManyRequests, "Reddit is rate-limiting requests; try again shortly")
			return
		default:
			writeError(w, http.StatusBadGateway, fmt.Sprintf("Reddit returned an unexpected status (%d)", resp.StatusCode))
			return
		}

		// On a 200 Reddit still reports validation failures (banned, rate limit,
		// empty title, …) in json.errors as [code, message, field] triples.
		var submitResp struct {
			JSON struct {
				Errors [][]string `json:"errors"`
				Data   struct {
					URL string `json:"url"`
				} `json:"data"`
			} `json:"json"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
			writeError(w, http.StatusBadGateway, "Unexpected response from Reddit")
			return
		}
		if len(submitResp.JSON.Errors) > 0 {
			msg := "Reddit rejected the post"
			if e := submitResp.JSON.Errors[0]; len(e) >= 2 && e[1] != "" {
				msg = e[1]
			}
			writeError(w, http.StatusBadRequest, msg)
			return
		}

		if url := submitResp.JSON.Data.URL; url != "" {
			if err := st.AddDocumentPost(r.Context(), doc.ID, url); err != nil {
				writeError(w, http.StatusInternalServerError, "Posted, but could not save the post URL")
				return
			}
		}

		updated, err := st.GetDocument(r.Context(), doc.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load document")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}
