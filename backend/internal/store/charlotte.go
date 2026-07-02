package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/farrellm/nisaba/internal/model"
)

// CharlotteStore is a read-only data-access layer over the external "charlotte-cli"
// executable, an older file-based version of this app, browsed by the "Charlotte"
// pages. It shells out (`--list` to enumerate document names, `--doc <name>` to dump
// one as JSON) and maps the result onto the shapes the live Store returns
// (model.Document and friends) so the same API/frontend types are reused.
//
// Documents are identified by name, not int id; we assign stable ids by indexing the
// sorted `--list` output, so the same id resolves to the same name on a later fetch.
type CharlotteStore struct {
	exe string
}

// NewCharlotteStore returns a CharlotteStore that runs the given executable (a name
// resolved on PATH, or an absolute path). It never fails here — a missing executable
// surfaces as an error per request rather than crashing startup.
func NewCharlotteStore(exe string) *CharlotteStore {
	return &CharlotteStore{exe: exe}
}

// list runs `<exe> --list` and returns the document names sorted lexicographically,
// which fixes the id↔name mapping used by both list and get.
func (s *CharlotteStore) list(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, s.exe, "--list").Output()
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// ListCharlotteDocuments returns every Charlotte document as a summary (no blocks or
// attributes). The name is used as the title and as the stable id (its 1-based index in
// the sorted list); names under "archive/" are marked archived. Timestamps are left zero
// — the frontend hides them for this source.
func (s *CharlotteStore) ListCharlotteDocuments(ctx context.Context) ([]model.Document, error) {
	names, err := s.list(ctx)
	if err != nil {
		return nil, err
	}
	docs := make([]model.Document, 0, len(names))
	for i, name := range names {
		title, labels := parseCharlotteName(name)
		if labels == nil {
			labels = []string{}
		}
		docs = append(docs, model.Document{
			ID:         int64(i + 1),
			Title:      title,
			IsArchived: strings.HasPrefix(name, "archive/"),
			Attributes: map[string]string{},
			Blocks:     []model.Block{},
			Labels:     labels,
			PostURLs:   []string{},
		})
	}
	return docs, nil
}

// GetCharlotteDocument resolves the id to a name via the sorted list, dumps that
// document with `--doc <name>`, and maps the JSON onto a model.Document. An out-of-range
// id is ErrNotFound; a non-zero CLI exit (the doc fails to parse in the legacy tool)
// surfaces as an error.
func (s *CharlotteStore) GetCharlotteDocument(ctx context.Context, id int64) (model.Document, error) {
	names, err := s.list(ctx)
	if err != nil {
		return model.Document{}, err
	}
	if id < 1 || id > int64(len(names)) {
		return model.Document{}, ErrNotFound
	}
	name := names[id-1]

	out, err := exec.CommandContext(ctx, s.exe, "--doc", name).Output()
	if err != nil {
		return model.Document{}, fmt.Errorf("charlotte --doc %q: %w", name, err)
	}
	return parseCharlotteDoc(out, id, name)
}

// parseCharlotteName splits a document name into its display title and any labels.
// Archived names follow "archive/${label}/${name}": the leading "archive/" marks
// the archived state, the next path segment becomes a label, and the remainder is
// the title. Names with no label segment ("archive/${name}" or a bare name) yield
// the name unchanged and no labels.
func parseCharlotteName(name string) (title string, labels []string) {
	rest := strings.TrimPrefix(name, "archive/")
	if rest != name { // archived
		if i := strings.Index(rest, "/"); i >= 0 {
			return rest[i+1:], []string{rest[:i]}
		}
	}
	return rest, nil
}

// charlotteDoc / charlotteBlock mirror the JSON shape emitted by `charlotte-cli --doc`.
// Fields the read-only view does not use (blockPrompts/blockTemp/...) are ignored.
type charlotteDoc struct {
	DocTitle  string            `json:"docTitle"`
	DocURL    string            `json:"docUrl"`
	DocTags   map[string]string `json:"docTags"`
	DocBlocks []charlotteBlock  `json:"docBlocks"`
}

type charlotteBlock struct {
	BlockMode      string            `json:"blockMode"`
	BlockModel     string            `json:"blockModel"`
	BlockTags      map[string]string `json:"blockTags"`
	BlockResponses []string          `json:"blockResponses"`
}

// parseCharlotteDoc decodes a `--doc` JSON dump and maps it onto a model.Document with
// the given id and name. Block and response ids are synthesized from running counters so
// they are unique across the whole document. It is pure (no shelling out) for testability.
func parseCharlotteDoc(data []byte, id int64, name string) (model.Document, error) {
	var cd charlotteDoc
	if err := json.Unmarshal(data, &cd); err != nil {
		return model.Document{}, fmt.Errorf("charlotte parse %q: %w", name, err)
	}

	title, labels := parseCharlotteName(name)
	if labels == nil {
		labels = []string{}
	}
	doc := model.Document{
		ID:         id,
		Title:      cd.DocTitle,
		IsArchived: strings.HasPrefix(name, "archive/"),
		Attributes: cd.DocTags,
		Blocks:     []model.Block{},
		Labels:     labels,
		PostURLs:   []string{},
	}
	if doc.Title == "" {
		doc.Title = title
	}
	if doc.Attributes == nil {
		doc.Attributes = map[string]string{}
	}
	if cd.DocURL != "" {
		url := cd.DocURL
		doc.URL = &url
	}

	var blockID, responseID int64
	for i, cb := range cd.DocBlocks {
		blockID++
		attrs := cb.BlockTags
		if attrs == nil {
			attrs = map[string]string{}
		}
		b := model.Block{
			ID:         blockID,
			DocumentID: id,
			Mode:       cb.BlockMode,
			Position:   i,
			Attributes: attrs,
			Responses:  []model.Response{},
		}
		for j, value := range cb.BlockResponses {
			responseID++
			b.Responses = append(b.Responses, model.Response{
				ID:       responseID,
				BlockID:  blockID,
				Value:    value,
				Model:    cb.BlockModel,
				Position: j,
			})
		}
		doc.Blocks = append(doc.Blocks, b)
	}
	return doc, nil
}
