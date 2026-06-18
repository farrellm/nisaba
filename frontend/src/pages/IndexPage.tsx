import { useEffect, useState } from 'react'
import { Box, Button, Container, Divider, Stack, Typography } from '@mui/material'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { api } from '../api/client'
import { fonts } from '../theme'

interface HealthResponse {
  status: string
  db: string
}

// One row of the status ledger: a mono label, a hairline rule, and the value
// with a small state dot.
function LedgerRow({ label, value, ok }: { label: string; value: string; ok: boolean }) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 2, py: 1.5 }}>
      <Typography variant="overline" sx={{ color: 'text.secondary', minWidth: 96 }}>
        {label}
      </Typography>
      <Box sx={{ flex: 1, borderBottom: '1px dotted', borderColor: 'divider', transform: 'translateY(-3px)' }} />
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Box
          sx={{
            width: 8,
            height: 8,
            borderRadius: '50%',
            bgcolor: ok ? 'primary.main' : 'warning.main',
          }}
        />
        <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem' }}>{value}</Typography>
      </Box>
    </Box>
  )
}

export default function IndexPage() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .get<HealthResponse>('/api/healthz')
      .then(setHealth)
      .catch((e: unknown) => setError(String(e)))
  }, [])

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
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
        <Typography sx={{ fontFamily: fonts.display, fontWeight: 600, fontSize: '1.5rem', letterSpacing: '-0.02em' }}>
          Nisaba
        </Typography>
        <Stack direction="row" spacing={2} alignItems="center">
          <Typography variant="overline" sx={{ color: 'text.secondary' }}>
            {user?.username}
          </Typography>
          <Button variant="text" size="small" onClick={handleLogout} sx={{ color: 'text.primary' }}>
            Log out
          </Button>
        </Stack>
      </Box>

      <Container maxWidth="md" sx={{ pt: { xs: 7, md: 12 }, pb: 8 }}>
        <Typography variant="overline" sx={{ color: 'primary.main', display: 'block', mb: 2 }}>
          The scribe's table
        </Typography>
        <Typography
          variant="h1"
          sx={{ fontSize: 'clamp(2.75rem, 7vw, 4.5rem)', mb: 2 }}
        >
          Good to see you,
          <br />
          {user?.username}.
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ maxWidth: 480 }}>
          Your documents will live here. Nothing's been written yet — this is where it begins.
        </Typography>

        {/* Status ledger */}
        <Box sx={{ mt: 8, maxWidth: 520 }}>
          <Typography variant="overline" sx={{ color: 'text.secondary' }}>
            System status
          </Typography>
          <Divider sx={{ mt: 1, mb: 1 }} />
          {error ? (
            <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'error.main', py: 1.5 }}>
              {error}
            </Typography>
          ) : (
            <>
              <LedgerRow label="API" value={health?.status ?? '…'} ok={health?.status === 'ok'} />
              <LedgerRow label="Database" value={health?.db ?? '…'} ok={health?.db === 'ok'} />
            </>
          )}
        </Box>
      </Container>
    </Box>
  )
}
