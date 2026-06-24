package handler

import "strings"

// parseTopLevelTags scans s for top-level XML-style tags and returns a map of
// tag name -> inner text. Only tags that are not nested inside another tag's
// body are considered; nested tags are kept verbatim as part of the enclosing
// tag's value (we never recurse). Opening-tag attributes are ignored for the
// key. Self-closing tags and empty bodies yield an empty-string value. Text
// outside any tag is ignored. When the same top-level name appears more than
// once, the last occurrence wins. A top-level tag that is never closed (e.g. a
// truncated response) is auto-closed at end of string, taking the rest of the
// text as its value.
//
// encoding/xml is unsuitable here because model responses are free-form text,
// not well-formed XML, so we scan bytes directly and degrade gracefully on
// malformed input.
func parseTopLevelTags(s string) map[string]string {
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

// applyRenames rewrites keys in updates according to renames (from -> to), using
// move semantics: when a "from" key is present its value is reassigned to "to"
// (overwriting any existing "to") and the original "from" key is removed. Keys
// not named in renames are untouched.
func applyRenames(updates map[string]string, renames map[string]string) {
	for from, to := range renames {
		if v, ok := updates[from]; ok {
			updates[to] = v
			delete(updates, from)
		}
	}
}

// findMatchingClose returns the index where the closing tag for name begins and
// the index just past that closing tag, starting the search at from. It counts
// nested start tags of the same name so the outermost close is matched. Returns
// (-1, -1) when no matching close exists.
func findMatchingClose(s string, from int, name string) (int, int) {
	depth := 0
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
				if depth == 0 {
					return i, tagEnd + 1
				}
				depth--
			}
		} else if isNameStart(byteAt(inner, 0)) && tagName(inner) == name &&
			!strings.HasSuffix(strings.TrimSpace(inner), "/") {
			depth++
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
