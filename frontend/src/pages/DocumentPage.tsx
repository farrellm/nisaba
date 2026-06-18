import { Box, Button, Container, Link as MuiLink, Stack, Typography } from '@mui/material'
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { fonts } from '../theme'

// DocumentPage is a stub for now — it will eventually load and render the full
// document via GET /api/documents/:id.
export default function DocumentPage() {
  const { id } = useParams()
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
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
          <MuiLink
            component={RouterLink}
            to="/documents"
            underline="hover"
            sx={{ fontFamily: fonts.mono, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.08em' }}
          >
            Documents
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

      <Container maxWidth="md" sx={{ pt: { xs: 7, md: 12 }, pb: 8 }}>
        <Typography variant="overline" sx={{ color: 'primary.main', display: 'block', mb: 2 }}>
          Document {id}
        </Typography>
        <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)', mb: 2 }}>
          Coming soon
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ maxWidth: 480 }}>
          This document's editor hasn't been built yet.
        </Typography>
      </Container>
    </Box>
  )
}
