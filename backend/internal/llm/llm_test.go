package llm

import (
	"errors"
	"strings"
	"testing"
)

// TestModelKeys covers the invariants the init() normalization guarantees: every
// entry is addressable by a unique, non-empty key and still carries a
// provider-native ID.
func TestModelKeys(t *testing.T) {
	seen := make(map[string]bool)
	for _, m := range Models() {
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

// TestModelVariants pins the behavior this indirection exists for: two entries
// sharing one provider-native ID resolve separately and keep their own options.
func TestModelVariants(t *testing.T) {
	base, ok := lookup("z-ai/glm-5.2")
	if !ok {
		t.Fatal(`lookup("z-ai/glm-5.2") not found`)
	}
	max, ok := lookup("z-ai/glm-5.2:max")
	if !ok {
		t.Fatal(`lookup("z-ai/glm-5.2:max") not found`)
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
