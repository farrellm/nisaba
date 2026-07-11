package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/blockrun"
	"github.com/farrellm/nisaba/internal/mode"
	"github.com/farrellm/nisaba/internal/model"
)

// documentGetter is the smallest store view the ownership check needs; the
// per-file store interfaces embed it. *store.Store satisfies all of them.
type documentGetter interface {
	GetDocument(ctx context.Context, id int64) (model.Document, error)
}

// BlockStore is the consumer-side view of the data layer the block handlers use.
type BlockStore interface {
	documentGetter
	CreateBlock(ctx context.Context, b model.Block) (model.Block, error)
	GetBlock(ctx context.Context, id int64) (model.Block, error)
	ReplaceBlockAttributes(ctx context.Context, blockID int64, attrs map[string]string) error
	MergeDocumentAttributes(ctx context.Context, documentID int64, attrs map[string]string) error
	DeleteBlock(ctx context.Context, id int64) error
	UpdateResponse(ctx context.Context, r model.Response) error
}

// ownedDocument loads the document named by the {id} URL param and confirms the
// logged-in user owns it. On any failure it writes the appropriate response and
// returns ok=false; resources owned by another user surface as 404 so their
// existence isn't leaked.
func ownedDocument(w http.ResponseWriter, r *http.Request, st documentGetter) (model.Document, bool) {
	userID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Not logged in")
		return model.Document{}, false
	}
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid document id")
		return model.Document{}, false
	}
	doc, err := st.GetDocument(r.Context(), id)
	if err != nil {
		notFoundOr500(w, r, err, "Document not found", "Could not load document")
		return model.Document{}, false
	}
	if doc.UserID != userID {
		writeError(w, http.StatusNotFound, "Document not found")
		return model.Document{}, false
	}
	return doc, true
}

// findBlock returns the block with the {blockId} URL param from an already-owned
// document. Missing or non-matching ids yield 404.
func findBlock(w http.ResponseWriter, r *http.Request, doc model.Document) (model.Block, bool) {
	blockID, err := pathID(r, "blockId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid block id")
		return model.Block{}, false
	}
	for _, b := range doc.Blocks {
		if b.ID == blockID {
			return b, true
		}
	}
	writeError(w, http.StatusNotFound, "Block not found")
	return model.Block{}, false
}

type newBlock struct {
	Mode string `json:"mode"`
}

// CreateBlock appends a block to a document. The new block's attributes are
// seeded from the document's attributes for the chosen mode's keys (empty
// string where the document has no value).
func CreateBlock(st BlockStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st)
		if !ok {
			return
		}

		var body newBlock
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		m, ok := mode.Get(body.Mode)
		if !ok {
			writeError(w, http.StatusBadRequest, "Unknown mode")
			return
		}

		block, err := st.CreateBlock(r.Context(), model.Block{
			DocumentID: doc.ID,
			Mode:       m.Name,
			Position:   len(doc.Blocks),
		})
		if err != nil {
			internalError(w, r, "Could not create block", err)
			return
		}

		attrs := make(map[string]string, len(m.Keys))
		for _, key := range m.Keys {
			attrs[key] = doc.Attributes[key] // "" when absent
		}
		if err := st.ReplaceBlockAttributes(r.Context(), block.ID, attrs); err != nil {
			internalError(w, r, "Could not create block", err)
			return
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			internalError(w, r, "Could not load block", err)
			return
		}
		writeJSON(w, http.StatusCreated, hydrated)
	}
}

type updateBlock struct {
	Attributes map[string]string `json:"attributes"`
}

// UpdateBlock replaces a block's key/values. Keys outside the mode's fixed key
// set are ignored, so the stored attributes always match the mode.
func UpdateBlock(st BlockStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st)
		if !ok {
			return
		}
		block, ok := findBlock(w, r, doc)
		if !ok {
			return
		}

		var body updateBlock
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		m, ok := mode.Get(block.Mode)
		if !ok {
			writeError(w, http.StatusInternalServerError, "Block has an unknown mode")
			return
		}
		attrs := blockrun.MergedAttrs(block, m, body.Attributes)
		if err := st.ReplaceBlockAttributes(r.Context(), block.ID, attrs); err != nil {
			internalError(w, r, "Could not update block", err)
			return
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			internalError(w, r, "Could not load block", err)
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}

// CopyBlock saves the block's key/values and promotes them into the document's
// shared attributes (merging, so values set by other blocks survive). It accepts
// the same body shape as UpdateBlock so the caller's on-screen edits are saved
// before they're copied up.
func CopyBlock(st BlockStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st)
		if !ok {
			return
		}
		block, ok := findBlock(w, r, doc)
		if !ok {
			return
		}

		var body updateBlock
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		m, ok := mode.Get(block.Mode)
		if !ok {
			writeError(w, http.StatusInternalServerError, "Block has an unknown mode")
			return
		}
		attrs := blockrun.MergedAttrs(block, m, body.Attributes)
		if err := st.ReplaceBlockAttributes(r.Context(), block.ID, attrs); err != nil {
			internalError(w, r, "Could not update block", err)
			return
		}
		if err := st.MergeDocumentAttributes(r.Context(), doc.ID, attrs); err != nil {
			internalError(w, r, "Could not update document", err)
			return
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			internalError(w, r, "Could not load block", err)
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}

// runBody decodes the optional attribute edits of a run request. An empty body
// is fine (the block's saved values are used); a malformed one is a 400.
func runBody(w http.ResponseWriter, r *http.Request, doc model.Document, block model.Block) (map[string]string, bool) {
	var body updateBlock
	if err := decodeJSON(r, &body); err != nil && err != io.EOF {
		slog.Error("run failed: decode request body",
			"err", err, "user", doc.UserID, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return nil, false
	}
	return body.Attributes, true
}

// writeRunError maps a blockrun error onto the JSON error response, preserving
// the per-step statuses and messages.
func writeRunError(w http.ResponseWriter, err error) {
	var runErr *blockrun.Error
	switch {
	case errors.Is(err, blockrun.ErrUnknownMode):
		writeError(w, http.StatusInternalServerError, "Block has an unknown mode")
	case errors.Is(err, blockrun.ErrNoModel):
		writeError(w, http.StatusBadRequest, "No model selected")
	case errors.As(err, &runErr):
		switch runErr.Step {
		case blockrun.StepUpdateBlock:
			writeError(w, http.StatusInternalServerError, "Could not update block")
		case blockrun.StepMergeDocument:
			writeError(w, http.StatusInternalServerError, "Could not update document")
		case blockrun.StepRenderPrompt:
			writeError(w, http.StatusInternalServerError, "Could not render prompt")
		case blockrun.StepRenderSystem:
			writeError(w, http.StatusInternalServerError, "Could not render system prompt")
		case blockrun.StepGenerate:
			writeError(w, http.StatusBadGateway, "Model request failed: "+runErr.Err.Error())
		default: // StepSave
			writeError(w, http.StatusInternalServerError, "Could not save response")
		}
	default:
		writeError(w, http.StatusInternalServerError, "Could not run block")
	}
}

// RunBlock saves the block's key/values, promotes them into the document's
// shared attributes, renders the mode's mustache template against them to build
// a prompt, runs it, saves the result as a response, and writes the result back
// into the document's attributes under the mode's output key. It accepts the
// same optional body as UpdateBlock so the caller's on-screen edits are saved
// before the run.
func RunBlock(st documentGetter, runner *blockrun.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st)
		if !ok {
			return
		}
		block, ok := findBlock(w, r, doc)
		if !ok {
			return
		}
		edits, ok := runBody(w, r, doc, block)
		if !ok {
			return
		}

		run, err := runner.Prepare(r.Context(), doc, block, edits)
		if err != nil {
			writeRunError(w, err)
			return
		}

		hydrated, err := runner.Execute(r.Context(), run)
		if err != nil {
			writeRunError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}

// RunBlockStream is the streaming variant of RunBlock: it runs the model with
// the streaming generator and pushes the reply to the client as it arrives,
// framed as newline-delimited JSON (NDJSON). Each line is one event:
//
//	{"type":"delta","text":"..."}   incremental text
//	{"type":"ping"}                  keepalive while the model runs (client ignores)
//	{"type":"error","message":"..."} terminal failure (after streaming began)
//	{"type":"done","block":{...}}    the fully hydrated block, like RunBlock's body
//
// Setup/validation failures before streaming begins still use the normal JSON
// error path; once the 200 NDJSON stream has started, errors can only be
// reported as an "error" event.
func RunBlockStream(st documentGetter, runner *blockrun.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st)
		if !ok {
			return
		}
		block, ok := findBlock(w, r, doc)
		if !ok {
			return
		}
		edits, ok := runBody(w, r, doc, block)
		if !ok {
			return
		}

		run, err := runner.Prepare(r.Context(), doc, block, edits)
		if err != nil {
			writeRunError(w, err)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, "Streaming unsupported")
			return
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		enc := json.NewEncoder(w)
		// writeEvent is called from both the generation goroutine (via the delta
		// callback) and the keepalive ticker goroutine, so serialize writes.
		var mu sync.Mutex
		writeEvent := func(v any) {
			// Errors here mean the client went away; nothing useful to do but
			// stop. The encoder appends the newline that delimits each event.
			mu.Lock()
			defer mu.Unlock()
			_ = enc.Encode(v)
			flusher.Flush()
		}

		// Keepalive: emit a ping every 10s while the model runs so an
		// intermediate proxy (e.g. the Vite dev proxy's 120s inactivity timeout)
		// doesn't drop a long, quiet generation. The client ignores ping events.
		stop := make(chan struct{})
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					writeEvent(map[string]string{"type": "ping"})
				case <-stop:
					return
				}
			}
		}()

		hydrated, err := runner.ExecuteStream(r.Context(), run, func(delta string) {
			writeEvent(map[string]string{"type": "delta", "text": delta})
		})
		// Stop the keepalive and wait for its goroutine to exit before writing the
		// terminal event, so no stray ping can land after done/error.
		close(stop)
		<-finished
		if err != nil {
			var runErr *blockrun.Error
			msg := "Could not save response"
			if errors.As(err, &runErr) && runErr.Step == blockrun.StepGenerate {
				msg = "Model request failed: " + runErr.Err.Error()
			}
			writeEvent(map[string]string{"type": "error", "message": msg})
			return
		}
		writeEvent(map[string]any{"type": "done", "block": hydrated})
	}
}

// ReparseResponse re-runs only the parse + merge step against an existing
// response named by {responseId}, without calling the model. It feeds that
// response's stored text back into the document's shared attributes the same way
// RunBlock does (top-level XML tags plus the mode's output key, merged), so a user
// can re-derive attributes from any past response.
func ReparseResponse(st BlockStore, runner *blockrun.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st)
		if !ok {
			return
		}
		block, ok := findBlock(w, r, doc)
		if !ok {
			return
		}

		m, ok := mode.Get(block.Mode)
		if !ok {
			writeError(w, http.StatusInternalServerError, "Block has an unknown mode")
			return
		}

		responseID, err := pathID(r, "responseId")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid response id")
			return
		}
		var output string
		found := false
		for _, resp := range block.Responses {
			if resp.ID == responseID {
				output = resp.Value
				found = true
				break
			}
		}
		if !found {
			writeError(w, http.StatusNotFound, "Response not found")
			return
		}

		// Re-derive the document's shared key/values from this response.
		if err := runner.Reparse(r.Context(), doc, m, output); err != nil {
			internalError(w, r, "Could not update document", err)
			return
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			internalError(w, r, "Could not load block", err)
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}

// UpdateResponse replaces the text of an existing response named by {responseId}
// and re-derives the document's shared attributes from the new text (same merge
// as RunBlock/ReparseResponse), so editing a response keeps the document
// consistent without re-running the model.
func UpdateResponse(st BlockStore, runner *blockrun.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st)
		if !ok {
			return
		}
		block, ok := findBlock(w, r, doc)
		if !ok {
			return
		}

		m, ok := mode.Get(block.Mode)
		if !ok {
			writeError(w, http.StatusInternalServerError, "Block has an unknown mode")
			return
		}

		responseID, err := pathID(r, "responseId")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid response id")
			return
		}
		found := false
		for _, resp := range block.Responses {
			if resp.ID == responseID {
				found = true
				break
			}
		}
		if !found {
			writeError(w, http.StatusNotFound, "Response not found")
			return
		}

		var body struct {
			Value string `json:"value"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := st.UpdateResponse(r.Context(), model.Response{ID: responseID, Value: body.Value}); err != nil {
			internalError(w, r, "Could not update response", err)
			return
		}

		// Re-derive the document's shared key/values from the edited text.
		if err := runner.Reparse(r.Context(), doc, m, body.Value); err != nil {
			internalError(w, r, "Could not update document", err)
			return
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			internalError(w, r, "Could not load block", err)
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}

// DeleteBlock removes a block from a document. Its attributes and responses are
// removed by the database via ON DELETE CASCADE.
func DeleteBlock(st BlockStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st)
		if !ok {
			return
		}
		block, ok := findBlock(w, r, doc)
		if !ok {
			return
		}
		if err := st.DeleteBlock(r.Context(), block.ID); err != nil {
			notFoundOr500(w, r, err, "Block not found", "Could not delete block")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
