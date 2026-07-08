package handler

import "testing"

func TestResolveLegacyMode(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"registry passthrough", "story", "story"},
		{"registry passthrough generic", "generic", "generic"},
		// Anansi translations
		{"story-full", "story-full", "story"},
		{"editor-agent", "editor-agent", "story-edit"},
		{"brainstorm-thinking-1", "brainstorm-thinking-1", "brainstorm-tools-1"},
		{"brainstorm-thinking-2", "brainstorm-thinking-2", "brainstorm-tools-2"},
		// Charlotte translations
		{"story-opus", "story-opus", "story"},
		{"story-start", "story-start", "story"},
		{"story-next", "story-next", "story-sequel"},
		{"continue-opus", "continue-opus", "story-sequel"},
		{"typo continue", "contiunue-opus", "story-sequel"},
		{"rewrite", "rewrite", "story-revise"},
		{"brainstorm", "brainstorm", "brainstorm-1"},
		{"brainstorm-c", "brainstorm-c", "brainstorm-creative-2"},
		{"revise-outline", "revise-outline", "revise-outline-1"},
		{"authors-thinking", "authors-thinking", "authors"},
		// no registry/map entry -> generic fallback
		{"image", "image", "generic"},
		{"characters", "characters", "generic"},
		{"unknown", "totally-made-up-mode", "generic"},
		{"empty", "", "generic"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveLegacyMode(c.in); got != c.want {
				t.Fatalf("resolveLegacyMode(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
