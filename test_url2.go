package main

import (
	"fmt"
	"net/url"
	"path"
)

func main() {
	parsed, _ := url.Parse("https://reddit.com/r/subreddit/comments/id/foo/../../../../../../api/v1/me")
	fmt.Println(parsed.Path)
	fmt.Println(parsed.EscapedPath())
	fmt.Println(path.Clean(parsed.Path))
}
