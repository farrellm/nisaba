package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cbroglie/mustache"
	"github.com/go-chi/chi/v5"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/llm"
	"github.com/farrellm/nisaba/internal/mode"
	"github.com/farrellm/nisaba/internal/model"
	"github.com/farrellm/nisaba/internal/store"
)

// ownedDocument loads the document named by the {id} URL param and confirms the
// logged-in user owns it. On any failure it writes the appropriate response and
// returns ok=false; resources owned by another user surface as 404 so their
// existence isn't leaked.
func ownedDocument(w http.ResponseWriter, r *http.Request, st *store.Store, sess *auth.Sessions) (model.Document, bool) {
	userID, ok := sess.UserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Not logged in")
		return model.Document{}, false
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid document id")
		return model.Document{}, false
	}
	doc, err := st.GetDocument(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Document not found")
			return model.Document{}, false
		}
		writeError(w, http.StatusInternalServerError, "Could not load document")
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
	blockID, err := strconv.ParseInt(chi.URLParam(r, "blockId"), 10, 64)
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
func CreateBlock(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
		if !ok {
			return
		}

		var body newBlock
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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
			writeError(w, http.StatusInternalServerError, "Could not create block")
			return
		}

		attrs := make(map[string]string, len(m.Keys))
		for _, key := range m.Keys {
			attrs[key] = doc.Attributes[key] // "" when absent
		}
		if err := st.ReplaceBlockAttributes(r.Context(), block.ID, attrs); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not create block")
			return
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load block")
			return
		}
		writeJSON(w, http.StatusCreated, hydrated)
	}
}

type updateBlock struct {
	Attributes map[string]string `json:"attributes"`
}

// mergedBlockAttrs builds the block's attribute map for the mode's fixed key
// set, taking each key from body when present and otherwise from the block's
// existing value. Keys outside the mode's key set are dropped, so the result
// always matches the mode.
func mergedBlockAttrs(block model.Block, m mode.Mode, body map[string]string) map[string]string {
	attrs := make(map[string]string, len(m.Keys))
	for _, key := range m.Keys {
		if v, present := body[key]; present {
			attrs[key] = v
		} else {
			attrs[key] = block.Attributes[key]
		}
	}
	return attrs
}

// UpdateBlock replaces a block's key/values. Keys outside the mode's fixed key
// set are ignored, so the stored attributes always match the mode.
func UpdateBlock(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
		if !ok {
			return
		}
		block, ok := findBlock(w, r, doc)
		if !ok {
			return
		}

		var body updateBlock
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		m, ok := mode.Get(block.Mode)
		if !ok {
			writeError(w, http.StatusInternalServerError, "Block has an unknown mode")
			return
		}
		attrs := mergedBlockAttrs(block, m, body.Attributes)
		if err := st.ReplaceBlockAttributes(r.Context(), block.ID, attrs); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not update block")
			return
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load block")
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}

// CopyBlock saves the block's key/values and promotes them into the document's
// shared attributes (merging, so values set by other blocks survive). It accepts
// the same body shape as UpdateBlock so the caller's on-screen edits are saved
// before they're copied up.
func CopyBlock(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
		if !ok {
			return
		}
		block, ok := findBlock(w, r, doc)
		if !ok {
			return
		}

		var body updateBlock
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		m, ok := mode.Get(block.Mode)
		if !ok {
			writeError(w, http.StatusInternalServerError, "Block has an unknown mode")
			return
		}
		attrs := mergedBlockAttrs(block, m, body.Attributes)
		if err := st.ReplaceBlockAttributes(r.Context(), block.ID, attrs); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not update block")
			return
		}
		if err := st.MergeDocumentAttributes(r.Context(), doc.ID, attrs); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not update document")
			return
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load block")
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}

// RunBlock saves the block's key/values, promotes them into the document's
// shared attributes, renders the mode's mustache template against them to build
// a prompt, runs it, saves the result as a response, and writes the result back
// into the document's attributes under the mode's output key. It accepts the
// same optional body as UpdateBlock so the caller's on-screen edits are saved
// before the run.
func RunBlock(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, block, m, prompt, system, ok := prepareRun(w, r, st, sess)
		if !ok {
			return
		}

		output, err := llm.Generate(r.Context(), doc.SelectedModel, system, prompt, m.Tools)
		if err != nil {
			slog.Error("run failed: model request",
				"err", err, "user", doc.UserID, "doc", doc.ID,
				"block", block.ID, "mode", block.Mode, "model", doc.SelectedModel)
			writeError(w, http.StatusBadGateway, "Model request failed: "+err.Error())
			return
		}

		hydrated, err := finishRun(r.Context(), st, doc, block, m, output)
		if err != nil {
			slog.Error("run failed: save response",
				"err", err, "user", doc.UserID, "doc", doc.ID,
				"block", block.ID, "mode", block.Mode, "model", doc.SelectedModel)
			writeError(w, http.StatusInternalServerError, "Could not save response")
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}

// RunBlockStream is the streaming variant of RunBlock: it runs the model with
// llm.GenerateStream and pushes the reply to the client as it arrives, framed as
// newline-delimited JSON (NDJSON). Each line is one event:
//
//	{"type":"delta","text":"..."}   incremental text
//	{"type":"ping"}                  keepalive while the model runs (client ignores)
//	{"type":"error","message":"..."} terminal failure (after streaming began)
//	{"type":"done","block":{...}}    the fully hydrated block, like RunBlock's body
//
// Setup/validation failures before streaming begins still use the normal JSON
// error path (prepareRun); once the 200 NDJSON stream has started, errors can
// only be reported as an "error" event.
func RunBlockStream(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, block, m, prompt, system, ok := prepareRun(w, r, st, sess)
		if !ok {
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

		output, err := llm.GenerateStream(r.Context(), doc.SelectedModel, system, prompt, m.Tools,
			func(delta string) {
				writeEvent(map[string]string{"type": "delta", "text": delta})
			})
		// Stop the keepalive and wait for its goroutine to exit before writing the
		// terminal event, so no stray ping can land after done/error.
		close(stop)
		<-finished
		if err != nil {
			slog.Error("run failed: model request (stream)",
				"err", err, "user", doc.UserID, "doc", doc.ID,
				"block", block.ID, "mode", block.Mode, "model", doc.SelectedModel)
			writeEvent(map[string]string{"type": "error", "message": "Model request failed: " + err.Error()})
			return
		}

		hydrated, err := finishRun(r.Context(), st, doc, block, m, output)
		if err != nil {
			slog.Error("run failed: save response (stream)",
				"err", err, "user", doc.UserID, "doc", doc.ID,
				"block", block.ID, "mode", block.Mode, "model", doc.SelectedModel)
			writeEvent(map[string]string{"type": "error", "message": "Could not save response"})
			return
		}
		writeEvent(map[string]any{"type": "done", "block": hydrated})
	}
}

// prepareRun performs the shared setup for a block run: it confirms ownership,
// resolves the mode and selected model, persists the caller's attribute edits
// into both the block and the document, and renders the prompt + system prompt
// (honoring per-user template overrides). On any failure it writes the
// appropriate JSON error and returns ok=false.
func prepareRun(w http.ResponseWriter, r *http.Request, st *store.Store, sess *auth.Sessions) (doc model.Document, block model.Block, m mode.Mode, prompt, system string, ok bool) {
	doc, ok = ownedDocument(w, r, st, sess)
	if !ok {
		return
	}
	block, ok = findBlock(w, r, doc)
	if !ok {
		return
	}

	m, ok = mode.Get(block.Mode)
	if !ok {
		slog.Warn("run failed: unknown mode",
			"user", doc.UserID, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		writeError(w, http.StatusInternalServerError, "Block has an unknown mode")
		return doc, block, m, "", "", false
	}

	if doc.SelectedModel == "" {
		slog.Warn("run failed: no model selected",
			"user", doc.UserID, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		writeError(w, http.StatusBadRequest, "No model selected")
		return doc, block, m, "", "", false
	}

	// Save the caller's edits (empty body falls back to existing values) and
	// promote them into the document's shared attributes before running.
	var body updateBlock
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		slog.Error("run failed: decode request body",
			"err", err, "user", doc.UserID, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return doc, block, m, "", "", false
	}
	attrs := mergedBlockAttrs(block, m, body.Attributes)
	if err := st.ReplaceBlockAttributes(r.Context(), block.ID, attrs); err != nil {
		slog.Error("run failed: update block attributes",
			"err", err, "user", doc.UserID, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		writeError(w, http.StatusInternalServerError, "Could not update block")
		return doc, block, m, "", "", false
	}
	if err := st.MergeDocumentAttributes(r.Context(), doc.ID, attrs); err != nil {
		slog.Error("run failed: merge document attributes",
			"err", err, "user", doc.UserID, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		writeError(w, http.StatusInternalServerError, "Could not update document")
		return doc, block, m, "", "", false
	}

	// Resolve a per-user template override (falls back to the embedded
	// default when the user has none); a lookup failure degrades gracefully.
	username := ""
	if u, err := st.GetUser(r.Context(), doc.UserID); err == nil {
		username = u.Username
	}

	prompt, err := mustache.Render(mode.TemplateFor(username, m), attrs)
	if err != nil {
		slog.Error("run failed: render prompt",
			"err", err, "user", username, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		writeError(w, http.StatusInternalServerError, "Could not render prompt")
		return doc, block, m, "", "", false
	}

	provider := llm.ProviderFor(doc.SelectedModel)
	systemTmpl, systemSource := mode.SystemPrompt(username, provider)
	slog.Info("system prompt",
		"user", username, "provider", provider,
		"model", doc.SelectedModel, "source", systemSource)
	system, err = mustache.Render(systemTmpl, attrs)
	if err != nil {
		slog.Error("run failed: render system prompt",
			"err", err, "user", username, "provider", provider,
			"doc", doc.ID, "block", block.ID, "mode", block.Mode, "model", doc.SelectedModel)
		writeError(w, http.StatusInternalServerError, "Could not render system prompt")
		return doc, block, m, "", "", false
	}

	return doc, block, m, prompt, system, true
}

// finishRun persists a completed model reply and feeds it back into the
// document: it stores the response, reparses it into the document's shared
// attributes, and returns the freshly hydrated block. Shared by RunBlock and
// RunBlockStream.
func finishRun(ctx context.Context, st *store.Store, doc model.Document, block model.Block, m mode.Mode, output string) (model.Block, error) {
	if _, err := st.CreateResponse(ctx, model.Response{
		BlockID:  block.ID,
		Value:    output,
		Model:    doc.SelectedModel,
		Position: len(block.Responses),
	}); err != nil {
		return model.Block{}, err
	}

	// Feed the result back into the document's shared key/values.
	if err := reparseInto(ctx, st, doc, m, output); err != nil {
		return model.Block{}, err
	}

	return st.GetBlock(ctx, block.ID)
}

// reparseInto re-derives a document's shared attributes from a response's text:
// top-level XML tags each populate an attribute (any tag name; nested tags stay
// verbatim in the value), the mode's output key (when set) wins over a same-named
// tag, and the merged result is written back to the document.
func reparseInto(ctx context.Context, st *store.Store, doc model.Document, m mode.Mode, output string) error {
	updates := parseTopLevelTags(output)
	applyRenames(updates, m.Renames)
	if m.Output != "" {
		updates[m.Output] = output
	}
	if len(updates) > 0 {
		if err := st.MergeDocumentAttributes(ctx, doc.ID, updates); err != nil {
			return err
		}
		if _, err := st.UpdateDocument(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

// ReparseResponse re-runs only the parse + merge step against an existing
// response named by {responseId}, without calling the model. It feeds that
// response's stored text back into the document's shared attributes the same way
// RunBlock does (top-level XML tags plus the mode's output key, merged), so a user
// can re-derive attributes from any past response.
func ReparseResponse(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
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

		responseID, err := strconv.ParseInt(chi.URLParam(r, "responseId"), 10, 64)
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
		if err := reparseInto(r.Context(), st, doc, m, output); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not update document")
			return
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load block")
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}

// UpdateResponse replaces the text of an existing response named by {responseId}
// and re-derives the document's shared attributes from the new text (same merge
// as RunBlock/ReparseResponse), so editing a response keeps the document
// consistent without re-running the model.
func UpdateResponse(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
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

		responseID, err := strconv.ParseInt(chi.URLParam(r, "responseId"), 10, 64)
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
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := st.UpdateResponse(r.Context(), model.Response{ID: responseID, Value: body.Value}); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not update response")
			return
		}

		// Re-derive the document's shared key/values from the edited text.
		if err := reparseInto(r.Context(), st, doc, m, body.Value); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not update document")
			return
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load block")
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}

// DeleteBlock removes a block from a document. Its attributes and responses are
// removed by the database via ON DELETE CASCADE.
func DeleteBlock(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
		if !ok {
			return
		}
		block, ok := findBlock(w, r, doc)
		if !ok {
			return
		}
		if err := st.DeleteBlock(r.Context(), block.ID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "Block not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Could not delete block")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
