package llm

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider/openai"
)

// blockedNames are substrings that disqualify a generated name. A candidate is
// rejected if any of these appears anywhere in it. Ported verbatim from the
// Python BLOCKED_NAMES list.
var blockedNames = []string{
	"Alistair",
	"Black",
	"Chen",
	"Darian",
	"Elara",
	"Iris",
	"Kael",
	"Lilith",
	"Lira",
	"Marcus",
	"Ozono",
	"Silas",
	"Thorn",
	"Vance",
}

// namesResponse is the structured output shape requested from the model: a flat
// list of candidate names. Mirrors the Python NamesResponse model.
type namesResponse struct {
	Names []string `json:"names"`
}

// generateNames asks the model for ~20 candidate character names for the given
// description, returning them unfiltered. It hard-codes the model per request;
// this is the only vendor-aware part, so it stays in internal/llm.
func generateNames(ctx context.Context, description string) ([]string, error) {
	prompt := fmt.Sprintf("Generate a list of 20 names (first and last) for a "+
		"character with the following description:\n%s", description)

	res, err := goai.GenerateObject[namesResponse](ctx, openai.Chat("gpt-5.4"),
		goai.WithPrompt(prompt),
		goai.WithTemperature(1.0),
	)
	if err != nil {
		return nil, err
	}
	return res.Object.Names, nil
}

// pickAllowed partitions names into those matching the blocklist (any blocked
// substring) and those that don't, then returns a random allowed name. If none
// are allowed it falls back to a random blocked name; if there are no names at
// all it returns an error (the Python original would panic here).
func pickAllowed(names []string) (string, error) {
	var matched, unmatched []string
	for _, s := range names {
		blocked := false
		for _, sub := range blockedNames {
			if strings.Contains(s, sub) {
				blocked = true
				break
			}
		}
		if blocked {
			matched = append(matched, s)
		} else {
			unmatched = append(unmatched, s)
		}
	}

	switch {
	case len(unmatched) > 0:
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(unmatched))))
		if err != nil {
			return "", fmt.Errorf("failed to pick name: %w", err)
		}
		return unmatched[n.Int64()], nil
	case len(matched) > 0:
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(matched))))
		if err != nil {
			return "", fmt.Errorf("failed to pick name: %w", err)
		}
		return matched[n.Int64()], nil
	default:
		return "", fmt.Errorf("no names were generated")
	}
}

// GenerateNameTool is a tool that generates a unique character name for a given
// description. It asks the model for a batch of candidates, drops any containing
// a blocked substring, and returns one allowed name at random. Attach it to a
// mode's Tools to make it callable.
var GenerateNameTool = goai.NewTool(
	"generate_name",
	"Generate a unique name for a character with the specified description.",
	func(ctx context.Context, args struct {
		Description string `json:"description" jsonschema:"description=A detailed description of the character to name."`
	}) (string, error) {
		if strings.TrimSpace(args.Description) == "" {
			return "", fmt.Errorf("generate_name: description is required")
		}

		names, err := generateNames(ctx, args.Description)
		if err != nil {
			return "", err
		}
		return pickAllowed(names)
	},
)
