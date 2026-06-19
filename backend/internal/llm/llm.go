// Package llm wraps the provider-agnostic dragon-born/go-llm library so the rest
// of the app never depends on a specific LLM vendor. It exposes a single fixed,
// cross-provider model list (the source of truth for the UI selector and for
// validating a document's selected model) and a Generate helper.
//
// Requests route through go-llm's default provider (OpenRouter), so a single
// OPENROUTER_API_KEY reaches every model regardless of its upstream provider.
package llm

import (
	"context"

	ai "gopkg.in/dragon-born/go-llm.v1"
)

// systemPrompt keeps the model's reply to just the requested writing — no
// preamble or meta-commentary that would pollute the block response.
const systemPrompt = "You are a writing assistant. Produce only the requested text. Do not add preamble, explanations, or meta-commentary."

// maxToolIterations bounds the agentic tool-call loop in Generate so a model
// that keeps requesting tools can't spin forever.
const maxToolIterations = 5

// ToolDef is a tool/function a mode can attach to its LLM calls. It carries the
// function name, description, a JSON-schema Parameters map, and a Handler that
// executes the call. Aliased to go-llm's type so callers (e.g. internal/mode)
// configure tools without importing the vendor library directly.
type ToolDef = ai.ToolDef

// Params returns a fluent builder for a tool's JSON-schema Parameters map, e.g.
// llm.Params().String("q", "query", true).Build(). Re-exported from go-llm so
// callers stay vendor-agnostic.
var Params = ai.Params

// Model is one selectable model in the fixed list. ID is the go-llm model
// identifier stored in documents.selected_model.
type Model struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Provider string `json:"provider"`
}

// models is the fixed, cross-provider list. IDs come from go-llm's own model
// constants so they can't drift. Edit here to add/remove a model.
var models = []Model{
	{ID: string(ai.ModelClaudeOpus), Label: "Claude Opus 4.5", Provider: "anthropic"},
	{ID: string(ai.ModelClaudeSonnet), Label: "Claude Sonnet 4.5", Provider: "anthropic"},
	{ID: string(ai.ModelClaudeHaiku), Label: "Claude Haiku 4.5", Provider: "anthropic"},
	{ID: string(ai.ModelGPT5), Label: "GPT-5.2", Provider: "openai"},
	{ID: string(ai.ModelGemini3Pro), Label: "Gemini 3 Pro", Provider: "google"},
}

// Models returns the fixed model list in display order.
func Models() []Model {
	return models
}

// Valid reports whether id is one of the fixed models.
func Valid(id string) bool {
	for _, m := range models {
		if m.ID == id {
			return true
		}
	}
	return false
}

// Generate sends prompt to the given model and returns its text reply. The
// model id must be one from Models(); routing/auth is handled by go-llm.
//
// When tools is non-empty, each tool's definition and handler are attached and
// the request runs through go-llm's agentic loop (RunTools): the model may
// invoke tools, whose results are fed back, until it returns a final text reply
// or maxToolIterations is reached. With no tools it uses the plain ask path.
func Generate(ctx context.Context, model, prompt string, tools []ToolDef) (string, error) {
	b := ai.New(ai.Model(model)).System(systemPrompt)
	if len(tools) == 0 {
		return b.Ask(prompt)
	}
	for _, t := range tools {
		b = b.ToolDef(t)
	}
	return b.User(prompt).RunTools(maxToolIterations)
}
