package handler

import (
	"testing"

	"github.com/farrellm/nisaba/internal/mode"
	"github.com/farrellm/nisaba/internal/model"
)

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
