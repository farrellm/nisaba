import { describe, expect, it } from 'vitest'
import { type DiffSegment, wordCount, wordDiff } from './wordDiff'

// Cases follow wordDiff.md section by section.

const eq = (text: string): DiffSegment => ({ type: 'equal', text })
const add = (text: string): DiffSegment => ({ type: 'add', text })
const remove = (text: string): DiffSegment => ({ type: 'remove', text })

// side rebuilds one input from the segments: drop the adds to get `before`,
// drop the removes to get `after` (exact per side, even with absorbed text).
function side(segments: DiffSegment[], drop: 'add' | 'remove'): string {
  return segments
    .filter((s) => s.type !== drop)
    .map((s) => s.text)
    .join('')
}

function expectInvariants(before: string, after: string): DiffSegment[] {
  const segments = wordDiff(before, after)
  expect(side(segments, 'add')).toBe(before)
  expect(side(segments, 'remove')).toBe(after)
  for (const s of segments) expect(s.text).not.toBe('')
  // Coalesced: no two neighbours share a type.
  for (let i = 1; i < segments.length; i++) {
    expect(segments[i]!.type).not.toBe(segments[i - 1]!.type)
  }
  return segments
}

describe('wordDiff edge cases', () => {
  it('returns nothing for two empty inputs', () => {
    expect(wordDiff('', '')).toEqual([])
  })

  it('collapses identical inputs to one equal segment', () => {
    expect(wordDiff('same text\nhere', 'same text\nhere')).toEqual([eq('same text\nhere')])
  })

  it('marks everything added when before is empty', () => {
    expect(wordDiff('', 'new words')).toEqual([add('new words')])
  })

  it('marks everything removed when after is empty', () => {
    expect(wordDiff('old words', '')).toEqual([remove('old words')])
  })
})

describe('wordDiff tokenization', () => {
  it('keeps punctuation separate so surrounding edits leave the word equal', () => {
    expect(wordDiff('room.', 'room and smiled.')).toEqual([eq('room'), add(' and smiled'), eq('.')])
  })

  it('treats internal apostrophes as part of the word', () => {
    expect(wordDiff("I don't know", 'I do not know')).toEqual([
      eq('I '),
      remove("don't"),
      add('do not'),
      eq(' know'),
    ])
  })

  it('preserves whitespace and newlines exactly', () => {
    expectInvariants('a  b\n\n\tc', 'a  x\n\n\tc')
  })
})

describe('wordDiff semantic cleanup', () => {
  it('merges adjacent word edits across a shared space', () => {
    expect(wordDiff('the quick brown fox', 'the slow red fox')).toEqual([
      eq('the '),
      remove('quick brown'),
      add('slow red'),
      eq(' fox'),
    ])
  })

  it('never absorbs an equality containing a newline', () => {
    expect(wordDiff('alpha\nbeta', 'gamma\ndelta')).toEqual([
      remove('alpha'),
      add('gamma'),
      eq('\n'),
      remove('beta'),
      add('delta'),
    ])
  })

  it('keeps a preserved phrase that outweighs its neighbouring edits', () => {
    expect(wordDiff('X keep these four words Y', 'P keep these four words Q')).toEqual([
      remove('X'),
      add('P'),
      eq(' keep these four words '),
      remove('Y'),
      add('Q'),
    ])
  })

  it('collapses a rewrite sharing only function words into one strike and insert', () => {
    const segments = expectInvariants(
      'the cat sat on a mat, then slept',
      'a dog ran to the park, and barked',
    )
    expect(segments.map((s) => s.type)).toEqual(['remove', 'add'])
  })
})

describe('wordDiff size guard', () => {
  // Enough tokens that the word-level table would exceed the cell cap, with
  // edits at both ends so prefix/suffix trimming can't shrink the middle.
  const lines = Array.from({ length: 1500 }, (_, i) => `line ${i} has some words`)
  const before = ['alpha opening line', ...lines, 'closing line omega'].join('\n')
  const after = ['beta opening line', ...lines, 'closing line zeta'].join('\n')

  it('refines changed lines to word level instead of striking whole lines', () => {
    const segments = expectInvariants(before, after)
    expect(segments.filter((s) => s.type === 'remove').map((s) => s.text)).toEqual([
      'alpha',
      'omega',
    ])
    expect(segments.filter((s) => s.type === 'add').map((s) => s.text)).toEqual(['beta', 'zeta'])
  })
})

describe('wordDiff invariants', () => {
  const pairs: [string, string][] = [
    ['Hi, world!', 'Hello, world?'],
    ["It's re-read twice.", "It's read once—maybe."],
    ['résumé café', 'resume cafe'],
    ['one\n\ntwo\n\nthree', 'one\n\n2\n\nthree\n\nfour'],
    ["'tis well--known", 'tis well-known'],
    ['a b c d e f g', 'g f e d c b a'],
  ]
  it.each(pairs)('rebuilds both sides: %j -> %j', (before, after) => {
    expectInvariants(before, after)
  })
})

describe('wordCount', () => {
  it.each([
    ['', 0],
    ['.', 0],
    ['  \n ', 0],
    ["don't stop.", 2],
    ['re-read the résumé', 3],
    ['Hi, world!', 2],
  ])('%j has %i words', (s, n) => {
    expect(wordCount(s)).toBe(n)
  })
})
