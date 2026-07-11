package reddit

import (
	"net/url"
	"strings"
)

// postPath validates a user-supplied Reddit post URL and returns the safe,
// traversal-free path to forward to oauth.reddit.com. ok is false for any
// non-Reddit host, non-permalink path, or path containing dot-segments (which
// would otherwise let "../" traverse to another oauth.reddit.com endpoint once
// the upstream resolves it — SSRF). The dot-segment check runs on the decoded
// path: EscapedPath() is always a consistent escaping of Path, so a decoded path
// free of "."/".." segments cannot produce traversal in either literal ("../")
// or percent-encoded ("%2e%2e") form.
func postPath(raw string) (path string, ok bool) {
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
	// Accept post permalinks (…/comments/<id>/…), which return the 2-element
	// array the caller decodes, and share links (…/s/<id>), which redirect to a
	// permalink and are resolved before the API call. Reject subreddit/user/home
	// URLs up front.
	if !strings.Contains(parsed.Path, "/comments/") && !isShareLink(parsed.Path) {
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
		strings.Contains(lowerRawPath, "%2e") || strings.Contains(lowerRawPath, "%2f") ||
		strings.Contains(lowerRawPath, "%252e") || strings.Contains(lowerRawPath, "%252f") {
		return "", false
	}

	// Use the escaped path so reserved characters survive the round-trip.
	return strings.TrimRight(parsed.EscapedPath(), "/"), true
}

// isShareLink reports whether path is a Reddit share link of the form
// /r/<sub>/s/<id> (or /u/... , /user/...), which Reddit issues from the mobile
// "share" action and redirects to the canonical comments permalink.
func isShareLink(path string) bool {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	return len(segs) == 4 && segs[2] == "s"
}
