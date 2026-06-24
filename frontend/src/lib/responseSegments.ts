// responseSegments parses a free-form LLM response into an ordered, lossless
// list of segments: the text outside any top-level XML-style tag, and each
// top-level tag with its verbatim inner body.
//
// This mirrors the backend's parseTopLevelTags (backend/internal/handler/
// response.go) byte-scan, with two differences: it preserves order and keeps
// the text between/around tags (the backend builds an attribute map and drops
// non-tag text), and it keeps duplicate tag names (the backend's "last wins"
// only matters for attributes). Like the backend, it degrades gracefully on
// malformed input rather than using a strict XML parser.

export type ResponseSegment =
  | { kind: 'text'; text: string }
  | { kind: 'tag'; name: string; inner: string }

export function parseResponseSegments(s: string): ResponseSegment[] {
  const out: ResponseSegment[] = []
  let textStart = 0
  let i = 0

  const flushText = (end: number) => {
    if (end > textStart) out.push({ kind: 'text', text: s.slice(textStart, end) })
  }

  while (i < s.length) {
    const lt = s.indexOf('<', i)
    if (lt < 0) break
    i = lt

    // Not a start tag (closing tag, comment, declaration, stray '<'): leave it
    // as part of the surrounding text and keep scanning.
    if (i + 1 >= s.length || !isNameStart(s[i + 1])) {
      i++
      continue
    }

    const gt = s.indexOf('>', i)
    if (gt < 0) break // unterminated start tag; nothing more to parse
    const inner = s.slice(i + 1, gt) // between '<' and '>'

    const name = tagName(inner)
    if (name === '') {
      i = gt + 1
      continue
    }

    // Self-closing tag: empty body, no closing tag to find.
    if (inner.trim().endsWith('/')) {
      flushText(i)
      out.push({ kind: 'tag', name, inner: '' })
      i = gt + 1
      textStart = i
      continue
    }

    const valueStart = gt + 1
    const close = findMatchingClose(s, valueStart, name)
    if (close === null) {
      // Never-closed tag: response likely truncated. Auto-close at end of
      // string and capture the rest as this tag's body, mirroring the backend.
      flushText(i)
      out.push({ kind: 'tag', name, inner: s.slice(valueStart) })
      textStart = s.length
      break
    }
    flushText(i)
    out.push({ kind: 'tag', name, inner: s.slice(valueStart, close.start) })
    i = close.after
    textStart = i
  }

  flushText(s.length)
  return out
}

// findMatchingClose returns the index where the closing tag for name begins and
// the index just past it, starting at from. It counts nested same-name start
// tags so the outermost close is matched. Returns -1 when no close exists.
function findMatchingClose(s: string, from: number, name: string): { start: number; after: number } | null {
  let depth = 0
  let i = from
  while (i < s.length) {
    const lt = s.indexOf('<', i)
    if (lt < 0) return null
    i = lt
    const gt = s.indexOf('>', i)
    if (gt < 0) return null
    const inner = s.slice(i + 1, gt)

    if (inner.startsWith('/')) {
      if (tagName(inner.slice(1)) === name) {
        if (depth === 0) return { start: i, after: gt + 1 }
        depth--
      }
    } else if (isNameStart(inner[0] ?? '') && tagName(inner) === name && !inner.trim().endsWith('/')) {
      depth++
    }
    i = gt + 1
  }
  return null
}

// tagName extracts the element name from a start tag's interior (between '<'
// and '>'), stopping at the first whitespace or '/'.
function tagName(inner: string): string {
  let end = inner.length
  for (let j = 0; j < inner.length; j++) {
    const c = inner[j]
    if (c === ' ' || c === '\t' || c === '\n' || c === '\r' || c === '/') {
      end = j
      break
    }
  }
  const name = inner.slice(0, end)
  if (name === '' || !isNameStart(name[0])) return ''
  return name
}

function isNameStart(c: string): boolean {
  return c === '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
