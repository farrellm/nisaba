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

//go:embed templates/system.mustache
var systemTmpl string

//go:embed templates/generic.mustache
var genericTmpl string

//go:embed templates/brainstorm-1.mustache
var brainstorm1Tmpl string

//go:embed templates/brainstorm-2.mustache
var brainstorm2Tmpl string

//go:embed templates/brainstorm-creative-2.mustache
var brainstormCreative2Tmpl string

//go:embed templates/brainstorm-tools-1.mustache
var brainstormTools1Tmpl string

//go:embed templates/brainstorm-tools-2.mustache
var brainstormTools2Tmpl string

//go:embed templates/authors.mustache
var authorsTmpl string

//go:embed templates/revise-outline-1.mustache
var reviseOutline1Tmpl string

//go:embed templates/revise-outline-2.mustache
var reviseOutline2Tmpl string

//go:embed templates/scp-outline.mustache
var scpOutlineTmpl string

//go:embed templates/story.mustache
var storyTmpl string

//go:embed templates/story-sequel.mustache
var storySequelTmpl string

//go:embed templates/story-edit.mustache
var storyEditTmpl string

//go:embed templates/story-revise.mustache
var storyReviseTmpl string

// modes is the registry, ordered for display. Each mode's response carries its
// products in top-level XML tags (e.g. <characters>, <outline>), which are parsed
// back into document attributes — so downstream modes consume them via Keys and
// Output stays empty.
var modes = []Mode{
	{Name: "generic", Label: "Generic", Keys: []string{"prompt"}, Template: genericTmpl},
	{Name: "brainstorm-1", Label: "Brainstorm", Keys: []string{"prompt"}, Template: brainstorm1Tmpl},
	{Name: "brainstorm-2", Label: "Brainstorm (two-act)", Keys: []string{"prompt"}, Template: brainstorm2Tmpl},
	{Name: "brainstorm-creative-2", Label: "Brainstorm (creative, two-act)", Keys: []string{"prompt"}, Template: brainstormCreative2Tmpl},
	{Name: "brainstorm-tools-1", Label: "Brainstorm (tools)", Keys: []string{"prompt"}, Template: brainstormTools1Tmpl, Tools: []llm.ToolDef{llm.GenerateNameTool}},
	{Name: "brainstorm-tools-2", Label: "Brainstorm (tools, two-act)", Keys: []string{"prompt"}, Template: brainstormTools2Tmpl, Tools: []llm.ToolDef{llm.GenerateNameTool}},
	{Name: "authors", Label: "Suggest authors", Keys: []string{"outline", "characters"}, Template: authorsTmpl},
	{Name: "revise-outline-1", Label: "Revise outline", Keys: []string{"prompt", "characters", "outline"}, Template: reviseOutline1Tmpl},
	{Name: "revise-outline-2", Label: "Revise outline (two-act)", Keys: []string{"prompt", "characters", "outline"}, Template: reviseOutline2Tmpl},
	{Name: "scp-outline", Label: "SCP outline", Keys: []string{"prompt"}, Template: scpOutlineTmpl},
	{Name: "story", Label: "Story", Keys: []string{"characters", "author", "outline"}, Template: storyTmpl},
	{Name: "story-sequel", Label: "Sequel", Keys: []string{"story", "characters", "author", "style_analysis", "sequel_outline"}, Template: storySequelTmpl},
	{Name: "story-edit", Label: "Edit story", Keys: []string{"story", "edit"}, Template: storyEditTmpl},
	{Name: "story-revise", Label: "Revise story", Keys: []string{"story"}, Template: storyReviseTmpl},
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
	return override(username, m.Name, m.Template)
}

// SystemPrompt returns the mustache template for the LLM system prompt,
// preferring a per-user override at "<TemplatesBaseDir>-<username>/system.mustache"
// (the same dir as mode overrides) when present, otherwise the embedded default.
func SystemPrompt(username string) string {
	return override(username, "system", systemTmpl)
}

// override returns the per-user override file
// "<TemplatesBaseDir>-<username>/<name>.mustache" when it exists and is readable,
// otherwise fallback. A non-safe username always yields fallback.
func override(username, name, fallback string) string {
	if !safeUsername(username) {
		return fallback
	}
	b, err := os.ReadFile(filepath.Join(TemplatesBaseDir+"-"+username, name+".mustache"))
	if err != nil {
		return fallback
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
