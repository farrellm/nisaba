import { useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { useSearchParams } from 'react-router'
import {
  Box,
  Button,
  Container,
  Divider,
  Fab,
  MenuItem,
  Link as MuiLink,
  Select,
  Tooltip,
  Typography,
  type SelectChangeEvent,
} from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import { useAuth } from '../auth/AuthContext'
import { fonts } from '../theme'
import type { RedditPost } from '../api/types'
import Masthead from '../components/Masthead'
import RedditPromptDialog from '../components/RedditPromptDialog'
import RedditUrlDialog from '../components/RedditUrlDialog'
import StatusLine from '../components/StatusLine'
import { listStatusSx, postLinkSx } from '../lib/styles'
import { ALL_SUBREDDITS, filterBySubreddit, subredditOptions } from '../lib/subreddits'
import { useFetch } from '../lib/useFetch'
import { usePageTitle } from '../lib/usePageTitle'

type Tab = 'newest' | 'saved'

const TABS: { key: Tab; label: string }[] = [
  { key: 'newest', label: 'Newest' },
  { key: 'saved', label: 'Saved' },
]

// RedditPostsPage lists Reddit posts to turn into documents, in two tabs: the
// newest posts from the user's configured subreddit, and the Reddit account's
// saved posts (filterable by subreddit, defaulting to the user's). Tab and
// filter live in the URL (?tab=saved&sr=…). Clicking a title opens a dialog
// that turns the post into a new document.
export default function RedditPostsPage() {
  const { user } = useAuth()
  const subreddit = user?.subreddit ?? ''
  const [params, setParams] = useSearchParams()
  const tab: Tab = params.get('tab') === 'saved' ? 'saved' : 'newest'
  const sr = params.get('sr') || subreddit
  usePageTitle(tab === 'saved' ? 'Saved posts' : 'Writing Prompts')

  const [selected, setSelected] = useState<RedditPost | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [urlDialogOpen, setUrlDialogOpen] = useState(false)
  const tabRefs = useRef<Record<Tab, HTMLButtonElement | null>>({ newest: null, saved: null })

  function openPost(post: RedditPost) {
    setSelected(post)
    setDialogOpen(true)
  }

  function selectTab(next: Tab) {
    if (next === tab) return
    setParams(next === 'saved' ? { tab: 'saved' } : {})
  }

  function setFilter(next: string) {
    setParams(
      next.toLowerCase() === subreddit.toLowerCase()
        ? { tab: 'saved' }
        : { tab: 'saved', sr: next },
      { replace: true },
    )
  }

  function onTabKeyDown(e: KeyboardEvent) {
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return
    e.preventDefault()
    const next: Tab = tab === 'newest' ? 'saved' : 'newest'
    selectTab(next)
    tabRefs.current[next]?.focus()
  }

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead active="prompts" />

      <Container maxWidth="md" sx={{ pt: { xs: 5, md: 8 }, pb: 12 }}>
        {tab === 'newest' && (
          <Typography variant="overline" sx={{ color: 'primary.main', display: 'block', mb: 2 }}>
            r/{subreddit}
          </Typography>
        )}

        {/* The headline doubles as the tab switcher. */}
        <Box
          role="tablist"
          aria-label="Prompt sources"
          onKeyDown={onTabKeyDown}
          sx={{ display: 'flex', flexWrap: 'wrap', columnGap: { xs: 2.5, md: 4 }, mb: 4 }}
        >
          {TABS.map(({ key, label }) => {
            const active = key === tab
            return (
              <Typography
                key={key}
                component="button"
                type="button"
                role="tab"
                id={`tab-${key}`}
                aria-selected={active}
                aria-controls={`panel-${key}`}
                tabIndex={active ? 0 : -1}
                ref={(el: HTMLButtonElement | null) => {
                  tabRefs.current[key] = el
                }}
                onClick={() => selectTab(key)}
                variant="h1"
                sx={{
                  fontSize: 'clamp(2.25rem, 6vw, 3.5rem)',
                  background: 'none',
                  border: 0,
                  p: 0,
                  cursor: active ? 'default' : 'pointer',
                  color: active ? 'text.primary' : 'text.secondary',
                  textDecoration: 'none',
                  textDecorationThickness: '2px',
                  textUnderlineOffset: '0.12em',
                  transition: 'color 120ms',
                  '&:hover': active ? {} : { textDecoration: 'underline' },
                  '&:focus-visible': {
                    outline: '2px solid',
                    outlineColor: 'primary.main',
                    outlineOffset: '4px',
                    borderRadius: 1,
                  },
                }}
              >
                {label}
              </Typography>
            )
          })}
        </Box>

        <Box role="tabpanel" id={`panel-${tab}`} aria-labelledby={`tab-${tab}`}>
          {tab === 'newest' ? (
            <NewestPanel subreddit={subreddit} onOpen={openPost} />
          ) : (
            <SavedPanel subreddit={subreddit} filter={sr} onFilter={setFilter} onOpen={openPost} />
          )}
        </Box>
      </Container>

      <Tooltip title="Import from URL" placement="left">
        <Fab
          color="primary"
          aria-label="Import from URL"
          onClick={() => setUrlDialogOpen(true)}
          sx={{ position: 'fixed', bottom: 32, right: 32, borderRadius: '50%' }}
        >
          <AddIcon />
        </Fab>
      </Tooltip>

      <RedditUrlDialog
        open={urlDialogOpen}
        onClose={() => setUrlDialogOpen(false)}
        onResolved={(post) => {
          setUrlDialogOpen(false)
          openPost(post)
        }}
      />

      <RedditPromptDialog open={dialogOpen} post={selected} onClose={() => setDialogOpen(false)} />
    </Box>
  )
}

// NewestPanel is the newest posts of the user's subreddit.
function NewestPanel({
  subreddit,
  onOpen,
}: {
  subreddit: string
  onOpen: (post: RedditPost) => void
}) {
  const { data: posts, error, loading } = useFetch<RedditPost[]>('/api/reddit/posts')
  return (
    <>
      <Divider sx={{ mb: 1 }} />
      {error ? (
        <FetchError
          error={error}
          href={`https://old.reddit.com/r/${encodeURIComponent(subreddit)}/new`}
          label={`Browse r/${subreddit}/new on Reddit ↗`}
        />
      ) : loading ? (
        <StatusLine sx={listStatusSx}>Loading…</StatusLine>
      ) : posts && posts.length > 0 ? (
        posts.map((post, i) => <PostRow key={`${i}-${post.url}`} post={post} onOpen={onOpen} />)
      ) : (
        <StatusLine sx={listStatusSx}>Nothing here yet.</StatusLine>
      )}
    </>
  )
}

// SavedPanel is the Reddit account's saved posts, filtered to one subreddit
// (or all of them). It fetches only when the Saved tab is shown.
function SavedPanel({
  subreddit,
  filter,
  onFilter,
  onOpen,
}: {
  subreddit: string
  filter: string
  onFilter: (sr: string) => void
  onOpen: (post: RedditPost) => void
}) {
  const { data, error, loading } = useFetch<RedditPost[]>('/api/reddit/saved')
  const all = useMemo(() => data ?? [], [data])
  const options = useMemo(() => {
    const opts = subredditOptions(all, subreddit)
    // A subreddit from the URL that no saved post carries still needs an option.
    if (
      filter !== ALL_SUBREDDITS &&
      !opts.some((o) => o.name.toLowerCase() === filter.toLowerCase())
    ) {
      opts.push({ name: filter, count: 0 })
    }
    return opts
  }, [all, subreddit, filter])
  const value =
    filter === ALL_SUBREDDITS
      ? ALL_SUBREDDITS
      : (options.find((o) => o.name.toLowerCase() === filter.toLowerCase())?.name ?? filter)
  const posts = useMemo(() => filterBySubreddit(all, value), [all, value])
  const showSubreddit = value === ALL_SUBREDDITS

  return (
    <>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, mb: 0.5 }}>
        <Typography variant="overline" sx={{ color: 'text.secondary' }}>
          Subreddit
        </Typography>
        <Select
          value={value}
          onChange={(e: SelectChangeEvent) => onFilter(e.target.value)}
          size="small"
          variant="standard"
          disableUnderline
          inputProps={{ 'aria-label': 'Filter saved posts by subreddit' }}
          sx={{ fontFamily: fonts.mono, fontSize: '0.8rem', minWidth: 0 }}
        >
          {options.map((o) => (
            <MenuItem
              key={o.name}
              value={o.name}
              sx={{ fontFamily: fonts.mono, fontSize: '0.8rem' }}
            >
              r/{o.name}
              {data ? ` (${o.count})` : ''}
            </MenuItem>
          ))}
          <MenuItem value={ALL_SUBREDDITS} sx={{ fontFamily: fonts.mono, fontSize: '0.8rem' }}>
            All subreddits{data ? ` (${all.length})` : ''}
          </MenuItem>
        </Select>
      </Box>

      <Divider sx={{ mb: 1 }} />

      {error ? (
        <FetchError
          error={error}
          href="https://old.reddit.com/user/me/saved"
          label="Open your saved posts on Reddit ↗"
        />
      ) : loading ? (
        <StatusLine sx={listStatusSx}>Loading…</StatusLine>
      ) : posts.length > 0 ? (
        posts.map((post, i) => (
          <PostRow
            key={`${i}-${post.url}`}
            post={post}
            onOpen={onOpen}
            showSubreddit={showSubreddit}
          />
        ))
      ) : all.length === 0 ? (
        <StatusLine sx={listStatusSx}>No saved posts on this Reddit account.</StatusLine>
      ) : (
        <Box sx={{ display: 'flex', alignItems: 'baseline', flexWrap: 'wrap', gap: 1.5 }}>
          <StatusLine sx={listStatusSx}>No saved posts from r/{value}.</StatusLine>
          <Button
            size="small"
            onClick={() => onFilter(ALL_SUBREDDITS)}
            sx={{ fontFamily: fonts.mono, fontSize: '0.8rem', minWidth: 0, p: 0 }}
          >
            Show all subreddits
          </Button>
        </Box>
      )}
    </>
  )
}

// FetchError reports a failed listing and offers browsing on Reddit instead;
// a post found there can come back via Import from URL.
function FetchError({ error, href, label }: { error: string; href: string; label: string }) {
  return (
    <>
      <StatusLine tone="error" sx={listStatusSx}>
        {error}
      </StatusLine>
      <MuiLink
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        variant="overline"
        sx={postLinkSx}
      >
        {label}
      </MuiLink>
    </>
  )
}

// PostRow is one ledger line: the post title, optionally followed by its
// subreddit. The whole row opens the import dialog.
function PostRow({
  post,
  onOpen,
  showSubreddit = false,
}: {
  post: RedditPost
  onOpen: (post: RedditPost) => void
  showSubreddit?: boolean
}) {
  return (
    <Box
      onClick={() => onOpen(post)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onOpen(post)
        }
      }}
      // A card-sized row; a real <button> would fight the Box layout.
      // oxlint-disable-next-line jsx-a11y/prefer-tag-over-role
      role="button"
      tabIndex={0}
      sx={{
        cursor: 'pointer',
        '&:hover .post-title': { color: 'primary.main' },
        '&:focus-visible': {
          outline: '2px solid',
          outlineColor: 'primary.main',
          outlineOffset: '2px',
          borderRadius: 1,
        },
      }}
    >
      {/* On phones the subreddit drops under the title instead of squeezing it. */}
      <Box
        sx={{
          display: 'flex',
          flexDirection: { xs: 'column', sm: 'row' },
          alignItems: 'baseline',
          columnGap: 2,
          rowGap: 0.5,
          py: 1.75,
        }}
      >
        <Typography
          className="post-title"
          sx={{
            fontFamily: fonts.display,
            fontSize: '1.15rem',
            transition: 'color 120ms',
            flex: 1,
          }}
        >
          {post.title}
        </Typography>
        {showSubreddit && post.subreddit && (
          <Typography
            sx={{
              fontFamily: fonts.mono,
              fontSize: '0.75rem',
              color: 'text.secondary',
              flexShrink: 0,
            }}
          >
            r/{post.subreddit}
          </Typography>
        )}
      </Box>
      <Divider sx={{ borderStyle: 'dotted' }} />
    </Box>
  )
}
