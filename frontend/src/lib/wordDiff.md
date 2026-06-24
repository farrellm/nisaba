# Word-level diff algorithm

`wordDiff.ts` computes a word-level diff between two strings and returns ordered
segments suitable for an inline "track-changes" view (used by the per-attribute
diff page, `BlockAttributeDiffPage.tsx`). It is intentionally dependency-free.

```ts
type DiffSegment = { type: 'equal' | 'add' | 'remove'; text: string }
function wordDiff(before: string, after: string): DiffSegment[]
```

Each segment marks a run of text that is unchanged (`equal`), present only in
`after` (`add`), or present only in `before` (`remove`). Concatenating the
`equal`+`remove` text reproduces `before`; concatenating `equal`+`add`
reproduces `after`.

## 1. Tokenize

Each input is split into a flat array of tokens, where every token is exactly
one of:

- a **whitespace run** — `\s+`
- an **alphanumeric word run** — `[\p{L}\p{N}]+` (the Unicode `u` flag keeps
  accented and non-Latin letters inside one token, e.g. `résumé`)
- a **single punctuation/symbol character** — `[^\s\p{L}\p{N}]`

```ts
s.match(/\s+|[\p{L}\p{N}]+|[^\s\p{L}\p{N}]/gu) ?? []
```

So `"Hi, world!"` → `["Hi", ",", " ", "world", "!"]`.

Two consequences of this token granularity:

- **Whitespace is preserved as its own tokens**, so reconstructed output keeps
  the original spacing and newlines exactly (the view renders with
  `white-space: pre-wrap`).
- **Punctuation is separated from adjacent words**, so editing the text around a
  word doesn't mark the word itself as changed. `room.` → `room and smiled.`
  diffs as `room`(equal) + ` and smiled`(add) + `.`(equal), not a wholesale
  replacement. The flip side: intra-word marks also split (`don't` →
  `don` `'` `t`).

## 2. Longest common subsequence (LCS)

Given token arrays `a` (from `before`) and `b` (from `after`), build a table
where `lcs[i][j]` is the length of the longest common subsequence of the
suffixes `a[i:]` and `b[j:]`. Filled bottom-up from the ends:

```
lcs[i][j] = a[i] === b[j]
  ? lcs[i+1][j+1] + 1
  : max(lcs[i+1][j], lcs[i][j+1])
```

Tokens are compared by exact string equality. The LCS is the set of tokens that
stay put; everything else is an insertion or deletion.

## 3. Backtrack into segments

Walk forward from `(0, 0)` using the table to decide each step:

- `a[i] === b[j]` → emit `equal`, advance both `i` and `j`.
- else if `lcs[i+1][j] >= lcs[i][j+1]` → emit `remove` (token only in `before`),
  advance `i`.
- else → emit `add` (token only in `after`), advance `j`.

After one array is exhausted, the remaining `a` tokens are `remove`s and the
remaining `b` tokens are `add`s. The `>=` tie-break makes deletions precede
insertions at an equal-cost fork, which keeps the output deterministic.

### Coalescing

Adjacent tokens of the same type are merged into one segment as they're emitted
(the new token's text is appended to the previous segment when types match).
This turns token-level output into readable runs — e.g. a contiguous insertion
of several words/spaces becomes a single `add` segment rather than many.

## 4. Word counts

`wordCount(s)` counts alphanumeric word runs only (`[\p{L}\p{N}]+` with the `u`
flag), ignoring whitespace and punctuation. The diff page sums it over the
`add` and `remove` segments to show a `−N / +M words` summary, so a
punctuation-only change (e.g. a stray `.`) contributes `0` and isn't reported as
a changed word.

## Complexity & edge cases

- **Time / space:** `O(n·m)` for token arrays of length `n` and `m` (the LCS
  table). Fine for prose-sized attribute values; not intended for very large
  inputs.
- **Empty inputs:** `tokenize("")` yields `[]`. Empty `before` → everything is
  `add`; empty `after` → everything is `remove`; both empty → `[]`.
- **Identical inputs** collapse to a single `equal` segment.
