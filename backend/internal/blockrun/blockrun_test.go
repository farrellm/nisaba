package blockrun

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/farrellm/nisaba/internal/llm"
	"github.com/farrellm/nisaba/internal/mode"
	"github.com/farrellm/nisaba/internal/model"
)

// fakeStore records the writes a run performs and serves canned reads.
type fakeStore struct {
	blockAttrs map[int64]map[string]string // ReplaceBlockAttributes writes
	docAttrs   map[int64]map[string]string // MergeDocumentAttributes writes (merged)
	responses  []model.Response
	user       model.User
	block      model.Block // served by GetBlock

	replaceErr error
	mergeErr   error
	createErr  error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		blockAttrs: map[int64]map[string]string{},
		docAttrs:   map[int64]map[string]string{},
		user:       model.User{ID: 1, Username: "tester"},
	}
}

func (f *fakeStore) ReplaceBlockAttributes(_ context.Context, blockID int64, attrs map[string]string) error {
	if f.replaceErr != nil {
		return f.replaceErr
	}
	f.blockAttrs[blockID] = attrs
	return nil
}

func (f *fakeStore) MergeDocumentAttributes(_ context.Context, documentID int64, attrs map[string]string) error {
	if f.mergeErr != nil {
		return f.mergeErr
	}
	merged := f.docAttrs[documentID]
	if merged == nil {
		merged = map[string]string{}
		f.docAttrs[documentID] = merged
	}
	for k, v := range attrs {
		merged[k] = v
	}
	return nil
}

func (f *fakeStore) UpdateDocument(_ context.Context, doc model.Document) (model.Document, error) {
	return doc, nil
}

func (f *fakeStore) GetUser(_ context.Context, id int64) (model.User, error) {
	return f.user, nil
}

func (f *fakeStore) CreateResponse(_ context.Context, resp model.Response) (model.Response, error) {
	if f.createErr != nil {
		return model.Response{}, f.createErr
	}
	f.responses = append(f.responses, resp)
	return resp, nil
}

func (f *fakeStore) GetBlock(_ context.Context, id int64) (model.Block, error) {
	return f.block, nil
}

// fakeGen returns a canned reply (streamed as a single delta) or a canned error.
type fakeGen struct {
	output string
	err    error
}

func (g fakeGen) Generate(context.Context, string, string, string, []llm.Tool) (string, error) {
	return g.output, g.err
}

func (g fakeGen) GenerateStream(_ context.Context, _, _, _ string, _ []llm.Tool, onDelta func(string)) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	onDelta(g.output)
	return g.output, nil
}

func testService(t *testing.T, st Store, gen Generator) *Service {
	t.Helper()
	return New(st, gen, mode.NewTemplates(t.TempDir()+"/templates"))
}

func testDocBlock() (model.Document, model.Block) {
	doc := model.Document{ID: 10, UserID: 1, SelectedModel: "claude-sonnet-5"}
	block := model.Block{
		ID:         20,
		DocumentID: doc.ID,
		Mode:       "story",
		Attributes: map[string]string{"characters": "Ada", "author": "Le Guin", "outline": "a voyage"},
	}
	return doc, block
}

func TestPrepareUnknownMode(t *testing.T) {
	doc, block := testDocBlock()
	block.Mode = "no-such-mode"
	_, err := testService(t, newFakeStore(), fakeGen{}).Prepare(context.Background(), doc, block, nil)
	if !errors.Is(err, ErrUnknownMode) {
		t.Fatalf("err = %v, want ErrUnknownMode", err)
	}
}

func TestPrepareNoModel(t *testing.T) {
	doc, block := testDocBlock()
	doc.SelectedModel = ""
	_, err := testService(t, newFakeStore(), fakeGen{}).Prepare(context.Background(), doc, block, nil)
	if !errors.Is(err, ErrNoModel) {
		t.Fatalf("err = %v, want ErrNoModel", err)
	}
}

func TestPreparePersistsEditsAndRenders(t *testing.T) {
	st := newFakeStore()
	doc, block := testDocBlock()

	run, err := testService(t, st, fakeGen{}).Prepare(context.Background(), doc, block,
		map[string]string{"author": "Gene Wolfe", "outside": "dropped"})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"characters": "Ada", "author": "Gene Wolfe", "outline": "a voyage"}
	if !reflect.DeepEqual(st.blockAttrs[block.ID], want) {
		t.Errorf("block attrs = %v, want %v", st.blockAttrs[block.ID], want)
	}
	if !reflect.DeepEqual(st.docAttrs[doc.ID], want) {
		t.Errorf("doc attrs = %v, want %v", st.docAttrs[doc.ID], want)
	}
	if !strings.Contains(run.Prompt, "Gene Wolfe") || !strings.Contains(run.Prompt, "a voyage") {
		t.Errorf("prompt not rendered from merged attrs:\n%s", run.Prompt)
	}
	if run.System == "" {
		t.Error("system prompt is empty")
	}
}

func TestPrepareStepErrors(t *testing.T) {
	doc, block := testDocBlock()

	st := newFakeStore()
	st.replaceErr = errors.New("boom")
	_, err := testService(t, st, fakeGen{}).Prepare(context.Background(), doc, block, nil)
	var runErr *Error
	if !errors.As(err, &runErr) || runErr.Step != StepUpdateBlock {
		t.Fatalf("err = %v, want Error{StepUpdateBlock}", err)
	}

	st = newFakeStore()
	st.mergeErr = errors.New("boom")
	_, err = testService(t, st, fakeGen{}).Prepare(context.Background(), doc, block, nil)
	if !errors.As(err, &runErr) || runErr.Step != StepMergeDocument {
		t.Fatalf("err = %v, want Error{StepMergeDocument}", err)
	}
}

func TestExecuteSavesAndReparses(t *testing.T) {
	st := newFakeStore()
	doc, block := testDocBlock()
	st.block = block
	svc := testService(t, st, fakeGen{output: "<style_analysis>terse</style_analysis><story>the end</story>"})

	run, err := svc.Prepare(context.Background(), doc, block, nil)
	if err != nil {
		t.Fatal(err)
	}
	hydrated, err := svc.Execute(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if hydrated.ID != block.ID {
		t.Errorf("hydrated block id = %d, want %d", hydrated.ID, block.ID)
	}

	if len(st.responses) != 1 {
		t.Fatalf("responses saved = %d, want 1", len(st.responses))
	}
	resp := st.responses[0]
	if resp.BlockID != block.ID || resp.Model != doc.SelectedModel || !strings.Contains(resp.Value, "the end") {
		t.Errorf("saved response = %+v", resp)
	}

	// The reply's top-level tags land in the document's shared attributes.
	if got := st.docAttrs[doc.ID]["story"]; got != "the end" {
		t.Errorf("doc story attr = %q, want %q", got, "the end")
	}
	if got := st.docAttrs[doc.ID]["style_analysis"]; got != "terse" {
		t.Errorf("doc style_analysis attr = %q, want %q", got, "terse")
	}
}

func TestExecuteGenerateError(t *testing.T) {
	st := newFakeStore()
	doc, block := testDocBlock()
	svc := testService(t, st, fakeGen{err: errors.New("provider down")})

	run, err := svc.Prepare(context.Background(), doc, block, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Execute(context.Background(), run)
	var runErr *Error
	if !errors.As(err, &runErr) || runErr.Step != StepGenerate {
		t.Fatalf("err = %v, want Error{StepGenerate}", err)
	}
	if len(st.responses) != 0 {
		t.Errorf("responses saved = %d, want 0", len(st.responses))
	}
}

func TestExecuteStreamForwardsDeltas(t *testing.T) {
	st := newFakeStore()
	doc, block := testDocBlock()
	st.block = block
	svc := testService(t, st, fakeGen{output: "<story>streamed</story>"})

	run, err := svc.Prepare(context.Background(), doc, block, nil)
	if err != nil {
		t.Fatal(err)
	}
	var deltas []string
	if _, err := svc.ExecuteStream(context.Background(), run, func(d string) { deltas = append(deltas, d) }); err != nil {
		t.Fatal(err)
	}
	if len(deltas) != 1 || !strings.Contains(deltas[0], "streamed") {
		t.Errorf("deltas = %v", deltas)
	}
	if got := st.docAttrs[doc.ID]["story"]; got != "streamed" {
		t.Errorf("doc story attr = %q, want %q", got, "streamed")
	}
}

func TestReparseRenamesAndOutputKey(t *testing.T) {
	st := newFakeStore()
	doc, _ := testDocBlock()
	svc := testService(t, st, fakeGen{})

	m, ok := mode.Get("story-edit") // Renames: rewritten_story -> story
	if !ok {
		t.Fatal("story-edit mode missing")
	}
	if err := svc.Reparse(context.Background(), doc, m, "<rewritten_story>better</rewritten_story>"); err != nil {
		t.Fatal(err)
	}
	if got := st.docAttrs[doc.ID]["story"]; got != "better" {
		t.Errorf("doc story attr = %q, want %q", got, "better")
	}
	if _, ok := st.docAttrs[doc.ID]["rewritten_story"]; ok {
		t.Error("rewritten_story should have been renamed away")
	}
}

func TestApplyRenames(t *testing.T) {
	tests := []struct {
		name    string
		updates map[string]string
		renames map[string]string
		want    map[string]string
	}{
		{
			name:    "source present is moved and original dropped",
			updates: map[string]string{"revised_outline": "x", "keep": "y"},
			renames: map[string]string{"revised_outline": "outline"},
			want:    map[string]string{"outline": "x", "keep": "y"},
		},
		{
			name:    "source absent is a no-op",
			updates: map[string]string{"keep": "y"},
			renames: map[string]string{"revised_outline": "outline"},
			want:    map[string]string{"keep": "y"},
		},
		{
			name:    "existing target is overwritten by source",
			updates: map[string]string{"rewritten_story": "new", "story": "old"},
			renames: map[string]string{"rewritten_story": "story"},
			want:    map[string]string{"story": "new"},
		},
		{
			name:    "nil renames is a no-op",
			updates: map[string]string{"a": "b"},
			renames: nil,
			want:    map[string]string{"a": "b"},
		},
		{
			name:    "empty renames is a no-op",
			updates: map[string]string{"a": "b"},
			renames: map[string]string{},
			want:    map[string]string{"a": "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyRenames(tt.updates, tt.renames)
			if !reflect.DeepEqual(tt.updates, tt.want) {
				t.Errorf("applyRenames() = %v, want %v", tt.updates, tt.want)
			}
		})
	}
}
