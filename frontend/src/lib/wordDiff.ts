// wordDiff computes a word-level diff between two strings, returning ordered
// segments suitable for an inline "track-changes" view. It's intentionally
// dependency-free (the repo keeps a lean dep list) and mirrors the scan-style
// utilities like responseSegments.ts.
//
// Tokens alternate between word runs and whitespace runs so the rendered output
// reconstructs the original spacing and newlines exactly. The diff itself is a
// classic longest-common-subsequence over the token arrays, then backtracked
// into equal/remove/add segments (remove = in `before` only, add = in `after`
// only). Adjacent same-type tokens are coalesced into one segment.

export type DiffSegment = { type: 'equal' | 'add' | 'remove'; text: string }

// Split into whitespace runs, alphanumeric word runs, and individual
// punctuation/symbol characters, e.g. "Hi, world!" ->
// ["Hi", ",", " ", "world", "!"]. The Unicode (u) flag keeps accented and
// non-Latin letters inside word tokens. Empty input yields no tokens.
function tokenize(s: string): string[] {
  return s.match(/\s+|[\p{L}\p{N}]+|[^\s\p{L}\p{N}]/gu) ?? []
}

export function wordDiff(before: string, after: string): DiffSegment[] {
  const a = tokenize(before)
  const b = tokenize(after)
  const n = a.length
  const m = b.length

  // LCS length table: lcs[i][j] = LCS of a[i:] and b[j:].
  const lcs: number[][] = Array.from({ length: n + 1 }, () => new Array(m + 1).fill(0))
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lcs[i][j] = a[i] === b[j] ? lcs[i + 1][j + 1] + 1 : Math.max(lcs[i + 1][j], lcs[i][j + 1])
    }
  }

  const segments: DiffSegment[] = []
  const push = (type: DiffSegment['type'], text: string) => {
    const last = segments[segments.length - 1]
    if (last && last.type === type) last.text += text
    else segments.push({ type, text })
  }

  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      push('equal', a[i])
      i++
      j++
    } else if (lcs[i + 1][j] >= lcs[i][j + 1]) {
      push('remove', a[i])
      i++
    } else {
      push('add', b[j])
      j++
    }
  }
  while (i < n) push('remove', a[i++])
  while (j < m) push('add', b[j++])

  return segments
}

// wordCount counts alphanumeric word runs in a string, ignoring whitespace and
// punctuation (so a pure-punctuation segment contributes 0).
export function wordCount(s: string): number {
  return (s.match(/[\p{L}\p{N}]+/gu) ?? []).length
}
