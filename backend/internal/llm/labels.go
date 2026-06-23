package llm

import (
	"context"
	_ "embed"
	"regexp"
	"strings"

	"github.com/cbroglie/mustache"
)

//go:embed templates/suggest-labels.mustache
var suggestLabelsTmpl string

//go:embed templates/available-labels.mustache
var availableLabelsTmpl string

// suggestLabelsModel is the model used to suggest story labels. Hard-coded per
// request; like generateNames this is the only vendor-aware part, so it stays in
// internal/llm. It must be an id from the fixed models list (routed by clientFor).
const suggestLabelsModel = "claude-sonnet-4-6"

// labelRe matches the inner text of each <label>…</label> tag. The model emits
// labels nested inside a <suggestion> block, so parseTopLevelTags (which only
// reads top-level tags, and lives in internal/handler) doesn't fit; a focused
// scan is simpler and degrades gracefully on the surrounding scratch reasoning.
var labelRe = regexp.MustCompile(`(?s)<label>(.*?)</label>`)

// SuggestLabels renders the suggest-labels template for story, asks the
// hard-coded model to analyze it, and returns the suggested labels in order.
// Returns a non-nil (possibly empty) slice when the call succeeds.
func SuggestLabels(ctx context.Context, story string) ([]string, error) {
	prompt, err := mustache.Render(suggestLabelsTmpl, map[string]string{"story": story})
	if err != nil {
		return nil, err
	}

	res, err := Generate(ctx, suggestLabelsModel, "", prompt, nil)
	if err != nil {
		return nil, err
	}
	return parseLabels(res), nil
}

// SelectLabels renders the available-labels template for story and the supplied
// pool, asks the hard-coded model which of those labels fit, and returns the
// chosen ones (a subset of available). Returns a non-nil (possibly empty) slice;
// an empty pool short-circuits without an LLM call.
func SelectLabels(ctx context.Context, story string, available []string) ([]string, error) {
	if len(available) == 0 {
		return []string{}, nil
	}

	prompt, err := mustache.Render(availableLabelsTmpl, map[string]string{
		"story":     story,
		"available": strings.Join(available, "\n"),
	})
	if err != nil {
		return nil, err
	}

	res, err := Generate(ctx, suggestLabelsModel, "", prompt, nil)
	if err != nil {
		return nil, err
	}
	// The model is told to only pick from the list, but filter defensively so a
	// stray or reworded label can't leak through.
	return keepAvailable(parseLabels(res), available), nil
}

// keepAvailable returns the members of labels that match a name in available
// (case-insensitively), in their original order, using the canonical casing from
// available and dropping duplicates. Always returns non-nil.
func keepAvailable(labels, available []string) []string {
	kept := []string{}
	for _, l := range labels {
		for _, a := range available {
			if strings.EqualFold(l, a) {
				canonical := a
				dup := false
				for _, k := range kept {
					if strings.EqualFold(k, canonical) {
						dup = true
						break
					}
				}
				if !dup {
					kept = append(kept, canonical)
				}
				break
			}
		}
	}
	return kept
}

// parseLabels extracts the trimmed inner text of every <label> tag in s, in
// order, dropping any that are empty after trimming. Always returns non-nil.
func parseLabels(s string) []string {
	labels := []string{}
	for _, m := range labelRe.FindAllStringSubmatch(s, -1) {
		if label := strings.TrimSpace(m[1]); label != "" {
			labels = append(labels, label)
		}
	}
	return labels
}
