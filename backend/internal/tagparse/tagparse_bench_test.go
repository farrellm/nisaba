package tagparse

import (
	"strings"
	"testing"
)

// benchProse returns roughly n bytes of plain prose with no '<' bytes, so the
// parser's scan loop covers it in one IndexByte hop per tag boundary.
func benchProse(n int) string {
	const sentence = "The lighthouse keeper counted the waves as they broke against the rocks below. "
	return strings.Repeat(sentence, n/len(sentence)+1)[:n]
}

func BenchmarkParseTopLevelTags(b *testing.B) {
	prose50k := benchProse(50 * 1024)
	prose2k := benchProse(2 * 1024)

	var multi strings.Builder
	for i, name := range []string{"story", "characters", "outline", "author", "style_analysis", "edit", "prompt", "sequel_outline"} {
		multi.WriteString("Some connective prose between tags.\n<" + name + ">\n")
		if i%3 == 0 {
			multi.WriteString("<nested attr=\"x\">inner <deeper>text</deeper></nested>\n")
		}
		multi.WriteString(prose2k)
		multi.WriteString("\n</" + name + ">\n")
	}

	cases := []struct {
		name  string
		input string
	}{
		{"story50k", "Preamble text.\n<story>\n" + prose50k + "\n</story>\nTrailing text."},
		{"multiTag", multi.String()},
		{"unclosed", "<story>\n" + prose50k},
		{"noTags", benchProse(10 * 1024)},
		{"repeatedOpen", "<story>\n" + prose2k + "\n<story>\n" + prose2k + "\n</story>"},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(c.input)))
			for i := 0; i < b.N; i++ {
				Parse(c.input)
			}
		})
	}
}
