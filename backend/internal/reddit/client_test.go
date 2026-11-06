package reddit

import (
	"encoding/json"
	"testing"
)

func TestPostsFrom(t *testing.T) {
	raw := `{"data":{"after":"t3_x","children":[
		{"kind":"t3","data":{"title":"A post","permalink":"/r/WritingPrompts/comments/a/a_post/","author":"u1","subreddit":"WritingPrompts"}},
		{"kind":"t1","data":{"permalink":"/r/other/comments/b/x/c/","author":"u2","subreddit":"other"}},
		{"kind":"t3","data":{"title":"B","permalink":"/r/Fantasy/comments/c/b/","author":"u3","subreddit":"Fantasy"}}
	]}}`
	var l listing
	if err := json.Unmarshal([]byte(raw), &l); err != nil {
		t.Fatal(err)
	}
	if l.Data.After != "t3_x" {
		t.Errorf("after = %q", l.Data.After)
	}
	got := postsFrom(l)
	want := []Post{
		{Title: "A post", URL: "https://www.reddit.com/r/WritingPrompts/comments/a/a_post/", Author: "u1", Subreddit: "WritingPrompts"},
		{Title: "B", URL: "https://www.reddit.com/r/Fantasy/comments/c/b/", Author: "u3", Subreddit: "Fantasy"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d posts, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("post %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
