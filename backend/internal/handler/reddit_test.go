package handler

import "testing"

func TestRedditPostPath(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantPath string
		wantOK   bool
	}{
		{
			name:     "full permalink",
			in:       "https://www.reddit.com/r/WritingPrompts/comments/abc123/a_title/",
			wantPath: "/r/WritingPrompts/comments/abc123/a_title",
			wantOK:   true,
		},
		{
			name:     "no scheme",
			in:       "www.reddit.com/r/WritingPrompts/comments/abc123/a_title",
			wantPath: "/r/WritingPrompts/comments/abc123/a_title",
			wantOK:   true,
		},
		{
			name:     "bare reddit.com host",
			in:       "https://reddit.com/r/WritingPrompts/comments/abc123/a_title",
			wantPath: "/r/WritingPrompts/comments/abc123/a_title",
			wantOK:   true,
		},
		{
			name:     "share link",
			in:       "https://www.reddit.com/r/WritingPrompts/s/Fjx2tnO4PR",
			wantPath: "/r/WritingPrompts/s/Fjx2tnO4PR",
			wantOK:   true,
		},
		{
			name:   "empty",
			in:     "   ",
			wantOK: false,
		},
		{
			name:   "non-reddit host",
			in:     "https://evil.com/r/x/comments/abc123/t",
			wantOK: false,
		},
		{
			name:   "host suffix spoof",
			in:     "https://reddit.com.evil.com/r/x/comments/abc123/t",
			wantOK: false,
		},
		{
			name:   "not a permalink",
			in:     "https://www.reddit.com/r/WritingPrompts/",
			wantOK: false,
		},
		{
			name:   "literal traversal",
			in:     "https://reddit.com/r/x/comments/abc123/t/../../../../../../api/v1/me",
			wantOK: false,
		},
		{
			name:   "encoded traversal",
			in:     "https://reddit.com/r/x/comments/abc123/t/%2e%2e/%2e%2e/api/v1/me",
			wantOK: false,
		},
		{
			name:   "single dot segment",
			in:     "https://reddit.com/r/x/comments/abc123/./t",
			wantOK: false,
		},
		{
			name:   "double encoded traversal",
			in:     "https://reddit.com/r/x/comments/abc123/t/%252e%252e/%252e%252e/api/v1/me",
			wantOK: false,
		},
		{
			name:   "encoded slash",
			in:     "https://reddit.com/r/x/comments/abc123/t/..%2f..%2fapi/v1/me",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotOK := redditPostPath(tt.in)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotOK && gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}
