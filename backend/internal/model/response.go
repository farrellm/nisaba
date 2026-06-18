package model

// Response is a generated answer attached to a Block, recording both the text
// and the model that produced it. Position orders responses within a block.
type Response struct {
	ID       int64  `json:"id"`
	BlockID  int64  `json:"blockId"`
	Value    string `json:"value"`
	Model    string `json:"model"`
	Position int    `json:"position"`
}
