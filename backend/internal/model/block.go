package model

// Block belongs to a Document and holds zero or more responses plus string
// attributes whose keys are unique per block. Position orders blocks within a document.
type Block struct {
	ID         int64             `json:"id"`
	DocumentID int64             `json:"documentId"`
	Mode       string            `json:"mode"`
	Position   int               `json:"position"`
	Attributes map[string]string `json:"attributes"`
	Responses  []Response        `json:"responses"`
}
