package mode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cbroglie/mustache"
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

// TestTemplatesDoNotEscapeAttributes guards against mustache's HTML escaping
// mangling attribute text. A "{{key}}" interpolation runs template.HTMLEscape,
// which turns an apostrophe into "&#39;" and a quote into "&#34;"; the raw
// "{{{key}}}" form does not. Nothing here ever renders HTML — the rendered
// prompt goes straight to an LLM — so every slot must use the raw form, or
// character descriptions and outlines reach the model full of entities.
func TestTemplatesDoNotEscapeAttributes(t *testing.T) {
	// Each key gets its own sentinel so one raw slot can't mask an escaped
	// sibling; every character here is one HTMLEscape rewrites.
	sentinel := func(key string) string {
		return `Flannery O'Connor & "` + key + `" <tag> 5 > 3`
	}

	for _, m := range All() {
		attrs := map[string]string{}
		for _, k := range m.Keys {
			attrs[k] = sentinel(k)
		}
		out, err := mustache.Render(m.Template, attrs)
		if err != nil {
			t.Errorf("%s: render: %v", m.Name, err)
			continue
		}
		for _, k := range m.Keys {
			if !strings.Contains(out, sentinel(k)) {
				t.Errorf("%s: key %q was escaped — use {{{%s}}} rather than {{%s}}",
					m.Name, k, k, k)
			}
		}
	}
}
