// Package mode defines the fixed, code-managed set of writing modes. Each mode
// declares a stable set of input keys, the document attribute key its output is
// written back to, and a mustache template that turns a block's key/values into
// a prompt. The set is fixed at build time — there is no runtime CRUD.
package mode

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/farrellm/nisaba/internal/llm"
)

// Mode is one entry in the fixed registry.
type Mode struct {
	Name     string        `json:"name"`   // stable id, stored in blocks.mode
	Label    string        `json:"label"`  // human-facing name for the UI
	Keys     []string      `json:"keys"`   // input attribute keys (fixed per mode)
	Output   string        `json:"output"` // document attribute key the response populates
	Template string        `json:"-"`      // mustache prompt; server-side only
	Tools    []llm.ToolDef `json:"-"`      // tool functions attached to the LLM call; server-side only
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

// TemplatesBaseDir is the on-disk path to the default templates directory.
// Per-user overrides live in siblings named "<TemplatesBaseDir>-<username>".
var TemplatesBaseDir = "internal/mode/templates"

// TemplateFor returns the mustache template for mode m, preferring a per-user
// override at "<TemplatesBaseDir>-<username>/<m.Name>.mustache" when it exists
// and is readable, otherwise the embedded default (m.Template). The fallback is
// per-file, so a user may override only some modes.
func TemplateFor(username string, m Mode) string {
	if !safeUsername(username) {
		return m.Template
	}
	path := filepath.Join(TemplatesBaseDir+"-"+username, m.Name+".mustache")
	b, err := os.ReadFile(path)
	if err != nil {
		return m.Template
	}
	return string(b)
}

// safeUsername reports whether username is safe to use as a path component.
// Usernames are free-form user input, so we allow only [A-Za-z0-9_-] to keep a
// crafted name (e.g. containing "/" or "..") from escaping the base directory.
func safeUsername(username string) bool {
	if username == "" {
		return false
	}
	for _, r := range username {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}
