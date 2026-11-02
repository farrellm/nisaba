import { useEffect, useState, type FormEvent } from 'react'
import {
  Alert,
  Box,
  Button,
  Container,
  Divider,
  FormControlLabel,
  Stack,
  Switch,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
  useColorScheme,
} from '@mui/material'
import LightModeOutlined from '@mui/icons-material/LightModeOutlined'
import DarkModeOutlined from '@mui/icons-material/DarkModeOutlined'
import SettingsBrightnessOutlined from '@mui/icons-material/SettingsBrightnessOutlined'
import { useAuth } from '../auth/AuthContext'
import { api } from '../api/client'
import { useAsyncAction } from '../lib/useAsyncAction'
import Masthead from '../components/Masthead'
import { usePageTitle } from '../lib/usePageTitle'
import { disablePush, enablePush, getPushState, type PushState } from '../lib/push'

// What the Notifications section says for each state; the switch is only
// usable in 'on'/'off'.
const pushDescriptions: Record<PushState, string> = {
  on: 'This device gets a notification whenever a run finishes.',
  off: 'Get a notification on this device whenever a run finishes.',
  'needs-install':
    'On iPhone, notifications only work from the Home Screen app: tap Share → Add to Home Screen, then open Nisaba from there.',
  denied: 'Notifications are blocked for Nisaba in this browser’s settings.',
  unsupported: 'This browser doesn’t support push notifications.',
  'server-disabled': 'Push notifications aren’t configured on this server.',
}

// SettingsPage lets the user edit their configured subreddit. On save it PUTs to
// /api/auth/me and refreshes the auth context so the canonical value (which the
// server may have defaulted) is reflected everywhere. The appearance toggle is
// per-browser (MUI keeps it in localStorage) and applies immediately, as is the
// run-notifications switch (this browser's Web Push subscription).
export default function SettingsPage() {
  usePageTitle('Settings')
  const { user, refresh } = useAuth()
  const [subreddit, setSubreddit] = useState(user?.subreddit ?? '')
  const { busy: saving, error, run } = useAsyncAction()
  const [saved, setSaved] = useState(false)
  const { mode, setMode } = useColorScheme()
  const [pushState, setPushState] = useState<PushState | null>(null)
  const push = useAsyncAction()

  // Notification state is per-device (this browser's push subscription).
  useEffect(() => {
    let cancelled = false
    getPushState()
      .then((state) => {
        if (!cancelled) setPushState(state)
      })
      .catch(() => {
        if (!cancelled) setPushState('unsupported')
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Must run straight from the tap: iOS only shows the permission prompt for
  // a user gesture.
  function togglePush() {
    push.run(
      async () => {
        setPushState(pushState === 'on' ? await disablePush() : await enablePush())
      },
      { fallback: 'Couldn’t update notifications. Try again.' },
    )
  }

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

        <Divider sx={{ my: 5 }} />

        <Typography variant="h4" component="h2" sx={{ fontSize: '1.5rem', mb: 1 }}>
          Appearance
        </Typography>
        <Typography sx={{ color: 'text.secondary', mb: 2.5 }}>
          System matches your device&rsquo;s light or dark setting. Saved in this browser.
        </Typography>
        <ToggleButtonGroup
          exclusive
          color="primary"
          size="small"
          aria-label="Color theme"
          value={mode ?? 'system'}
          onChange={(_, next: 'light' | 'dark' | 'system' | null) => {
            if (next) setMode(next)
          }}
          sx={{ '& .MuiToggleButton-root': { gap: 1, px: 2, textTransform: 'none' } }}
        >
          <ToggleButton value="light">
            <LightModeOutlined fontSize="small" />
            Light
          </ToggleButton>
          <ToggleButton value="dark">
            <DarkModeOutlined fontSize="small" />
            Dark
          </ToggleButton>
          <ToggleButton value="system">
            <SettingsBrightnessOutlined fontSize="small" />
            System
          </ToggleButton>
        </ToggleButtonGroup>

        <Divider sx={{ my: 5 }} />

        <Typography variant="h4" component="h2" sx={{ fontSize: '1.5rem', mb: 1 }}>
          Notifications
        </Typography>
        <Typography sx={{ color: 'text.secondary', mb: 2.5 }}>
          {pushState === null ? 'Checking this device…' : pushDescriptions[pushState]}
        </Typography>
        {push.error !== null && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {push.error}
          </Alert>
        )}
        {(pushState === 'on' || pushState === 'off') && (
          <FormControlLabel
            control={
              <Switch checked={pushState === 'on'} onChange={togglePush} disabled={push.busy} />
            }
            label="Run notifications"
          />
        )}
      </Container>
    </Box>
  )
}
