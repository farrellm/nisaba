package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/farrellm/nisaba/internal/auth"
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
			UserID: id,
			Title:  title,
			URL:    url,
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
