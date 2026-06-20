import { useEffect, useState } from 'react'
import { Box, Button, Container, Divider, Link as MuiLink, Stack, Typography } from '@mui/material'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { api } from '../api/client'
import { fonts } from '../theme'
import type { RedditPost } from '../api/types'
import RedditPromptDialog from '../components/RedditPromptDialog'

const navLinkSx = {
  fontFamily: fonts.mono,
  fontSize: '0.75rem',
  textTransform: 'uppercase',
  letterSpacing: '0.08em',
} as const

// RedditPostsPage lists the newest posts from the user's configured subreddit.
// Clicking a title opens a dialog that turns the post into a new document.
export default function RedditPostsPage() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
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

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  function openPost(post: RedditPost) {
    setSelected(post)
    setDialogOpen(true)
  }

  const loading = posts === null && error === null

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      {/* Masthead bar */}
      <Box
        component="header"
        sx={{
          borderBottom: '1px solid',
          borderColor: 'divider',
          px: { xs: 3, md: 5 },
          py: 2,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <Stack direction="row" spacing={3} alignItems="baseline">
          <MuiLink component={RouterLink} to="/" underline="none" color="inherit">
            <Typography
              sx={{ fontFamily: fonts.display, fontWeight: 600, fontSize: '1.5rem', letterSpacing: '-0.02em' }}
            >
              Nisaba
            </Typography>
          </MuiLink>
          <MuiLink component={RouterLink} to="/documents" underline="hover" sx={navLinkSx}>
            Documents
          </MuiLink>
          <MuiLink component={RouterLink} to="/settings" underline="hover" sx={navLinkSx}>
            Settings
          </MuiLink>
        </Stack>
        <Stack direction="row" spacing={2} alignItems="center">
          <Typography variant="overline" sx={{ color: 'text.secondary' }}>
            {user?.username}
          </Typography>
          <Button variant="text" size="small" onClick={handleLogout} sx={{ color: 'text.primary' }}>
            Log out
          </Button>
        </Stack>
      </Box>

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
