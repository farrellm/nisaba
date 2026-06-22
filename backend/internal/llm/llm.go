// Package llm wraps the GoAI SDK (github.com/zendev-sh/goai) so the rest of the
// app never depends on a specific LLM vendor. It exposes a single fixed,
// cross-provider model list (the source of truth for the UI selector and for
// validating a document's selected model) and a Generate helper.
//
// Each model routes through the provider named in its Model.Provider — its own
// vendor (Anthropic, OpenAI, Google) or the OpenRouter aggregator. GoAI reads
// each provider's key from the environment: ANTHROPIC_API_KEY, OPENAI_API_KEY,
// GEMINI_API_KEY/GOOGLE_GENERATIVE_AI_API_KEY, and OPENROUTER_API_KEY.
package llm

import (
	"context"
	"fmt"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/anthropic"
	"github.com/zendev-sh/goai/provider/google"
	"github.com/zendev-sh/goai/provider/openai"
	"github.com/zendev-sh/goai/provider/openrouter"
)

// maxToolIterations bounds the agentic tool-call loop in Generate so a model
// that keeps requesting tools can't spin forever.
const maxToolIterations = 5

// Tool is a tool/function a mode can attach to its LLM calls. Aliased to GoAI's
// type so callers (e.g. internal/mode) configure tools without importing the
// vendor library directly; build one with goai.NewTool (see names.go).
type Tool = goai.Tool

// Model is one selectable model in the fixed list. ID is the provider-native
// model identifier stored in documents.selected_model; Provider selects which
// GoAI provider client routes the request.
type Model struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Provider string `json:"provider"`
}

// models is the fixed, cross-provider list. IDs are provider-native model names.
// Edit here to add/remove a model; Provider must be one clientFor understands.
var models = []Model{
	{ID: "claude-opus-4-5", Label: "Claude Opus 4.5", Provider: "anthropic"},
	{ID: "claude-sonnet-4-5", Label: "Claude Sonnet 4.5", Provider: "anthropic"},
	{ID: "claude-haiku-4-5", Label: "Claude Haiku 4.5", Provider: "anthropic"},
	{ID: "gpt-5.2", Label: "GPT-5.2", Provider: "openai"},
	{ID: "gemini-3-pro", Label: "Gemini 3 Pro", Provider: "google"},
	{ID: "z-ai/glm-5.2", Label: "GLM-5.2", Provider: "openrouter"},
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

// clientFor returns the GoAI provider client for a model id from the fixed list,
// routing to the provider named in its Model.Provider. Unknown ids error.
func clientFor(id string) (provider.LanguageModel, error) {
	for _, m := range models {
		if m.ID != id {
			continue
		}
		switch m.Provider {
		case "anthropic":
			return anthropic.Chat(id), nil
		case "openai":
			return openai.Chat(id), nil
		case "google":
			return google.Chat(id), nil
		case "openrouter":
			return openrouter.Chat(id), nil
		default:
			return nil, fmt.Errorf("model %q has unsupported provider %q", id, m.Provider)
		}
	}
	return nil, fmt.Errorf("unknown model %q", id)
}

// Generate sends prompt to the given model under the given system prompt and
// returns its text reply. The model id must be one from Models(); it routes
// directly to that model's provider.
//
// When tools is non-empty, each tool is attached and the request runs through
// GoAI's agentic loop (MaxSteps): the model may invoke tools, whose results are
// fed back, until it returns a final text reply or maxToolIterations is reached.
// With no tools it does a single generation.
func Generate(ctx context.Context, model, system, prompt string, tools []Tool) (string, error) {
	client, err := clientFor(model)
	if err != nil {
		return "", err
	}

	opts := []goai.Option{goai.WithSystem(system), goai.WithPrompt(prompt)}
	if len(tools) > 0 {
		opts = append(opts, goai.WithTools(tools...), goai.WithMaxSteps(maxToolIterations))
	}

	res, err := goai.GenerateText(ctx, client, opts...)
	if err != nil {
		return "", err
	}
	return res.Text, nil
}
