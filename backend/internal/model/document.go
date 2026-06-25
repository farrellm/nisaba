package model

import "time"

// Document belongs to a User and aggregates its blocks, string attributes, and
// label names. Attributes keys are unique per document. The flat database tables
// are the source of truth; the embedded slices/maps are populated for API responses.
// PostURLs are the permalinks of posts published from this document (e.g. Reddit).
type Document struct {
	ID            int64             `json:"id"`
	UserID        int64             `json:"userId"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	Title         string            `json:"title"`
	SelectedModel string            `json:"selectedModel"`
	Metadata      map[string]any    `json:"metadata"`
	IsArchived    bool              `json:"isArchived"`
	URL           *string           `json:"url"`
	Attributes    map[string]string `json:"attributes"`
	Blocks        []Block           `json:"blocks"`
	Labels        []string          `json:"labels"`
	PostURLs      []string          `json:"postUrls"`
}
