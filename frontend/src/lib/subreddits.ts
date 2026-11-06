// Subreddit filtering for the saved-posts tab of the Prompts page.

// ALL_SUBREDDITS is the filter value that shows every saved post.
export const ALL_SUBREDDITS = 'all'

export interface SubredditOption {
  name: string
  count: number
}

const same = (a: string, b: string) => a.toLowerCase() === b.toLowerCase()

// subredditOptions lists the subreddits present in posts with their counts:
// preferred first (always, even with no posts), then the rest by count
// descending, then name. Names compare case-insensitively.
export function subredditOptions(
  posts: { subreddit: string }[],
  preferred: string,
): SubredditOption[] {
  const counts = new Map<string, SubredditOption>()
  for (const p of posts) {
    const key = p.subreddit.toLowerCase()
    const entry = counts.get(key)
    if (entry) entry.count++
    else counts.set(key, { name: p.subreddit, count: 1 })
  }
  const first = counts.get(preferred.toLowerCase()) ?? { name: preferred, count: 0 }
  counts.delete(preferred.toLowerCase())
  const rest = [...counts.values()].sort(
    (a, b) => b.count - a.count || a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }),
  )
  return preferred ? [first, ...rest] : rest
}

// filterBySubreddit keeps posts from sr (case-insensitive); ALL_SUBREDDITS
// keeps everything.
export function filterBySubreddit<T extends { subreddit: string }>(posts: T[], sr: string): T[] {
  if (sr === ALL_SUBREDDITS) return posts
  return posts.filter((p) => same(p.subreddit, sr))
}
