package llm

import (
	"fmt"
	"math/rand"
	"strings"

	ai "gopkg.in/dragon-born/go-llm.v1"
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
func generateNames(description string) ([]string, error) {
	prompt := fmt.Sprintf("Generate a list of 20 names (first and last) for a "+
		"character with the following description:\n%s", description)

	var out namesResponse
	if err := ai.OpenAI().Use("gpt-5.4").
		Temperature(1.0).
		Schema(&out).
		AskInto(prompt, &out); err != nil {
		return nil, err
	}
	return out.Names, nil
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
		return unmatched[rand.Intn(len(unmatched))], nil
	case len(matched) > 0:
		return matched[rand.Intn(len(matched))], nil
	default:
		return "", fmt.Errorf("no names were generated")
	}
}

// GenerateNameTool is a tool that generates a unique character name for a given
// description. It asks the model for a batch of candidates, drops any containing
// a blocked substring, and returns one allowed name at random. Attach it to a
// mode's Tools to make it callable.
var GenerateNameTool = ToolDef{
	Name:        "generate_name",
	Description: "Generate a unique name for a character with the specified description.",
	Parameters: Params().
		String("description", "A detailed description of the character to name.", true).
		Build(),
	Handler: func(args map[string]any) (string, error) {
		description, _ := args["description"].(string)
		if strings.TrimSpace(description) == "" {
			return "", fmt.Errorf("generate_name: description is required")
		}

		names, err := generateNames(description)
		if err != nil {
			return "", err
		}
		return pickAllowed(names)
	},
}
