package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/model"
)

// fakeDocumentStore serves canned ListDocuments results; every other method
// panics so a test that strays off the list path fails loudly.
type fakeDocumentStore struct {
	DocumentStore // panics on any method not overridden below

	docs    []model.Document
	listErr error
}

func (f *fakeDocumentStore) ListDocuments(context.Context, int64, bool) ([]model.Document, error) {
	return f.docs, f.listErr
}

// asUser stamps a logged-in user id onto the request, standing in for the
// RequireUser middleware.
func asUser(r *http.Request, id int64) *http.Request {
	return r.WithContext(auth.WithUserID(r.Context(), id))
}

func TestListDocumentsRequiresUser(t *testing.T) {
	rec := httptest.NewRecorder()
	ListDocuments(&fakeDocumentStore{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/documents", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if got := rec.Body.String(); got != "{\"error\":\"Not logged in\"}\n" {
		t.Errorf("body = %q", got)
	}
}

func TestListDocumentsEmptyIsJSONArray(t *testing.T) {
	rec := httptest.NewRecorder()
	req := asUser(httptest.NewRequest(http.MethodGet, "/api/documents", nil), 1)
	ListDocuments(&fakeDocumentStore{docs: nil}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	// A nil slice from the store must serialize as [], never null.
	if got := rec.Body.String(); got != "[]\n" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestListDocumentsStoreError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := asUser(httptest.NewRequest(http.MethodGet, "/api/documents", nil), 1)
	ListDocuments(&fakeDocumentStore{listErr: errors.New("boom")}).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "Could not load documents" {
		t.Errorf("error = %q", body["error"])
	}
}

func TestListDocumentsReturnsDocs(t *testing.T) {
	rec := httptest.NewRecorder()
	req := asUser(httptest.NewRequest(http.MethodGet, "/api/documents", nil), 1)
	ListDocuments(&fakeDocumentStore{docs: []model.Document{{ID: 7, Title: "one"}}}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var docs []model.Document
	if err := json.Unmarshal(rec.Body.Bytes(), &docs); err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].ID != 7 || docs[0].Title != "one" {
		t.Errorf("docs = %+v", docs)
	}
}
