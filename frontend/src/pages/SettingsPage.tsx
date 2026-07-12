import { useEffect, useState, type FormEvent } from 'react'
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Container,
  Divider,
  Link as MuiLink,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { api, ApiError } from '../api/client'
import { fonts } from '../theme'
import AccountMenu from '../components/AccountMenu'
import { usePageTitle } from '../lib/usePageTitle'

const navLinkSx = {
  fontFamily: fonts.mono,
  fontSize: '0.75rem',
  textTransform: 'uppercase',
  letterSpacing: '0.08em',
} as const

// SettingsPage lets the user edit their configured subreddit. On save it PUTs to
// /api/auth/me and refreshes the auth context so the canonical value (which the
// server may have defaulted) is reflected everywhere.
export default function SettingsPage() {
  usePageTitle('Settings')
  const { user, refresh } = useAuth()
  const [subreddit, setSubreddit] = useState(user?.subreddit ?? '')
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const [saving, setSaving] = useState(false)

  // Sync the field once the user loads (RequireAuth guarantees user before render,
  // but it may briefly be null while refreshing).
  useEffect(() => {
    if (user) setSubreddit(user.subreddit)
  }, [user])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSaved(false)
    setSaving(true)
    try {
      await api.put('/api/auth/me', { subreddit })
      await refresh()
      setSaved(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong. Try again.')
    } finally {
      setSaving(false)
    }
  }

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
          <MuiLink component={RouterLink} to="/reddit" underline="hover" sx={navLinkSx}>
            Prompts
          </MuiLink>
          <MuiLink component={RouterLink} to="/documents" underline="hover" sx={navLinkSx}>
            Documents
          </MuiLink>
        </Stack>
        <AccountMenu />
      </Box>

      <Container maxWidth="sm" sx={{ pt: { xs: 5, md: 8 }, pb: 12 }}>
        <Typography variant="overline" sx={{ color: 'primary.main', display: 'block', mb: 2 }}>
          Your account
        </Typography>
        <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)', mb: 4 }}>
          Settings
        </Typography>

        <Divider sx={{ mb: 4 }} />

        <form onSubmit={handleSubmit} noValidate>
          <Stack spacing={2}>
            {error && <Alert severity="error">{error}</Alert>}
            {saved && <Alert severity="success">Settings saved.</Alert>}
            <TextField
              label="Subreddit"
              value={subreddit}
              onChange={(e) => setSubreddit(e.target.value)}
              helperText="The subreddit to pull writing prompts from. Defaults to WritingPrompts."
              fullWidth
            />
            <Box>
              <Button type="submit" variant="contained" disabled={saving}>
                {saving ? (
                  <>
                    <CircularProgress size={16} color="inherit" sx={{ mr: 1 }} />
                    Saving…
                  </>
                ) : (
                  'Save'
                )}
              </Button>
            </Box>
          </Stack>
        </form>
      </Container>
    </Box>
  )
}
