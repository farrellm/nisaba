package store

import "testing"

func TestParseCharlotteDoc(t *testing.T) {
	data := []byte(`{
		"docTitle": "Alchemical Salesgirl",
		"docUrl": "https://example.com/post",
		"docTags": {"author": "Kristen Ashley"},
		"docBlocks": [
			{
				"blockMode": "story-full",
				"blockModel": "gemini-2.5-pro",
				"blockTags": {"characters": "Lirael"},
				"blockResponses": ["once upon a time", "the end"]
			},
			{
				"blockMode": "edit",
				"blockModel": "claude-3-opus",
				"blockTags": {},
				"blockResponses": ["revised"]
			}
		]
	}`)

	doc, err := parseCharlotteDoc(data, 7, "alchemical-salesgirl.sc")
	if err != nil {
		t.Fatalf("parseCharlotteDoc: %v", err)
	}

	if doc.ID != 7 {
		t.Errorf("ID = %d, want 7", doc.ID)
	}
	if doc.Title != "Alchemical Salesgirl" {
		t.Errorf("Title = %q", doc.Title)
	}
	if doc.URL == nil || *doc.URL != "https://example.com/post" {
		t.Errorf("URL = %v", doc.URL)
	}
	if doc.IsArchived {
		t.Error("IsArchived = true, want false for non-archive name")
	}
	if doc.Attributes["author"] != "Kristen Ashley" {
		t.Errorf("Attributes = %v", doc.Attributes)
	}
	if len(doc.Blocks) != 2 {
		t.Fatalf("len(Blocks) = %d, want 2", len(doc.Blocks))
	}

	b0 := doc.Blocks[0]
	if b0.Mode != "story-full" || b0.Position != 0 || b0.DocumentID != 7 {
		t.Errorf("block0 = %+v", b0)
	}
	if len(b0.Responses) != 2 {
		t.Fatalf("len(block0.Responses) = %d, want 2", len(b0.Responses))
	}
	if b0.Responses[0].Value != "once upon a time" || b0.Responses[0].Model != "gemini-2.5-pro" {
		t.Errorf("block0 response0 = %+v", b0.Responses[0])
	}

	// Block and response ids must be unique across the whole document.
	seenBlock := map[int64]bool{}
	seenResp := map[int64]bool{}
	for _, b := range doc.Blocks {
		if seenBlock[b.ID] {
			t.Errorf("duplicate block id %d", b.ID)
		}
		seenBlock[b.ID] = true
		for _, r := range b.Responses {
			if seenResp[r.ID] {
				t.Errorf("duplicate response id %d", r.ID)
			}
			seenResp[r.ID] = true
		}
	}
}

func TestParseCharlotteDocArchivedAndDefaults(t *testing.T) {
	// Minimal dump: no title/url, null tags — title falls back to the name, maps
	// default to empty (never nil), and the archive/ prefix marks it archived.
	doc, err := parseCharlotteDoc([]byte(`{"docBlocks": []}`), 1, "archive/2024-08/ceo.yaml")
	if err != nil {
		t.Fatalf("parseCharlotteDoc: %v", err)
	}
	if doc.Title != "archive/2024-08/ceo.yaml" {
		t.Errorf("Title = %q, want name fallback", doc.Title)
	}
	if !doc.IsArchived {
		t.Error("IsArchived = false, want true for archive/ name")
	}
	if doc.URL != nil {
		t.Errorf("URL = %v, want nil", doc.URL)
	}
	if doc.Attributes == nil {
		t.Error("Attributes is nil, want empty map")
	}
}

func TestParseCharlotteDocInvalidJSON(t *testing.T) {
	if _, err := parseCharlotteDoc([]byte("not json"), 1, "x"); err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
