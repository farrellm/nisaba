// Package tagparse extracts top-level XML-style tags from free-form LLM
// output. encoding/xml is unsuitable here because model responses are not
// well-formed XML, so the scanner reads bytes directly and degrades gracefully
// on malformed input.
//
// The frontend mirrors this scan in frontend/src/lib/responseSegments.ts for
// ordered rendering; keep the two in sync.
package tagparse

import "strings"

// Parse scans s for top-level XML-style tags and returns a map of tag name ->
// inner text. Only tags that are not nested inside another tag's body are
// considered; nested tags are kept verbatim as part of the enclosing tag's
// value (we never recurse). Opening-tag attributes are ignored for the key.
// Self-closing tags and empty bodies yield an empty-string value. Text outside
// any tag is ignored. When the same top-level name appears more than once, the
// last occurrence wins. A repeated opening tag of the same name (e.g. a model
// that writes <x> again where it meant </x>) implicitly closes the open tag
// right before the second open, so each becomes its own top-level tag rather
// than one nesting the other. A top-level tag that is never closed (e.g. a
// truncated response) is auto-closed at end of string, taking the rest of the
// text as its value.
func Parse(s string) map[string]string {
	out := map[string]string{}
	i := 0
	for i < len(s) {
		lt := strings.IndexByte(s[i:], '<')
		if lt < 0 {
			break
		}
		i += lt

		// Skip closing tags, comments, and processing/doctype declarations;
		// none can start a top-level element here.
		if i+1 >= len(s) || !isNameStart(s[i+1]) {
			i++
			continue
		}

		gt := strings.IndexByte(s[i:], '>')
		if gt < 0 {
			break // unterminated start tag; nothing more to parse
		}
		tagEnd := i + gt         // index of '>'
		inner := s[i+1 : tagEnd] // between '<' and '>'

		name := tagName(inner)
		if name == "" {
			i = tagEnd + 1
			continue
		}

		// Self-closing tag: empty value, no closing tag to find.
		if strings.HasSuffix(strings.TrimSpace(inner), "/") {
			out[name] = ""
			i = tagEnd + 1
			continue
		}

		valueStart := tagEnd + 1
		closeIdx, afterClose := findMatchingClose(s, valueStart, name)
		if closeIdx < 0 {
			// Never-closed tag: the response was likely truncated (or the model
			// omitted the final close). Auto-close at end of string and capture
			// the rest as this tag's value. This necessarily ends parsing.
			out[name] = s[valueStart:]
			break
		}
		out[name] = s[valueStart:closeIdx]
		i = afterClose
	}
	return out
}

// findMatchingClose returns the index where the closing tag for name ends and
// the index just past that close, starting the search at from. It returns on the
// first same-name boundary: a real closing tag </name>, or a repeated opening
// tag <name> (not self-closing), which forces an implicit close right before that
// second open — so unbalanced "two opens, one close" input degrades into two
// sibling tags rather than one runaway unclosed tag. For an implicit close both
// returned indices are the position of the second open's '<', so the caller
// re-parses it as a fresh element. Self-closing same-name tags are treated as
// body content and skipped. Returns (-1, -1) when no boundary exists.
func findMatchingClose(s string, from int, name string) (int, int) {
	i := from
	for i < len(s) {
		lt := strings.IndexByte(s[i:], '<')
		if lt < 0 {
			return -1, -1
		}
		i += lt
		gt := strings.IndexByte(s[i:], '>')
		if gt < 0 {
			return -1, -1
		}
		tagEnd := i + gt
		inner := s[i+1 : tagEnd]

		if strings.HasPrefix(inner, "/") {
			if tagName(inner[1:]) == name {
				return i, tagEnd + 1
			}
		} else if isNameStart(byteAt(inner, 0)) && tagName(inner) == name &&
			!strings.HasSuffix(strings.TrimSpace(inner), "/") {
			// Repeated open of the same name: implicitly close just before it and
			// let the caller re-parse this open as a new element.
			return i, i
		}
		i = tagEnd + 1
	}
	return -1, -1
}

// tagName extracts the element name from the text between '<' and '>' (the
// opening tag's interior), stopping at the first whitespace or '/'.
func tagName(inner string) string {
	end := len(inner)
	for j := 0; j < len(inner); j++ {
		c := inner[j]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '/' {
			end = j
			break
		}
	}
	name := inner[:end]
	if name == "" || !isNameStart(name[0]) {
		return ""
	}
	return name
}

func isNameStart(c byte) bool {
	return c == '_' ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z')
}

func byteAt(s string, i int) byte {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}
