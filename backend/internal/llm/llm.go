// Package llm wraps the GoAI SDK (github.com/zendev-sh/goai) so the rest of the
// app never depends on a specific LLM vendor. It exposes a single fixed,
// cross-provider model list (the source of truth for the UI selector and for
// validating a document's selected model) and a Generate helper.
//
// Each model routes through the provider named in its Model.Provider — its own
// vendor (Anthropic, OpenAI, Google) or the OpenRouter aggregator. GoAI reads
// each provider's key from the environment: ANTHROPIC_API_KEY, OPENAI_API_KEY,
// GEMINI_API_KEY/GOOGLE_GENERATIVE_AI_API_KEY, OPENROUTER_API_KEY, and
// DEEPSEEK_API_KEY.
package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/anthropic"
	"github.com/zendev-sh/goai/provider/deepseek"
	"github.com/zendev-sh/goai/provider/google"
	"github.com/zendev-sh/goai/provider/openai"
	"github.com/zendev-sh/goai/provider/openrouter"
)

// maxToolIterations bounds the agentic tool-call loop in Generate so a model
// that keeps requesting tools can't spin forever.
const maxToolIterations = 5

// defaultMaxTokens caps output length when a model sets no MaxTokens override.
const defaultMaxTokens = 64000

// Tool is a tool/function a mode can attach to its LLM calls. Aliased to GoAI's
// type so callers (e.g. internal/mode) configure tools without importing the
// vendor library directly; build one with goai.NewTool (see names.go).
type Tool = goai.Tool

// Model is one selectable model in the fixed list. ID is the provider-native
// model identifier stored in documents.selected_model; Provider selects which
// GoAI provider client routes the request.
type Model struct {
	ID              string         `json:"id"`
	Label           string         `json:"label"`
	Provider        string         `json:"provider"`
	ProviderOptions map[string]any `json:"-"`
	// ToolProviderOptions are provider options applied only when the call carries
	// tools (merged over ProviderOptions).
	ToolProviderOptions map[string]any `json:"-"`
	// MaxTokens overrides the default output-token cap for this model; 0 uses defaultMaxTokens.
	MaxTokens int `json:"-"`
}

// Shared Anthropic provider options. anthropicThinking enables adaptive thinking
// with summarized output; anthropicCaching sets ephemeral cache_control on
// tool-bearing calls. Treated as read-only (generate copies them out, never
// mutates), so the same map may back multiple models.
var (
	anthropicThinking = map[string]any{
		"thinking": map[string]any{
			"type":    "adaptive",
			"display": "summarized",
		},
	}
	anthropicCaching = map[string]any{
		"cache_control": map[string]any{"type": "ephemeral"},
	}
	openaiReasoning = map[string]any{
		"reasoning_summary": "auto",
	}
)

// models is the fixed, cross-provider list. IDs are provider-native model names.
// Edit here to add/remove a model; Provider must be one clientFor understands.
var models = []Model{
	{ID: "claude-haiku-4-5", Label: "Claude Haiku 4.5", Provider: "anthropic",
		ToolProviderOptions: anthropicCaching},
	{ID: "claude-sonnet-5", Label: "Claude Sonnet 5", Provider: "anthropic",
		ProviderOptions: anthropicThinking, ToolProviderOptions: anthropicCaching},
	{ID: "claude-opus-4-8", Label: "Claude Opus 4.8", Provider: "anthropic",
		ProviderOptions: anthropicThinking, ToolProviderOptions: anthropicCaching},
	{ID: "claude-fable-5", Label: "Claude Fable 5", Provider: "anthropic",
		ToolProviderOptions: anthropicCaching},
	{ID: "gpt-5.6-terra", Label: "GPT-5.6 Terra", Provider: "openai",
		ProviderOptions: openaiReasoning},
	{ID: "gpt-5.6-sol", Label: "GPT-5.6 Sol", Provider: "openai",
		ProviderOptions: openaiReasoning},
	{ID: "gemini-3.5-flash", Label: "Gemini 3.5 Flash", Provider: "google"},
	{ID: "gemini-3.1-pro-preview", Label: "Gemini 3.1 Pro", Provider: "google"},
	{ID: "z-ai/glm-5.2", Label: "GLM 5.2", Provider: "openrouter"},
	{ID: "x-ai/grok-4.5", Label: "Grok 4.5", Provider: "openrouter"},
	{ID: "deepseek-v4-pro", Label: "DeepSeek V4 Pro", Provider: "deepseek"},
}

// Models returns the fixed model list in display order.
func Models() []Model {
	return models
}

// lookup returns the Model with the given id from the fixed list.
func lookup(id string) (Model, bool) {
	for _, m := range models {
		if m.ID == id {
			return m, true
		}
	}
	return Model{}, false
}

// Valid reports whether id is one of the fixed models.
func Valid(id string) bool {
	_, ok := lookup(id)
	return ok
}

// ProviderFor returns the provider name for a model id from the fixed list
// (e.g. "anthropic"), or "" if the id is unknown.
func ProviderFor(id string) string {
	m, ok := lookup(id)
	if !ok {
		return ""
	}
	return m.Provider
}

// clientFor returns the GoAI provider client for a model id from the fixed list,
// routing to the provider named in its Model.Provider. Unknown ids error.
func clientFor(id string) (provider.LanguageModel, error) {
	m, ok := lookup(id)
	if !ok {
		return nil, fmt.Errorf("unknown model %q", id)
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
	case "deepseek":
		return deepseek.Chat(id), nil
	default:
		return nil, fmt.Errorf("model %q has unsupported provider %q", id, m.Provider)
	}
}

// generate runs the GoAI call for one model and returns the raw result. Callers
// choose between the combined per-step output (Generate) and final-text-only
// (the label helpers in labels.go). The model id must be one from Models(); it
// routes directly to that model's provider.
//
// When tools is non-empty, each tool is attached and the request runs through
// GoAI's agentic loop (MaxSteps): the model may invoke tools, whose results are
// fed back, until it returns a final text reply or maxToolIterations is reached.
// That path also merges the model's ToolProviderOptions over its ProviderOptions,
// which is how cache control (Anthropic cache_control) is enabled per-model so the
// multi-step loop doesn't pay to re-send the prompt each step. With no tools it
// does a single generation.
// buildCall resolves the provider client and assembles the GoAI options shared
// by the buffered (generate) and streaming (GenerateStream) paths: system +
// prompt, plus the model's provider options, plus the tool loop (tools,
// max-steps, tool-only provider options) when tools are attached.
func buildCall(model, system, prompt string, tools []Tool) (provider.LanguageModel, []goai.Option, error) {
	client, err := clientFor(model)
	if err != nil {
		return nil, nil, err
	}
	m, _ := lookup(model) // unknown id already rejected by clientFor above

	maxTokens := defaultMaxTokens
	if m.MaxTokens > 0 {
		maxTokens = m.MaxTokens
	}

	// Log every model API call in one place: the hook fires in all paths
	// (buffered + streaming, success + error, once per tool-loop step), so the
	// finish reason is recorded even when the call errors.
	logResponse := func(info goai.ResponseInfo) {
		u := info.Usage
		slog.Info("llm generate",
			"model", model,
			"finish_reason", info.FinishReason,
			"error", info.Error,
			"status", info.StatusCode,
			"input_tokens", u.InputTokens,
			"output_tokens", u.OutputTokens,
			"total_tokens", u.TotalTokens,
			"reasoning_tokens", u.ReasoningTokens,
			"cache_read_tokens", u.CacheReadTokens,
			"cache_write_tokens", u.CacheWriteTokens,
		)
	}

	opts := []goai.Option{
		goai.WithSystem(system),
		goai.WithPrompt(prompt),
		goai.WithMaxOutputTokens(maxTokens),
		goai.WithOnResponse(logResponse),
	}

	provOpts := map[string]any{}
	for k, v := range m.ProviderOptions {
		provOpts[k] = v
	}
	if len(tools) > 0 {
		opts = append(opts,
			goai.WithTools(tools...),
			goai.WithMaxSteps(maxToolIterations),
		)
		for k, v := range m.ToolProviderOptions {
			provOpts[k] = v
		}
	}
	if len(provOpts) > 0 {
		opts = append(opts, goai.WithProviderOptions(provOpts))
	}
	return client, opts, nil
}

func generate(ctx context.Context, model, system, prompt string, tools []Tool) (*goai.TextResult, error) {
	client, opts, err := buildCall(model, system, prompt, tools)
	if err != nil {
		return nil, err
	}

	res, err := goai.GenerateText(ctx, client, opts...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Generate sends prompt to the given model under the given system prompt and
// returns its reply as the combined per-step output (see combineSteps): each
// step's thinking, when present, wrapped in <thinking>…</thinking> ahead of its
// text, concatenated across a multi-step tool loop.
func Generate(ctx context.Context, model, system, prompt string, tools []Tool) (string, error) {
	res, err := generate(ctx, model, system, prompt, tools)
	if err != nil {
		return "", err
	}
	return combineSteps(res), nil
}

// thinkingOpen/thinkingClose frame a step's reasoning text. Shared by
// combineSteps (the final, persisted value) and GenerateStream (the live
// preview) so the two stay in sync.
const (
	thinkingOpen  = "<thinking>\n"
	thinkingClose = "\n</thinking>\n"
)

// DeltaKind classifies a streamed delta for onDelta consumers: ordinary model
// output (text/thinking) vs. a completed tool-call block. The frontend uses
// the tool kind to refresh a non-streamed run's preview at tool boundaries.
type DeltaKind string

const (
	DeltaKindText DeltaKind = "text"
	DeltaKindTool DeltaKind = "tool"
)

// toolBlock frames one tool call as a <toolname> block carrying its arguments
// and result. Shared by combineSteps and GenerateStream so the live preview
// and the persisted value can't drift.
func toolBlock(name, input, result string) string {
	return "<" + name + ">\narguments: " + input + "\nresult: " + result + "\n</" + name + ">\n"
}

// toolResultText renders a tool execution's outcome the way GoAI renders it
// into ToolResult.Output (buildToolResults): the error text, truncated to 500
// runes, when the tool failed; the raw output otherwise.
func toolResultText(output string, err error) string {
	if err == nil {
		return output
	}
	errStr := err.Error()
	if runes := []rune(errStr); len(runes) > 500 {
		errStr = string(runes[:500]) + "..."
	}
	return "error: " + errStr
}

// GenerateStream is the streaming sibling of Generate. It pushes the model's
// reply to onDelta as it arrives and returns the same combined per-step output
// Generate would (via combineSteps on the final result), so the value persisted
// by the caller is identical whether or not streaming was used.
//
// It consumes GoAI's raw chunk stream (ts.Stream()) so reasoning streams live,
// framed exactly like combineSteps: reasoning deltas are wrapped in
// <thinking>…</thinking> (opened on the first reasoning token of a run, closed
// when text or a tool call follows) and text deltas are emitted as-is, both
// with DeltaKindText. GoAI still runs the automatic tool loop (WithMaxSteps)
// and executes tools itself; tool *results* never reach the raw chunk stream
// (GoAI consumes its internal tool-results chunk), so each call is instead
// emitted the moment its tool finishes executing — via the OnToolCall hook —
// as one atomic <toolname>…</toolname> block carrying arguments and result,
// with DeltaKindTool. The hook fires from GoAI's tool goroutines, concurrent
// with this stream loop, so all emission is serialized under one mutex.
func GenerateStream(ctx context.Context, model, system, prompt string, tools []Tool, onDelta func(DeltaKind, string)) (string, error) {
	client, opts, err := buildCall(model, system, prompt, tools)
	if err != nil {
		return "", err
	}

	// mu guards emission and the tool-call bookkeeping below; emit and
	// closeThinking must be called with it held.
	var mu sync.Mutex
	emit := func(kind DeltaKind, s string) {
		if onDelta != nil && s != "" {
			onDelta(kind, s)
		}
	}
	inThinking := false
	closeThinking := func() {
		if inThinking {
			emit(DeltaKindText, thinkingClose)
			inThinking = false
		}
	}

	// Calls seen on the stream whose result hasn't been emitted yet, in call
	// order, plus the IDs already emitted by the hook — the chunk channel is
	// buffered, so a fast tool can resolve before its ChunkToolCall is read.
	type toolCall struct{ id, name, input string }
	var pending []toolCall
	resolved := map[string]bool{}

	// Sequential execution makes the hook fire in call order, so the preview's
	// tool blocks land in the same order combineSteps persists them (parallel
	// tools resolve in completion order and the two can disagree).
	opts = append(opts, goai.WithSequentialToolExecution(), goai.WithOnToolCall(func(info goai.ToolCallInfo) {
		mu.Lock()
		defer mu.Unlock()
		closeThinking()
		emit(DeltaKindTool, toolBlock(info.ToolName, string(info.Input), toolResultText(info.Output, info.Error)))
		resolved[info.ToolCallID] = true
		for i, c := range pending {
			if c.id == info.ToolCallID {
				pending = append(pending[:i], pending[i+1:]...)
				break
			}
		}
	}))

	ts, err := goai.StreamText(ctx, client, opts...)
	if err != nil {
		return "", err
	}

	for chunk := range ts.Stream() {
		mu.Lock()
		switch chunk.Type {
		case provider.ChunkReasoning:
			if chunk.Text != "" { // trailing signature/metadata chunk carries no text
				if !inThinking {
					emit(DeltaKindText, thinkingOpen)
					inThinking = true
				}
				emit(DeltaKindText, chunk.Text)
			}
		case provider.ChunkText:
			closeThinking()
			emit(DeltaKindText, chunk.Text)
		case provider.ChunkToolCall:
			// The block itself is emitted by the OnToolCall hook above; just
			// note the call so one that never resolves (a tool without an
			// Execute, a loop cut short) still gets flushed after the stream.
			closeThinking()
			if !resolved[chunk.ToolCallID] {
				pending = append(pending, toolCall{id: chunk.ToolCallID, name: chunk.ToolName, input: chunk.ToolInput})
			}
		}
		mu.Unlock()
	}

	mu.Lock()
	closeThinking()
	for _, c := range pending {
		emit(DeltaKindTool, toolBlock(c.name, c.input, ""))
	}
	mu.Unlock()

	if err := ts.Err(); err != nil {
		return "", err
	}
	return combineSteps(ts.Result()), nil
}

// combineSteps joins every generation step's output in order. For each step the
// thinking (when present) is wrapped as "<thinking>\n…\n</thinking>\n", followed
// by the step's text, followed by one block per tool call tagged with the tool
// name and carrying its arguments and result. Across a multi-step tool loop the
// per-step outputs are concatenated.
func combineSteps(res *goai.TextResult) string {
	var b strings.Builder
	for _, s := range res.Steps {
		if s.Reasoning != "" {
			b.WriteString(thinkingOpen)
			b.WriteString(s.Reasoning)
			b.WriteString(thinkingClose)
		}
		b.WriteString(s.Text)
		// Index this step's results by call ID so each call pairs with its own
		// output; positional pairing breaks when the loop is cut short before
		// some tools run (ToolResults is then empty).
		results := make(map[string]string, len(s.ToolResults))
		for _, r := range s.ToolResults {
			results[r.ToolCallID] = r.Output
		}
		for _, c := range s.ToolCalls {
			b.WriteString(toolBlock(c.Name, string(c.Input), results[c.ID]))
		}
	}
	return b.String()
}
