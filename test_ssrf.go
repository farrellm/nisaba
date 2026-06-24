package main

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

func main() {
	raw := "https://reddit.com/comments/../../../../api/v1/me"
	parsed, _ := url.Parse(raw)

	cleanedPath := path.Clean(parsed.Path)
	parsed.Path = cleanedPath

	fmt.Println("Contains:", strings.Contains(cleanedPath, "/comments/"))
	pathStr := strings.TrimRight(parsed.EscapedPath(), "/")
	fmt.Println("PathStr:", pathStr)
}
