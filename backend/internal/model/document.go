package model

import "time"

// Document belongs to a User and aggregates its blocks, string attributes, and
// label names. Attributes keys are unique per document. The flat database tables
// are the source of truth; the embedded slices/maps are populated for API responses.
type Document struct {
	ID            int64             `json:"id"`
	UserID        int64             `json:"userId"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	SelectedModel string            `json:"selectedModel"`
	Metadata      map[string]any    `json:"metadata"`
	IsArchived    bool              `json:"isArchived"`
	URL           *string           `json:"url"`
	Attributes    map[string]string `json:"attributes"`
	Blocks        []Block           `json:"blocks"`
	Labels        []string          `json:"labels"`
}
