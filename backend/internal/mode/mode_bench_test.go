package mode

import (
	"strings"
	"testing"

	"github.com/cbroglie/mustache"
)

// benchAttrs builds a realistic attribute map with ~10KB of prose spread over
// the story mode's keys.
func benchAttrs() map[string]string {
	const sentence = "The lighthouse keeper counted the waves as they broke against the rocks below. "
	prose := func(n int) string { return strings.Repeat(sentence, n/len(sentence)+1)[:n] }
	return map[string]string{
		"characters": prose(4 * 1024),
		"author":     "Ursula K. Le Guin",
		"outline":    prose(6 * 1024),
	}
}

func BenchmarkTemplateFor(b *testing.B) {
	m, ok := Get("story")
	if !ok {
		b.Fatal("story mode missing")
	}
	// A safe username with no override dir: measures the os.ReadFile miss that
	// every un-overridden run pays.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		TemplateFor("bench-no-such-user", m)
	}
}

func BenchmarkSystemPrompt(b *testing.B) {
	// Provider-miss → user-miss → embedded fallback chain (two ReadFile misses).
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		SystemPrompt("bench-no-such-user", "anthropic")
	}
}

func BenchmarkRenderStoryTemplate(b *testing.B) {
	m, ok := Get("story")
	if !ok {
		b.Fatal("story mode missing")
	}
	attrs := benchAttrs()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := mustache.Render(m.Template, attrs); err != nil {
			b.Fatal(err)
		}
	}
}
