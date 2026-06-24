package main

import (
	"fmt"
	"net/url"
)

func main() {
	parsed, _ := url.Parse("https://reddit.com/comments/../../../../api/v1/me")
	fmt.Println(parsed.Path)
	fmt.Println(parsed.EscapedPath())
}
