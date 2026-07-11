package handler

import (
	"strings"
	"testing"

	"github.com/farrellm/nisaba/internal/mode"
	"github.com/farrellm/nisaba/internal/model"
)

// benchProse returns roughly n bytes of plain prose.
func benchProse(n int) string {
	const sentence = "The lighthouse keeper counted the waves as they broke against the rocks below. "
	return strings.Repeat(sentence, n/len(sentence)+1)[:n]
}

func BenchmarkApplyRenames(b *testing.B) {
	renames := map[string]string{"revised_outline": "outline", "rewritten_story": "story"}
	base := map[string]string{
		"revised_outline": benchProse(2048),
		"characters":      benchProse(1024),
		"author":          "Ursula K. Le Guin",
		"style_analysis":  benchProse(1024),
		"prompt":          benchProse(512),
		"edit":            benchProse(256),
		"sequel_outline":  benchProse(1024),
		"story":           benchProse(4096),
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		updates := make(map[string]string, len(base))
		for k, v := range base {
			updates[k] = v
		}
		applyRenames(updates, renames)
	}
}

func BenchmarkMergedBlockAttrs(b *testing.B) {
	m, ok := mode.Get("story-sequel")
	if !ok {
		b.Fatal("story-sequel mode missing")
	}
	block := model.Block{
		ID:   1,
		Mode: m.Name,
		Attributes: map[string]string{
			"story":          benchProse(8 * 1024),
			"characters":     benchProse(1024),
			"author":         "Ursula K. Le Guin",
			"style_analysis": benchProse(1024),
			"sequel_outline": benchProse(2048),
		},
	}
	body := map[string]string{
		"sequel_outline": benchProse(2048),
		"author":         "Gene Wolfe",
		"ignored_key":    "dropped because outside the mode's key set",
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mergedBlockAttrs(block, m, body)
	}
}
