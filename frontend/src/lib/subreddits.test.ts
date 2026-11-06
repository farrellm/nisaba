import { describe, expect, it } from 'vitest'
import { ALL_SUBREDDITS, filterBySubreddit, subredditOptions } from './subreddits'

const posts = [
  { subreddit: 'Fantasy' },
  { subreddit: 'writingprompts' },
  { subreddit: 'Fantasy' },
  { subreddit: 'Books' },
  { subreddit: 'Art' },
]

describe('subredditOptions', () => {
  it('puts preferred first and sorts the rest by count then name', () => {
    expect(subredditOptions(posts, 'WritingPrompts')).toEqual([
      { name: 'writingprompts', count: 1 },
      { name: 'Fantasy', count: 2 },
      { name: 'Art', count: 1 },
      { name: 'Books', count: 1 },
    ])
  })

  it('keeps preferred with a zero count when absent', () => {
    expect(subredditOptions([{ subreddit: 'Art' }], 'WritingPrompts')).toEqual([
      { name: 'WritingPrompts', count: 0 },
      { name: 'Art', count: 1 },
    ])
  })
})

describe('filterBySubreddit', () => {
  it('matches case-insensitively', () => {
    expect(filterBySubreddit(posts, 'WritingPrompts')).toEqual([{ subreddit: 'writingprompts' }])
  })

  it('passes everything for all', () => {
    expect(filterBySubreddit(posts, ALL_SUBREDDITS)).toBe(posts)
  })
})
