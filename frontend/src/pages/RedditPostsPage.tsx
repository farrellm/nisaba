import { useEffect, useState } from 'react'
import { Box, Container, Divider, Typography } from '@mui/material'
import { useAuth } from '../auth/AuthContext'
import { api } from '../api/client'
import { fonts } from '../theme'
import type { RedditPost } from '../api/types'
import Masthead from '../components/Masthead'
import RedditPromptDialog from '../components/RedditPromptDialog'

// RedditPostsPage lists the newest posts from the user's configured subreddit.
// Clicking a title opens a dialog that turns the post into a new document.
export default function RedditPostsPage() {
  const { user } = useAuth()
  const [posts, setPosts] = useState<RedditPost[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState<RedditPost | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)

  useEffect(() => {
    api
      .get<RedditPost[]>('/api/reddit/posts')
      .then(setPosts)
      .catch((e: unknown) => setError(String(e)))
  }, [])

  function openPost(post: RedditPost) {
    setSelected(post)
    setDialogOpen(true)
  }

  const loading = posts === null && error === null

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead active="prompts" />

      <Container maxWidth="md" sx={{ pt: { xs: 5, md: 8 }, pb: 12 }}>
        <Typography variant="overline" sx={{ color: 'primary.main', display: 'block', mb: 2 }}>
          r/{user?.subreddit}
        </Typography>
        <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)', mb: 4 }}>
          Newest prompts
        </Typography>

        <Divider sx={{ mb: 1 }} />

        {error ? (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'error.main', py: 1.5 }}>
            {error}
          </Typography>
        ) : loading ? (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'text.secondary', py: 1.5 }}>
            Loading…
          </Typography>
        ) : posts && posts.length > 0 ? (
          posts.map((post, i) => (
            <Box
              key={`${i}-${post.url}`}
              onClick={() => openPost(post)}
              sx={{ cursor: 'pointer', '&:hover .post-title': { color: 'primary.main' } }}
            >
              <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 2, py: 1.75 }}>
                <Typography
                  className="post-title"
                  sx={{ fontFamily: fonts.display, fontSize: '1.15rem', transition: 'color 120ms' }}
                >
                  {post.title}
                </Typography>
              </Box>
              <Divider sx={{ borderStyle: 'dotted' }} />
            </Box>
          ))
        ) : (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'text.secondary', py: 1.5 }}>
            Nothing here yet.
          </Typography>
        )}
      </Container>

      <RedditPromptDialog open={dialogOpen} post={selected} onClose={() => setDialogOpen(false)} />
    </Box>
  )
}
