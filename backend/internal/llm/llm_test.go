package llm

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
)

// TestModelKeys covers the invariants the init() normalization guarantees: every
// entry is addressable by a unique, non-empty key and still carries a
// provider-native ID. It walks the raw list rather than Models() so hidden
// entries are checked too.
func TestModelKeys(t *testing.T) {
	seen := make(map[string]bool)
	for _, m := range models {
		if m.Key == "" {
			t.Errorf("model %q has empty key", m.Label)
		}
		if m.ID == "" {
			t.Errorf("model %q has empty provider id", m.Key)
		}
		if seen[m.Key] {
			t.Errorf("duplicate model key %q", m.Key)
		}
		seen[m.Key] = true
		if !Valid(m.Key) {
			t.Errorf("Valid(%q) = false, want true", m.Key)
		}
	}
}

// swapModels replaces the fixed model list for the duration of one test,
// normalizing the replacement the way init does, and restores the real list
// afterwards. No test here runs in parallel, so the swap is safe.
func swapModels(t *testing.T, ms []Model) {
	t.Helper()
	orig := models
	t.Cleanup(func() { models = orig })
	normalizeModels(ms)
	models = ms
}

// TestModelVariants pins the behavior this indirection exists for: two entries
// sharing one provider-native ID resolve separately and keep their own options.
// It runs against a synthetic list rather than the shipping one — the fixed list
// is product data that turns over as models are added and retired, so pinning
// the mechanism to whichever variant happens to ship today makes the test fail
// on the next model refresh instead of on a real regression.
func TestModelVariants(t *testing.T) {
	const id = "vendor/model-1"
	swapModels(t, []Model{
		{ID: id, Label: "Model 1", Provider: "openrouter"},
		{Key: id + ":max", ID: id, Label: "Model 1 (max)", Provider: "openrouter",
			ProviderOptions: map[string]any{
				"reasoning": map[string]any{"effort": "xhigh"},
			}},
	})

	base, ok := lookup(id)
	if !ok {
		t.Fatalf("lookup(%q) not found", id)
	}
	max, ok := lookup(id + ":max")
	if !ok {
		t.Fatalf("lookup(%q) not found", id+":max")
	}
	// The plain entry declares no Key, so normalization defaults it to the ID;
	// the variant keeps the Key it declared.
	if base.Key != id {
		t.Errorf("base key = %q, want %q", base.Key, id)
	}
	if base.ID != max.ID {
		t.Errorf("variants route to different provider ids: %q vs %q", base.ID, max.ID)
	}
	if base.Label == max.Label {
		t.Errorf("variants share label %q", base.Label)
	}
	if base.ProviderOptions != nil {
		t.Errorf("base ProviderOptions = %v, want nil", base.ProviderOptions)
	}
	reasoning, ok := max.ProviderOptions["reasoning"].(map[string]any)
	if !ok {
		t.Fatalf("max ProviderOptions missing reasoning: %v", max.ProviderOptions)
	}
	if reasoning["effort"] != "xhigh" {
		t.Errorf("max reasoning effort = %v, want xhigh", reasoning["effort"])
	}
	// Both variants stay independently addressable everywhere a key is taken.
	for _, key := range []string{id, id + ":max"} {
		if !Valid(key) {
			t.Errorf("Valid(%q) = false, want true", key)
		}
		if got := ProviderFor(key); got != "openrouter" {
			t.Errorf("ProviderFor(%q) = %q, want %q", key, got, "openrouter")
		}
	}
	if got := len(Models()); got != 2 {
		t.Errorf("Models() returned %d entries, want both variants", got)
	}
}

// TestDuplicateModelKeysPanic pins the other half of normalization: a repeated
// key is a build-time mistake that must fail loudly at startup rather than
// leave the second entry unreachable.
func TestDuplicateModelKeysPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("normalizeModels did not panic on a duplicate key")
		}
	}()
	normalizeModels([]Model{
		{ID: "vendor/model-1", Label: "Model 1", Provider: "openrouter"},
		{ID: "vendor/model-1", Label: "Model 1 again", Provider: "openrouter"},
	})
}

// TestHiddenModel pins what Hidden means: gone from the served list, but still
// fully resolvable so stored keys keep working.
func TestHiddenModel(t *testing.T) {
	const key = "claude-haiku-4-5"
	if !Valid(key) {
		t.Errorf("Valid(%q) = false, want true", key)
	}
	if got := ProviderFor(key); got != "anthropic" {
		t.Errorf("ProviderFor(%q) = %q, want %q", key, got, "anthropic")
	}
	for _, m := range Models() {
		if m.Hidden {
			t.Errorf("Models() includes hidden model %q", m.Key)
		}
		if m.Key == key {
			t.Errorf("Models() includes %q, want it hidden", key)
		}
	}
}

func TestToolBlock(t *testing.T) {
	got := toolBlock("generate_name", `{"gender":"female"}`, "Ada")
	want := "<generate_name>\narguments: {\"gender\":\"female\"}\nresult: Ada\n</generate_name>\n"
	if got != want {
		t.Errorf("toolBlock = %q, want %q", got, want)
	}
}

func TestToolResultText(t *testing.T) {
	if got := toolResultText("Ada", nil); got != "Ada" {
		t.Errorf("success = %q, want %q", got, "Ada")
	}
	if got := toolResultText("partial", errors.New("boom")); got != "error: boom" {
		t.Errorf("error = %q, want %q", got, "error: boom")
	}
	long := strings.Repeat("x", 600)
	got := toolResultText("", errors.New(long))
	want := "error: " + strings.Repeat("x", 500) + "..."
	if got != want {
		t.Errorf("long error not truncated to 500 runes: len = %d", len(got))
	}
}

// TestCombineSteps covers both framings: a step with ordered Content (each
// non-empty thinking block wrapped on its own, in place relative to text, e.g.
// an Opus 5.5 progress update between tool calls) and a step without it
// (aggregate reasoning ahead of text). Tool blocks close each step either way.
func TestCombineSteps(t *testing.T) {
	call := provider.ToolCall{ID: "c1", Name: "generate_name", Input: json.RawMessage(`{"n":1}`)}
	res := &goai.TextResult{Steps: []goai.StepResult{
		{
			Content: []provider.Part{
				{Type: provider.PartReasoning, ProviderOptions: map[string]any{"redactedData": "enc"}},
				{Type: provider.PartReasoning, Text: "Reasoning."},
				{Type: provider.PartReasoning, Text: ""}, // omitted display / empty update
				{Type: provider.PartText, Text: "Picking a name."},
				{Type: provider.PartReasoning, Text: "Calling the tool."},
				{Type: provider.PartToolCall, ToolCallID: "c1", ToolName: "generate_name"},
			},
			Reasoning:   "Reasoning.Calling the tool.",
			Text:        "Picking a name.",
			ToolCalls:   []provider.ToolCall{call},
			ToolResults: []provider.ToolResult{{ToolCallID: "c1", Output: "Ada"}},
		},
		{Reasoning: "Done.", Text: "<name>Ada</name>"},
	}}
	want := "<thinking>\nReasoning.\n</thinking>\n" +
		"Picking a name." +
		"<thinking>\nCalling the tool.\n</thinking>\n" +
		toolBlock("generate_name", `{"n":1}`, "Ada") +
		"<thinking>\nDone.\n</thinking>\n" +
		"<name>Ada</name>"
	if got := combineSteps(res); got != want {
		t.Errorf("combineSteps =\n%s\nwant\n%s", got, want)
	}
}

func TestRefusal(t *testing.T) {
	for _, tc := range []struct {
		name string
		res  *goai.TextResult
		want error
	}{
		{"stop", &goai.TextResult{FinishReason: provider.FinishStop}, nil},
		{"refused", &goai.TextResult{FinishReason: provider.FinishContentFilter}, ErrRefused},
		{"refused mid tool loop", &goai.TextResult{
			FinishReason: provider.FinishStop,
			Steps: []goai.StepResult{
				{FinishReason: provider.FinishToolCalls},
				{FinishReason: provider.FinishContentFilter},
			},
		}, ErrRefused},
	} {
		if got := refusal(tc.res); !errors.Is(got, tc.want) {
			t.Errorf("%s: refusal = %v, want %v", tc.name, got, tc.want)
		}
	}
}
