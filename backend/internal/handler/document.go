package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/llm"
	"github.com/farrellm/nisaba/internal/model"
	"github.com/farrellm/nisaba/internal/store"
)

// ListDocuments returns the logged-in user's documents as summaries, most
// recently updated first. Archived documents are included only when the
// request carries ?archived=true.
func ListDocuments(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		includeArchived := r.URL.Query().Get("archived") == "true"
		docs, err := st.ListDocuments(r.Context(), id, includeArchived)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load documents")
			return
		}
		if docs == nil {
			docs = []model.Document{}
		}
		writeJSON(w, http.StatusOK, docs)
	}
}

type newDocument struct {
	Title string  `json:"title"`
	URL   *string `json:"url"`
}

// CreateDocument creates a document owned by the logged-in user and returns it.
func CreateDocument(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		var body newDocument
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		title := strings.TrimSpace(body.Title)
		if title == "" {
			writeError(w, http.StatusBadRequest, "Title is required")
			return
		}

		url := body.URL
		if url != nil {
			if trimmed := strings.TrimSpace(*url); trimmed == "" {
				url = nil
			} else {
				url = &trimmed
			}
		}

		doc, err := st.CreateDocument(r.Context(), model.Document{
			UserID:        id,
			Title:         title,
			URL:           url,
			SelectedModel: "claude-sonnet-5",
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not create document")
			return
		}
		writeJSON(w, http.StatusCreated, doc)
	}
}

// GetDocument returns a single fully-populated document owned by the logged-in
// user, or 404 if it does not exist or belongs to someone else.
func GetDocument(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid document id")
			return
		}

		doc, err := st.GetDocument(r.Context(), id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "Document not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Could not load document")
			return
		}
		if doc.UserID != userID {
			// Don't reveal that a document exists for another user.
			writeError(w, http.StatusNotFound, "Document not found")
			return
		}
		writeJSON(w, http.StatusOK, doc)
	}
}

// PublicDocumentAttribute returns a single document attribute value without
// requiring authentication, for the chrome-free markdown view. Access is open by
// document id (ids are sequential and guessable — by design). Returns an empty
// value when the document or key does not exist, so it never leaks existence.
func PublicDocumentAttribute(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid document id")
			return
		}
		key := chi.URLParam(r, "key")
		value, _, err := st.GetDocumentAttribute(r.Context(), id, key)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load attribute")
			return
		}
		title, err := st.GetDocumentTitle(r.Context(), id)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusInternalServerError, "Could not load attribute")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"value": value, "title": title})
	}
}

type updateDocument struct {
	SelectedModel *string            `json:"selectedModel"`
	Attributes    *map[string]string `json:"attributes"`
	IsArchived    *bool              `json:"isArchived"`
	Labels        *[]string          `json:"labels"`
}

// UpdateDocument changes a document's selected model, attribute values, archive
// state, and/or labels, and returns the refreshed, fully-populated document. Each
// field is optional; only the fields present in the request are applied.
func UpdateDocument(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
		if !ok {
			return
		}

		var body updateDocument
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if body.SelectedModel != nil || body.IsArchived != nil {
			if body.SelectedModel != nil {
				if !llm.Valid(*body.SelectedModel) {
					writeError(w, http.StatusBadRequest, "Unknown model")
					return
				}
				doc.SelectedModel = *body.SelectedModel
			}
			if body.IsArchived != nil {
				doc.IsArchived = *body.IsArchived
			}
			if _, err := st.UpdateDocument(r.Context(), doc); err != nil {
				writeError(w, http.StatusInternalServerError, "Could not update document")
				return
			}
		}

		if body.Attributes != nil {
			if err := st.MergeDocumentAttributes(r.Context(), doc.ID, *body.Attributes); err != nil {
				writeError(w, http.StatusInternalServerError, "Could not update document")
				return
			}
		}

		if body.Labels != nil {
			if err := st.SetDocumentLabels(r.Context(), doc.UserID, doc.ID, *body.Labels); err != nil {
				writeError(w, http.StatusInternalServerError, "Could not update document")
				return
			}
		}

		updated, err := st.GetDocument(r.Context(), doc.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load document")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

// DeleteDocument removes a document the logged-in user owns, cascading to its
// blocks, attributes, and label taggings. Returns 404 if it does not exist or
// belongs to someone else.
func DeleteDocument(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
		if !ok {
			return
		}
		if err := st.DeleteDocument(r.Context(), doc.UserID, doc.ID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "Document not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Could not delete document")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
