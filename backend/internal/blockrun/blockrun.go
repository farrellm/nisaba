// Package blockrun orchestrates a block run: persist the caller's edits,
// render the prompt and system prompt (honoring per-user template overrides),
// call the model, save the response, and feed the parsed result back into the
// document's shared attributes.
package blockrun

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cbroglie/mustache"

	"github.com/farrellm/nisaba/internal/llm"
	"github.com/farrellm/nisaba/internal/mode"
	"github.com/farrellm/nisaba/internal/model"
	"github.com/farrellm/nisaba/internal/tagparse"
)

// maxRunDuration bounds a detached model call so a hung provider can't leak
// the goroutine serving it.
const maxRunDuration = 15 * time.Minute

// Sentinel errors for run preconditions.
var (
	// ErrUnknownMode means the block names a mode missing from the registry.
	ErrUnknownMode = errors.New("blockrun: unknown mode")
	// ErrNoModel means the document has no model selected.
	ErrNoModel = errors.New("blockrun: no model selected")
)

// Step names the stage of a run that failed, so callers can map each stage
// onto its own response.
type Step int

const (
	StepUpdateBlock Step = iota
	StepMergeDocument
	StepRenderPrompt
	StepRenderSystem
	StepGenerate
	StepSave
)

// Error wraps a failure with the run step it happened in.
type Error struct {
	Step Step
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("blockrun: step %d: %v", e.Step, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Store is the consumer-side view of the data layer a run needs.
type Store interface {
	ReplaceBlockAttributes(ctx context.Context, blockID int64, attrs map[string]string) error
	MergeDocumentAttributes(ctx context.Context, documentID int64, attrs map[string]string) error
	UpdateDocument(ctx context.Context, doc model.Document) (model.Document, error)
	GetUser(ctx context.Context, id int64) (model.User, error)
	CreateResponse(ctx context.Context, resp model.Response) (model.Response, error)
	GetBlock(ctx context.Context, id int64) (model.Block, error)
}

// Generator abstracts the LLM call (internal/llm) for tests.
type Generator interface {
	Generate(ctx context.Context, model, system, prompt string, tools []llm.Tool) (string, error)
	GenerateStream(ctx context.Context, model, system, prompt string, tools []llm.Tool, onDelta func(llm.DeltaKind, string)) (string, error)
}

// LLM is the default Generator, backed by internal/llm.
type LLM struct{}

func (LLM) Generate(ctx context.Context, model, system, prompt string, tools []llm.Tool) (string, error) {
	return llm.Generate(ctx, model, system, prompt, tools)
}

func (LLM) GenerateStream(ctx context.Context, model, system, prompt string, tools []llm.Tool, onDelta func(llm.DeltaKind, string)) (string, error) {
	return llm.GenerateStream(ctx, model, system, prompt, tools, onDelta)
}

// Service runs blocks. Construct with New.
type Service struct {
	store     Store
	gen       Generator
	templates *mode.Templates
}

// New builds a Service over the given store, generator, and template resolver.
func New(store Store, gen Generator, templates *mode.Templates) *Service {
	return &Service{store: store, gen: gen, templates: templates}
}

// Run is a prepared, renderable unit of work: the resolved mode plus the
// rendered prompt and system prompt, ready to Execute.
type Run struct {
	Doc    model.Document
	Block  model.Block
	Mode   mode.Mode
	Prompt string
	System string
}

// MergedAttrs builds the block's attribute map for the mode's fixed key set,
// taking each key from edits when present and otherwise from the block's
// existing value. Keys outside the mode's key set are dropped, so the result
// always matches the mode.
func MergedAttrs(block model.Block, m mode.Mode, edits map[string]string) map[string]string {
	attrs := make(map[string]string, len(m.Keys))
	for _, key := range m.Keys {
		if v, present := edits[key]; present {
			attrs[key] = v
		} else {
			attrs[key] = block.Attributes[key]
		}
	}
	return attrs
}

// Prepare performs the shared setup for a block run: it resolves the mode and
// selected model, persists the caller's attribute edits into both the block
// and the document, and renders the prompt + system prompt (honoring per-user
// template overrides).
func (s *Service) Prepare(ctx context.Context, doc model.Document, block model.Block, edits map[string]string) (Run, error) {
	m, ok := mode.Get(block.Mode)
	if !ok {
		slog.Warn("run failed: unknown mode",
			"user", doc.UserID, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		return Run{}, ErrUnknownMode
	}

	if doc.SelectedModel == "" {
		slog.Warn("run failed: no model selected",
			"user", doc.UserID, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		return Run{}, ErrNoModel
	}

	// Save the caller's edits (absent keys fall back to existing values) and
	// promote them into the document's shared attributes before running.
	attrs := MergedAttrs(block, m, edits)
	if err := s.store.ReplaceBlockAttributes(ctx, block.ID, attrs); err != nil {
		slog.Error("run failed: update block attributes",
			"err", err, "user", doc.UserID, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		return Run{}, &Error{Step: StepUpdateBlock, Err: err}
	}
	if err := s.store.MergeDocumentAttributes(ctx, doc.ID, attrs); err != nil {
		slog.Error("run failed: merge document attributes",
			"err", err, "user", doc.UserID, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		return Run{}, &Error{Step: StepMergeDocument, Err: err}
	}

	// Resolve a per-user template override (falls back to the embedded
	// default when the user has none); a lookup failure degrades gracefully.
	username := ""
	if u, err := s.store.GetUser(ctx, doc.UserID); err == nil {
		username = u.Username
	}

	prompt, err := mustache.Render(s.templates.ModeTemplate(username, m), attrs)
	if err != nil {
		slog.Error("run failed: render prompt",
			"err", err, "user", username, "doc", doc.ID, "block", block.ID, "mode", block.Mode)
		return Run{}, &Error{Step: StepRenderPrompt, Err: err}
	}

	provider := llm.ProviderFor(doc.SelectedModel)
	systemTmpl, systemSource := s.templates.SystemPrompt(username, provider)
	slog.Info("system prompt",
		"user", username, "provider", provider,
		"model", doc.SelectedModel, "source", systemSource)
	system, err := mustache.Render(systemTmpl, attrs)
	if err != nil {
		slog.Error("run failed: render system prompt",
			"err", err, "user", username, "provider", provider,
			"doc", doc.ID, "block", block.ID, "mode", block.Mode, "model", doc.SelectedModel)
		return Run{}, &Error{Step: StepRenderSystem, Err: err}
	}

	return Run{Doc: doc, Block: block, Mode: m, Prompt: prompt, System: system}, nil
}

// detach unbinds the model call + save from the client connection so a
// mid-run disconnect (e.g. an nginx proxy_read_timeout) can't discard
// finished work: the run completes and the response is saved even if the
// browser is gone. Bounded so a hung provider call can't leak the goroutine.
func detach(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), maxRunDuration)
}

// Execute runs the model over a prepared Run, saves the reply as a response,
// feeds it back into the document, and returns the freshly hydrated block.
func (s *Service) Execute(ctx context.Context, run Run) (model.Block, error) {
	ctx, cancel := detach(ctx)
	defer cancel()

	output, err := s.gen.Generate(ctx, run.Doc.SelectedModel, run.System, run.Prompt, run.Mode.Tools)
	if err != nil {
		s.logRunError("run failed: model request", err, run)
		return model.Block{}, &Error{Step: StepGenerate, Err: err}
	}
	return s.finish(ctx, run, output, "")
}

// ExecuteStream is the streaming sibling of Execute: the model's reply is
// forwarded to onDelta as it arrives, then persisted exactly like Execute.
func (s *Service) ExecuteStream(ctx context.Context, run Run, onDelta func(llm.DeltaKind, string)) (model.Block, error) {
	ctx, cancel := detach(ctx)
	defer cancel()

	output, err := s.gen.GenerateStream(ctx, run.Doc.SelectedModel, run.System, run.Prompt, run.Mode.Tools, onDelta)
	if err != nil {
		s.logRunError("run failed: model request (stream)", err, run)
		return model.Block{}, &Error{Step: StepGenerate, Err: err}
	}
	return s.finish(ctx, run, output, " (stream)")
}

// finish persists a completed model reply and feeds it back into the
// document: it stores the response, reparses it into the document's shared
// attributes, and returns the freshly hydrated block.
func (s *Service) finish(ctx context.Context, run Run, output, logSuffix string) (model.Block, error) {
	hydrated, err := s.save(ctx, run, output)
	if err != nil {
		s.logRunError("run failed: save response"+logSuffix, err, run)
		return model.Block{}, &Error{Step: StepSave, Err: err}
	}
	return hydrated, nil
}

func (s *Service) save(ctx context.Context, run Run, output string) (model.Block, error) {
	if _, err := s.store.CreateResponse(ctx, model.Response{
		BlockID:  run.Block.ID,
		Value:    output,
		Model:    run.Doc.SelectedModel,
		Position: len(run.Block.Responses),
	}); err != nil {
		return model.Block{}, err
	}

	// Feed the result back into the document's shared key/values.
	if err := s.Reparse(ctx, run.Doc, run.Mode, output); err != nil {
		return model.Block{}, err
	}

	return s.store.GetBlock(ctx, run.Block.ID)
}

func (s *Service) logRunError(msg string, err error, run Run) {
	slog.Error(msg,
		"err", err, "user", run.Doc.UserID, "doc", run.Doc.ID,
		"block", run.Block.ID, "mode", run.Block.Mode, "model", run.Doc.SelectedModel)
}

// Reparse re-derives a document's shared attributes from a response's text:
// top-level XML tags each populate an attribute (any tag name; nested tags stay
// verbatim in the value), the mode's renames map produced tags onto their
// destination keys, the mode's output key (when set) wins over a same-named
// tag, and the merged result is written back to the document.
func (s *Service) Reparse(ctx context.Context, doc model.Document, m mode.Mode, output string) error {
	updates := tagparse.Parse(output)
	applyRenames(updates, m.Renames)
	if m.Output != "" {
		updates[m.Output] = output
	}
	if len(updates) > 0 {
		if err := s.store.MergeDocumentAttributes(ctx, doc.ID, updates); err != nil {
			return err
		}
		if _, err := s.store.UpdateDocument(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

// applyRenames rewrites keys in updates according to renames (from -> to), using
// move semantics: when a "from" key is present its value is reassigned to "to"
// (overwriting any existing "to") and the original "from" key is removed. Keys
// not named in renames are untouched.
func applyRenames(updates map[string]string, renames map[string]string) {
	for from, to := range renames {
		if v, ok := updates[from]; ok {
			updates[to] = v
			delete(updates, from)
		}
	}
}
