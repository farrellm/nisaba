package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func main() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Received request for:", r.URL.Path)
	}))
	defer ts.Close()

	endpoint := ts.URL + "/comments/../../../../api/v1/me"
	req, _ := http.NewRequest("GET", endpoint, nil)
	client := &http.Client{}
	client.Do(req)
}
