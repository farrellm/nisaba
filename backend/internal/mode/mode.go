// Package mode defines the fixed, code-managed set of writing modes. Each mode
// declares a stable set of input keys, the document attribute key its output is
// written back to, and a mustache template that turns a block's key/values into
// a prompt. The set is fixed at build time — there is no runtime CRUD.
package mode

import (
	_ "embed"
)

// Mode is one entry in the fixed registry.
type Mode struct {
	Name     string   `json:"name"`   // stable id, stored in blocks.mode
	Label    string   `json:"label"`  // human-facing name for the UI
	Keys     []string `json:"keys"`   // input attribute keys (fixed per mode)
	Output   string   `json:"output"` // document attribute key the response populates
	Template string   `json:"-"`      // mustache prompt; server-side only
}

//go:embed templates/brainstorm.mustache
var brainstormTmpl string

//go:embed templates/outline.mustache
var outlineTmpl string

//go:embed templates/draft.mustache
var draftTmpl string

// modes is the registry, ordered for display.
var modes = []Mode{
	{Name: "brainstorm", Label: "Brainstorm", Keys: []string{"topic", "audience"}, Output: "ideas", Template: brainstormTmpl},
	{Name: "outline", Label: "Outline", Keys: []string{"topic", "ideas"}, Output: "outline", Template: outlineTmpl},
	{Name: "draft", Label: "Draft", Keys: []string{"topic", "outline", "tone"}, Output: "draft", Template: draftTmpl},
}

// All returns the modes in display order.
func All() []Mode {
	return modes
}

// Get returns the mode with the given name and whether it exists.
func Get(name string) (Mode, bool) {
	for _, m := range modes {
		if m.Name == name {
			return m, true
		}
	}
	return Mode{}, false
}
