package mode

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSystemPromptResolution exercises the provider → per-user → embedded
// fallback chain for the system prompt override.
func TestSystemPromptResolution(t *testing.T) {
	const user = "tester"

	base := filepath.Join(t.TempDir(), "templates")
	userDir := base + "-" + user
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}

	templates := NewTemplates(base)

	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(userDir, name+".mustache"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	check := func(label, gotTmpl, gotSource, wantTmpl, wantSource string) {
		t.Helper()
		if gotTmpl != wantTmpl {
			t.Errorf("%s: template got %q, want %q", label, gotTmpl, wantTmpl)
		}
		if gotSource != wantSource {
			t.Errorf("%s: source got %q, want %q", label, gotSource, wantSource)
		}
	}

	// No overrides yet: embedded default regardless of provider.
	tmpl, source := templates.SystemPrompt(user, "anthropic")
	check("no override", tmpl, source, systemTmpl, "default")

	// Plain per-user override wins over the embedded default.
	write("system", "plain override")
	tmpl, source = templates.SystemPrompt(user, "anthropic")
	check("plain override", tmpl, source, "plain override", "system.mustache")
	// Empty provider still resolves to the plain override.
	tmpl, source = templates.SystemPrompt(user, "")
	check("empty provider", tmpl, source, "plain override", "system.mustache")

	// Per-provider override wins over the plain per-user override.
	write("system-anthropic", "anthropic override")
	tmpl, source = templates.SystemPrompt(user, "anthropic")
	check("provider override", tmpl, source, "anthropic override", "system-anthropic.mustache")
	// A different provider falls back to the plain override.
	tmpl, source = templates.SystemPrompt(user, "openai")
	check("other provider fallback", tmpl, source, "plain override", "system.mustache")

	// A non-safe username never resolves an override.
	tmpl, source = templates.SystemPrompt("../evil", "anthropic")
	check("unsafe username", tmpl, source, systemTmpl, "default")
}
