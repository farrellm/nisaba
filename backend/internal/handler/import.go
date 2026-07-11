package handler

import (
	"context"
	"net/http"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/mode"
	"github.com/farrellm/nisaba/internal/model"
	"github.com/farrellm/nisaba/internal/store"
)

// fallbackMode is the registry mode assigned to a legacy block whose mode name is
// neither in the current registry nor in legacyModeMap. It's a safe catch-all: the
// block keeps its attributes and responses and stays runnable/editable.
const fallbackMode = "generic"

// legacyModeMap translates block mode names from the legacy apps (Anansi/Charlotte)
// onto the current fixed mode registry where the names have diverged but a clear
// analog exists. Names that already match the registry aren't listed — resolveLegacyMode
// checks the registry first. Anything neither in the registry nor here falls back to
// `generic` (see resolveLegacyMode). Charlotte's vocabulary is larger and messier than
// Anansi's, so most of these entries come from Charlotte data.
var legacyModeMap = map[string]string{
	// stories / continuations
	"story-full":     "story",
	"story-opus":     "story",
	"story-start":    "story",
	"story-next":     "story-sequel",
	"continue-opus":  "story-sequel",
	"contiunue-opus": "story-sequel", // legacy typo, seen in the data
	"rewrite":        "story-revise",
	"editor-agent":   "story-edit",
	// brainstorming
	"brainstorm":              "brainstorm-1",
	"brainstorm-story":        "brainstorm-1",
	"brainstorm-c":            "brainstorm-creative-2",
	"brainstorm-thinking":     "brainstorm-tools-1",
	"brainstorm-thinking-1":   "brainstorm-tools-1",
	"brainstorm-thinking-2":   "brainstorm-tools-2",
	"brainstorm-c-thinking-1": "brainstorm-tools-1",
	// outlines
	"revise-outline":            "revise-outline-1",
	"revise-outline-1-thinking": "revise-outline-1",
	"revise-outline-2-thinking": "revise-outline-2",
	// authors
	"authors-thinking": "authors",
}

// resolveLegacyMode maps a legacy block mode name onto a current registry mode name.
// It prefers the name itself when the registry already has it, then a legacyModeMap
// translation (when the target is a real registry mode), and otherwise falls back to
// `generic` — so every legacy block imports as a usable mode.
func resolveLegacyMode(name string) string {
	if _, ok := mode.Get(name); ok {
		return name
	}
	if mapped, ok := legacyModeMap[name]; ok {
		if _, ok := mode.Get(mapped); ok {
			return mapped
		}
	}
	return fallbackMode
}

// importLegacyDocument recreates a legacy (Anansi/Charlotte) document aggregate as a
// brand-new document owned by userID, and returns the fully-populated new document.
// Each block's mode is resolved onto the current registry (with a `generic` fallback).
// The write is not a single transaction (the store methods each own theirs).
func importLegacyDocument(ctx context.Context, st *store.Store, userID int64, src model.Document) (model.Document, error) {
	doc, err := st.CreateDocument(ctx, model.Document{
		UserID:        userID,
		Title:         src.Title,
		URL:           src.URL,
		SelectedModel: "claude-sonnet-5",
	})
	if err != nil {
		return model.Document{}, err
	}

	if len(src.Attributes) > 0 {
		if err := st.ReplaceDocumentAttributes(ctx, doc.ID, src.Attributes); err != nil {
			return model.Document{}, err
		}
	}

	if len(src.Labels) > 0 {
		if err := st.SetDocumentLabels(ctx, userID, doc.ID, src.Labels); err != nil {
			return model.Document{}, err
		}
	}

	for i, b := range src.Blocks {
		block, err := st.CreateBlock(ctx, model.Block{
			DocumentID: doc.ID,
			Mode:       resolveLegacyMode(b.Mode),
			Position:   i,
		})
		if err != nil {
			return model.Document{}, err
		}
		if len(b.Attributes) > 0 {
			if err := st.ReplaceBlockAttributes(ctx, block.ID, b.Attributes); err != nil {
				return model.Document{}, err
			}
		}
		for j, resp := range b.Responses {
			if _, err := st.CreateResponse(ctx, model.Response{
				BlockID:  block.ID,
				Value:    resp.Value,
				Model:    resp.Model,
				Position: j,
			}); err != nil {
				return model.Document{}, err
			}
		}
	}

	return st.GetDocument(ctx, doc.ID)
}

// ImportReflexDocument copies a legacy Anansi (reflex.db) document into a new document
// owned by the logged-in user and returns it (201). 404 if the source id is unknown.
func ImportReflexDocument(rs *store.ReflexStore, st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid document id")
			return
		}
		src, err := rs.GetReflexDocument(r.Context(), id)
		if err != nil {
			notFoundOr500(w, r, err, "Document not found", "Could not load document")
			return
		}
		doc, err := importLegacyDocument(r.Context(), st, userID, src)
		if err != nil {
			internalError(w, r, "Could not import document", err)
			return
		}
		writeJSON(w, http.StatusCreated, doc)
	}
}

// ImportCharlotteDocument copies a legacy Charlotte document into a new document owned
// by the logged-in user and returns it (201). 404 if the source id is unknown.
func ImportCharlotteDocument(cs *store.CharlotteStore, st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid document id")
			return
		}
		src, err := cs.GetCharlotteDocument(r.Context(), id)
		if err != nil {
			notFoundOr500(w, r, err, "Document not found", "Could not load document")
			return
		}
		doc, err := importLegacyDocument(r.Context(), st, userID, src)
		if err != nil {
			internalError(w, r, "Could not import document", err)
			return
		}
		writeJSON(w, http.StatusCreated, doc)
	}
}
