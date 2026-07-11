// Package reddit is a minimal Reddit OAuth client: application-only
// (client_credentials) tokens for reading and a script-app password grant for
// submitting self posts. Reddit blocks anonymous JSON access and requires a
// descriptive User-Agent on every request, including the token exchange.
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
	// ErrAuth means the OAuth token exchange failed.
	ErrAuth = errors.New("reddit: could not authenticate")
	// ErrUnreachable means the HTTP request to Reddit itself failed.
	ErrUnreachable = errors.New("reddit: could not reach reddit")
	// ErrNotFound means Reddit answered 404 (unknown subreddit or post).
	ErrNotFound = errors.New("reddit: not found")
	// ErrRateLimited means Reddit answered 429.
	ErrRateLimited = errors.New("reddit: rate limited")
	// ErrRejected means Reddit answered 401; the cached app token has been
	// invalidated so the next call fetches a fresh one.
	ErrRejected = errors.New("reddit: request rejected")
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
// use). The OAuth endpoints return the same structure as the old .json ones.
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

// Client calls Reddit's OAuth API. It caches the application-only token
// (shared across requests, refreshed shortly before expiry) and is safe for
// concurrent use. username/password are optional script-app account
// credentials, needed only to submit posts.
type Client struct {
	clientID     string
	clientSecret string
	username     string
	password     string

	// http has an explicit timeout (unlike http.DefaultClient) so a slow
	// upstream can't hang a handler indefinitely. noRedirect additionally
	// refuses to follow redirects, to peek at a share link's Location and
	// re-validate the target rather than chasing it blindly.
	http       *http.Client
	noRedirect *http.Client

	mu      sync.Mutex
	token   string
	expires time.Time
}

// NewClient builds a Client. username/password are optional and only used to
// submit posts.
func NewClient(clientID, clientSecret, username, password string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		username:     username,
		password:     password,
		http:         &http.Client{Timeout: 10 * time.Second},
		noRedirect: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Configured reports whether Reddit app credentials were supplied.
func (c *Client) Configured() bool {
	return c.clientID != "" && c.clientSecret != ""
}

// CanSubmit reports whether the script-app account credentials needed to
// submit a post (in addition to the app credentials) were supplied.
func (c *Client) CanSubmit() bool {
	return c.Configured() && c.username != "" && c.password != ""
}

// invalidate forces the next accessToken call to fetch a fresh token (e.g.
// after Reddit rejects the current one with 401).
func (c *Client) invalidate() {
	c.mu.Lock()
	c.token = ""
	c.mu.Unlock()
}

// tokenRequest performs one OAuth token exchange with the given form.
func (c *Client) tokenRequest(ctx context.Context, form url.Values) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.reddit.com/api/v1/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, err
	}
	req.SetBasicAuth(c.clientID, c.clientSecret)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("reddit token request returned %d", resp.StatusCode)
	}

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", 0, err
	}
	if body.AccessToken == "" {
		return "", 0, fmt.Errorf("reddit token response missing access_token")
	}
	return body.AccessToken, body.ExpiresIn, nil
}

// accessToken returns the cached application-only OAuth token, fetching a
// fresh one (client_credentials grant) when absent or near expiry.
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}

	token, expiresIn, err := c.tokenRequest(ctx, url.Values{"grant_type": {"client_credentials"}})
	if err != nil {
		return "", err
	}
	c.token = token
	// Refresh a minute before expiry to avoid racing the boundary.
	c.expires = time.Now().Add(time.Duration(expiresIn-60) * time.Second)
	return c.token, nil
}

// userAccessToken fetches a fresh user-context OAuth token via the password
// grant, authenticating as the configured script-app account. Unlike the
// application-only token it is not cached: submissions are infrequent, and a
// per-submit fetch avoids tangling with the cached app token.
func (c *Client) userAccessToken(ctx context.Context) (string, error) {
	token, _, err := c.tokenRequest(ctx, url.Values{
		"grant_type": {"password"},
		"username":   {c.username},
		"password":   {c.password},
	})
	return token, err
}

// get performs an authenticated GET against oauth.reddit.com and maps the
// non-200 statuses onto the package's sentinel errors. The caller decodes the
// body and must close it.
func (c *Client) get(ctx context.Context, endpoint string) (*http.Response, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuth, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return resp, nil
	case http.StatusNotFound:
		resp.Body.Close()
		return nil, ErrNotFound
	case http.StatusTooManyRequests:
		resp.Body.Close()
		return nil, ErrRateLimited
	case http.StatusUnauthorized:
		resp.Body.Close()
		// Cached token may be stale; drop it so the next attempt refreshes.
		c.invalidate()
		return nil, ErrRejected
	default:
		code := resp.StatusCode
		resp.Body.Close()
		return nil, &StatusError{Code: code}
	}
}

// NewestPosts returns the newest posts of a subreddit (Reddit's /new listing,
// capped at 25) with permalinks expanded to full URLs.
func (c *Client) NewestPosts(ctx context.Context, subreddit string) ([]Post, error) {
	// The subreddit is escaped as defense in depth, even though callers
	// validate it on save.
	endpoint := "https://oauth.reddit.com/r/" + url.PathEscape(subreddit) + "/new?limit=25"
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

	resp, err := c.get(ctx, "https://oauth.reddit.com"+path+"?raw_json=1&limit=1")
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
	req.Header.Set("User-Agent", userAgent)
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

// SubmitSelfPost publishes a self (text) post to subreddit as the configured
// script-app account (the password grant) and returns the new post's URL.
// Validation failures Reddit reports on a 200 (banned, rate limit, empty
// title, …) surface as a *SubmitError.
func (c *Client) SubmitSelfPost(ctx context.Context, subreddit, title, body string) (postURL string, err error) {
	token, err := c.userAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrAuth, err)
	}

	form := url.Values{
		"sr":       {subreddit},
		"kind":     {"self"},
		"title":    {title},
		"text":     {body},
		"api_type": {"json"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth.reddit.com/api/submit", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through to decode; Reddit reports logical errors in the body.
	case http.StatusTooManyRequests:
		return "", ErrRateLimited
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
