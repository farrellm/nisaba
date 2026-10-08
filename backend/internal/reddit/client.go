// Package reddit is a minimal Reddit client that reads and submits self posts
// through old.reddit.com as a logged-in user (a reddit_session cookie) rather
// than the OAuth Data API. Reddit blocks anonymous JSON access and expects a
// descriptive User-Agent on every request.
package reddit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// userAgent identifies this app to Reddit.
const userAgent = "nisaba/1.0 (writing prompt importer)"

// Sentinel errors callers map onto their own responses.
var (
	// ErrAuth means Reddit refused the session (missing, expired or logged
	// out).
	ErrAuth = errors.New("reddit: session rejected")
	// ErrUnreachable means the HTTP request to Reddit itself failed.
	ErrUnreachable = errors.New("reddit: could not reach reddit")
	// ErrNotFound means Reddit answered 404 (unknown subreddit or post).
	ErrNotFound = errors.New("reddit: not found")
	// ErrRateLimited means Reddit answered 429.
	ErrRateLimited = errors.New("reddit: rate limited")
	// ErrBadResponse means Reddit's response body could not be decoded.
	ErrBadResponse = errors.New("reddit: unexpected response")
	// ErrInvalidURL means the supplied post URL is not a Reddit permalink.
	ErrInvalidURL = errors.New("reddit: not a reddit post url")
	// ErrShareResolve means a share link (…/s/<id>) could not be expanded to
	// its canonical permalink.
	ErrShareResolve = errors.New("reddit: could not resolve share link")
)

// StatusError reports an unexpected HTTP status from Reddit.
type StatusError struct {
	Code int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("reddit: unexpected status %d", e.Code)
}

// SubmitError carries a validation failure Reddit reported for a submitted
// post (banned, rate limit, empty title, …).
type SubmitError struct {
	Message string
}

func (e *SubmitError) Error() string {
	return "reddit: submit rejected: " + e.Message
}

// Post is a trimmed Reddit post.
type Post struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Author string `json:"author"`
}

// listing mirrors the shape of Reddit's listing responses (only the fields we
// use), as served by old.reddit.com's .json endpoints.
type listing struct {
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

// Client reads and posts through old.reddit.com as a logged-in user, by
// sending that user's reddit_session cookie (the same JSON the old.reddit web
// UI serves). It caches subreddit listings briefly to keep request volume low,
// and is safe for concurrent use.
type Client struct {
	session string

	// noRedirect has an explicit timeout (unlike http.DefaultClient) so a
	// slow upstream can't hang a handler indefinitely, and refuses to follow
	// redirects: to peek at a share link's Location and re-validate the target
	// rather than chasing it blindly, and to notice Reddit bouncing a stale
	// session to its login page.
	noRedirect *http.Client

	mu       sync.Mutex
	listings map[string]cachedListing
}

// cachedListing is one subreddit's newest posts and when they stop being fresh.
type cachedListing struct {
	posts   []Post
	expires time.Time
}

// listingTTL is how long NewestPosts reuses a subreddit's listing.
const listingTTL = 2 * time.Minute

// NewClient builds a Client authenticated by session, the value of a
// logged-in old.reddit.com reddit_session cookie.
func NewClient(session string) *Client {
	return &Client{
		session:  session,
		listings: map[string]cachedListing{},
		noRedirect: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Configured reports whether a Reddit session was supplied.
func (c *Client) Configured() bool {
	return c.session != ""
}

// authorize adds the session cookie and User-Agent to req.
func (c *Client) authorize(req *http.Request) {
	req.AddCookie(&http.Cookie{Name: "reddit_session", Value: c.session})
	req.Header.Set("User-Agent", userAgent)
}

// get performs a session-authenticated GET against old.reddit.com and maps the
// non-200 statuses onto the package's sentinel errors. The caller decodes the
// body and must close it.
func (c *Client) get(ctx context.Context, endpoint string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)

	resp, err := c.noRedirect.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}

	code := resp.StatusCode
	switch {
	case code == http.StatusOK:
		return resp, nil
	case code >= 300 && code < 400:
		// A stale session is bounced to the login page; old.reddit redirects
		// an unknown subreddit to its search page.
		loc := resp.Header.Get("Location")
		resp.Body.Close()
		if strings.Contains(loc, "/login") {
			return nil, ErrAuth
		}
		return nil, ErrNotFound
	}
	resp.Body.Close()
	switch code {
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case http.StatusUnauthorized, http.StatusForbidden:
		// Reddit blocks requests without a valid session outright.
		return nil, ErrAuth
	default:
		return nil, &StatusError{Code: code}
	}
}

// NewestPosts returns the newest posts of a subreddit (Reddit's /new listing,
// capped at 25) with permalinks expanded to full URLs. A subreddit's listing is
// reused for listingTTL.
func (c *Client) NewestPosts(ctx context.Context, subreddit string) ([]Post, error) {
	key := strings.ToLower(subreddit)
	c.mu.Lock()
	cached, ok := c.listings[key]
	c.mu.Unlock()
	if ok && time.Now().Before(cached.expires) {
		return cached.posts, nil
	}

	// The subreddit is escaped as defense in depth, even though callers
	// validate it on save.
	endpoint := "https://old.reddit.com/r/" + url.PathEscape(subreddit) + "/new.json?limit=25"
	resp, err := c.get(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var l listing
	if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadResponse, err)
	}

	posts := make([]Post, 0, len(l.Data.Children))
	for _, child := range l.Data.Children {
		posts = append(posts, Post{
			Title:  child.Data.Title,
			URL:    "https://www.reddit.com" + child.Data.Permalink,
			Author: child.Data.Author,
		})
	}

	c.mu.Lock()
	c.listings[key] = cachedListing{posts: posts, expires: time.Now().Add(listingTTL)}
	c.mu.Unlock()
	return posts, nil
}

// FetchPost fetches a single post by user-supplied URL, validating it against
// the SSRF guards in postPath and expanding share links (…/s/<id>) to their
// canonical permalinks first. The returned Post carries the normalized
// permalink URL.
func (c *Client) FetchPost(ctx context.Context, rawURL string) (Post, error) {
	path, ok := postPath(rawURL)
	if !ok {
		return Post{}, ErrInvalidURL
	}
	path, ok = c.resolveSharePath(ctx, path)
	if !ok {
		return Post{}, ErrShareResolve
	}

	resp, err := c.get(ctx, "https://old.reddit.com"+path+".json?raw_json=1&limit=1")
	if err != nil {
		return Post{}, err
	}
	defer resp.Body.Close()

	// The comments endpoint returns a 2-element array: [post listing, comments].
	var listings []listing
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		return Post{}, fmt.Errorf("%w: %w", ErrBadResponse, err)
	}
	if len(listings) == 0 || len(listings[0].Data.Children) == 0 {
		return Post{}, ErrNotFound
	}

	data := listings[0].Data.Children[0].Data
	return Post{
		Title:  data.Title,
		URL:    "https://www.reddit.com" + data.Permalink,
		Author: data.Author,
	}, nil
}

// resolveSharePath expands a Reddit share link (…/s/<id>) to its canonical
// comments permalink path by reading the redirect Reddit issues for it. Paths
// that are already permalinks are returned unchanged. The redirect target is
// re-validated through postPath so the SSRF guards apply to it too.
func (c *Client) resolveSharePath(ctx context.Context, path string) (string, bool) {
	if !isShareLink(path) {
		return path, true
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.reddit.com"+path, nil)
	if err != nil {
		return "", false
	}
	c.authorize(req)
	resp, err := c.noRedirect.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", false
	}
	resolved, ok := postPath(loc)
	if !ok || isShareLink(resolved) {
		return "", false
	}
	return resolved, true
}

// modhash fetches the session's modhash, the CSRF token old.reddit requires
// on state-changing requests. A session Reddit doesn't recognize yields an
// empty modhash, reported as ErrAuth.
func (c *Client) modhash(ctx context.Context) (string, error) {
	resp, err := c.get(ctx, "https://old.reddit.com/api/me.json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var me struct {
		Data struct {
			Modhash string `json:"modhash"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return "", fmt.Errorf("%w: %w", ErrBadResponse, err)
	}
	if me.Data.Modhash == "" {
		return "", ErrAuth
	}
	return me.Data.Modhash, nil
}

// SubmitSelfPost publishes a self (text) post to subreddit as the session's
// user, the way old.reddit's own submit form does, and returns the new post's
// URL. Validation failures Reddit reports on a 200 (banned, rate limit,
// captcha, empty title, …) surface as a *SubmitError.
func (c *Client) SubmitSelfPost(ctx context.Context, subreddit, title, body string) (postURL string, err error) {
	uh, err := c.modhash(ctx)
	if err != nil {
		return "", err
	}

	form := url.Values{
		"sr":       {subreddit},
		"kind":     {"self"},
		"title":    {title},
		"text":     {body},
		"api_type": {"json"},
		"uh":       {uh},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://old.reddit.com/api/submit", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	c.authorize(req)
	req.Header.Set("X-Modhash", uh)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.noRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through to decode; Reddit reports logical errors in the body.
	case http.StatusTooManyRequests:
		return "", ErrRateLimited
	case http.StatusUnauthorized, http.StatusForbidden:
		return "", ErrAuth
	default:
		return "", &StatusError{Code: resp.StatusCode}
	}

	// On a 200 Reddit still reports validation failures in json.errors as
	// [code, message, field] triples.
	var submitResp struct {
		JSON struct {
			Errors [][]string `json:"errors"`
			Data   struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"json"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		return "", fmt.Errorf("%w: %w", ErrBadResponse, err)
	}
	if len(submitResp.JSON.Errors) > 0 {
		msg := "Reddit rejected the post"
		if e := submitResp.JSON.Errors[0]; len(e) >= 2 && e[1] != "" {
			msg = e[1]
		}
		return "", &SubmitError{Message: msg}
	}
	return submitResp.JSON.Data.URL, nil
}
