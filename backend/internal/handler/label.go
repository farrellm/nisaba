package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/llm"
	"github.com/farrellm/nisaba/internal/store"
)

// ListLabels returns the logged-in user's label names, ordered by name. Labels
// are a user-global taxonomy; this feeds the edit-labels dialog's pool of
// existing labels to apply to a document.
func ListLabels(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		labels, err := st.ListLabels(r.Context(), userID)
		if err != nil {
			internalError(w, r, "Could not load labels", err)
			return
		}

		names := make([]string, 0, len(labels))
		for _, l := range labels {
			names = append(names, l.Name)
		}
		writeJSON(w, http.StatusOK, names)
	}
}

// RenameLabel renames one of the caller's labels across every document at once.
// Body: {"name": <current>, "newName": <new>}. When a label already named newName
// exists the two are merged (the response's "merged" flag is true); otherwise the
// label is renamed in place. 400 on a blank newName, 404 when name doesn't exist.
func RenameLabel(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		var body struct {
			Name    string `json:"name"`
			NewName string `json:"newName"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		merged, err := st.RenameLabel(r.Context(), userID, body.Name, body.NewName)
		switch {
		case errors.Is(err, store.ErrEmptyName):
			writeError(w, http.StatusBadRequest, "New label name cannot be empty")
			return
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "Label not found")
			return
		case err != nil:
			internalError(w, r, "Could not rename label", err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]bool{"merged": merged})
	}
}

// DeleteLabel removes one of the caller's labels, detaching it from every document
// (the documents themselves are kept). The label is named via the ?name= query
// param. 404 when the name doesn't exist.
func DeleteLabel(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := sess.UserID(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name == "" {
			writeError(w, http.StatusBadRequest, "Missing label name")
			return
		}

		switch err := st.DeleteLabelByName(r.Context(), userID, name); {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "Label not found")
			return
		case err != nil:
			internalError(w, r, "Could not delete label", err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// SuggestDocumentLabels suggests descriptive labels for a document by analyzing
// its "story" attribute with a fixed model (llm.SuggestLabels). It is read-only:
// it returns candidates for the caller to review and apply itself via
// PUT /api/documents/{id} — it does not attach anything.
func SuggestDocumentLabels(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
		if !ok {
			return
		}

		story := strings.TrimSpace(doc.Attributes["story"])
		if story == "" {
			writeError(w, http.StatusBadRequest, "Document has no story to label yet")
			return
		}

		labels, err := llm.SuggestLabels(r.Context(), story)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Model request failed")
			return
		}
		writeJSON(w, http.StatusOK, labels)
	}
}

// RecommendDocumentLabels picks the labels from the caller's existing pool that
// fit a document's "story" attribute (llm.SelectLabels). Like SuggestDocumentLabels
// it is read-only and returns a subset for the caller to apply itself via PUT; the
// difference is it chooses among labels the user already has rather than inventing
// new ones.
func RecommendDocumentLabels(st *store.Store, sess *auth.Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, ok := ownedDocument(w, r, st, sess)
		if !ok {
			return
		}

		story := strings.TrimSpace(doc.Attributes["story"])
		if story == "" {
			writeError(w, http.StatusBadRequest, "Document has no story to label yet")
			return
		}

		labels, err := st.ListLabels(r.Context(), doc.UserID)
		if err != nil {
			internalError(w, r, "Could not load labels", err)
			return
		}
		names := make([]string, 0, len(labels))
		for _, l := range labels {
			names = append(names, l.Name)
		}

		recommended, err := llm.SelectLabels(r.Context(), story, names)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Model request failed")
			return
		}
		writeJSON(w, http.StatusOK, recommended)
	}
}
