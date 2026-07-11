package handler

import (
	"context"
	"net/http"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/model"
)

// LegacySource is a read-only browser over one of the legacy apps' data:
// Anansi (store.ReflexStore, the reflex.db SQLite file) or Charlotte
// (store.CharlotteStore, the charlotte-cli executable). Both mirror the live
// document shapes, so one handler set serves both URL trees.
type LegacySource interface {
	ListDocuments(ctx context.Context) ([]model.Document, error)
	GetDocument(ctx context.Context, id int64) (model.Document, error)
}

// ListLegacyDocuments returns every legacy document as a summary for the
// read-only browser pages. It requires a logged-in session but is not scoped
// to the caller — the legacy users are unrelated to current accounts and the
// data is shared and read-only.
func ListLegacyDocuments(src LegacySource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserIDFrom(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		docs, err := src.ListDocuments(r.Context())
		if err != nil {
			internalError(w, r, "Could not load documents", err)
			return
		}
		writeJSON(w, http.StatusOK, docs)
	}
}

// GetLegacyDocument returns a single legacy document fully populated
// read-only, or 404 if it does not exist. Requires a logged-in session.
func GetLegacyDocument(src LegacySource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserIDFrom(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid document id")
			return
		}
		doc, err := src.GetDocument(r.Context(), id)
		if err != nil {
			notFoundOr500(w, r, err, "Document not found", "Could not load document")
			return
		}
		writeJSON(w, http.StatusOK, doc)
	}
}

// ImportLegacyDocument copies a legacy document into a new document owned by
// the logged-in user and returns it (201). 404 if the source id is unknown.
func ImportLegacyDocument(src LegacySource, st ImportStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFrom(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		id, err := pathID(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid document id")
			return
		}
		legacyDoc, err := src.GetDocument(r.Context(), id)
		if err != nil {
			notFoundOr500(w, r, err, "Document not found", "Could not load document")
			return
		}
		doc, err := importLegacyDocument(r.Context(), st, userID, legacyDoc)
		if err != nil {
			internalError(w, r, "Could not import document", err)
			return
		}
		writeJSON(w, http.StatusCreated, doc)
	}
}
