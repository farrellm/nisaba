package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

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

		if doc.SelectedModel == "" {
			writeError(w, http.StatusBadRequest, "No model selected")
			return
		}

		// Save the caller's edits (empty body falls back to existing values) and
		// promote them into the document's shared attributes before running.
		var body updateBlock
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
			writeError(w, http.StatusBadRequest, "Invalid request body")
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

		// Resolve a per-user template override (falls back to the embedded
		// default when the user has none); a lookup failure degrades gracefully.
		username := ""
		if u, err := st.GetUser(r.Context(), doc.UserID); err == nil {
			username = u.Username
		}

		prompt, err := mustache.Render(mode.TemplateFor(username, m), attrs)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not render prompt")
			return
		}

		system, err := mustache.Render(mode.SystemPrompt(username), attrs)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not render system prompt")
			return
		}

		output, err := llm.Generate(r.Context(), doc.SelectedModel, system, prompt, m.Tools)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Model request failed")
			return
		}

		if _, err := st.CreateResponse(r.Context(), model.Response{
			BlockID:  block.ID,
			Value:    output,
			Model:    doc.SelectedModel,
			Position: len(block.Responses),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not save response")
			return
		}

		// Feed the result back into the document's shared key/values. Top-level
		// XML tags in the response each populate a document attribute (any tag
		// name; nested tags stay verbatim in the value). When the mode declares
		// an output key, the full response is also saved under it — set last so
		// it wins over a same-named tag.
		updates := parseTopLevelTags(output)
		if m.Output != "" {
			updates[m.Output] = output
		}
		if len(updates) > 0 {
			if err := st.MergeDocumentAttributes(r.Context(), doc.ID, updates); err != nil {
				writeError(w, http.StatusInternalServerError, "Could not update document")
				return
			}
			if _, err := st.UpdateDocument(r.Context(), doc); err != nil {
				writeError(w, http.StatusInternalServerError, "Could not update document")
				return
			}
		}

		hydrated, err := st.GetBlock(r.Context(), block.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load block")
			return
		}
		writeJSON(w, http.StatusOK, hydrated)
	}
}
