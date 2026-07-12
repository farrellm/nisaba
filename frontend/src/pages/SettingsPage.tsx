import { useEffect, useState, type FormEvent } from 'react'
import { Alert, Box, Button, Container, Divider, Stack, TextField, Typography } from '@mui/material'
import { useAuth } from '../auth/AuthContext'
import { api } from '../api/client'
import { useAsyncAction } from '../lib/useAsyncAction'
import Masthead from '../components/Masthead'
import { usePageTitle } from '../lib/usePageTitle'

// SettingsPage lets the user edit their configured subreddit. On save it PUTs to
// /api/auth/me and refreshes the auth context so the canonical value (which the
// server may have defaulted) is reflected everywhere.
export default function SettingsPage() {
  usePageTitle('Settings')
  const { user, refresh } = useAuth()
  const [subreddit, setSubreddit] = useState(user?.subreddit ?? '')
  const { busy: saving, error, run } = useAsyncAction()
  const [saved, setSaved] = useState(false)

  // Sync the field once the user loads (RequireAuth guarantees user before render,
  // but it may briefly be null while refreshing).
  useEffect(() => {
    if (user) setSubreddit(user.subreddit)
  }, [user])

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSaved(false)
    run(async () => {
      await api.put('/api/auth/me', { subreddit })
      await refresh()
      setSaved(true)
    })
  }

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead />

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
                {saving ? 'Saving…' : 'Save'}
              </Button>
            </Box>
          </Stack>
        </form>
      </Container>
    </Box>
  )
}
