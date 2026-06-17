import { useEffect, useState } from 'react'
import { ThemeProvider, CssBaseline, Container, Typography, Box, Chip } from '@mui/material'
import theme from './theme'

interface HealthResponse {
  status: string
  db: string
}

function App() {
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch('/api/healthz')
      .then((r) => r.json())
      .then(setHealth)
      .catch((e: unknown) => setError(String(e)))
  }, [])

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Container maxWidth="sm">
        <Box sx={{ mt: 8, textAlign: 'center' }}>
          <Typography variant="h3" gutterBottom>
            Nisaba
          </Typography>
          <Typography variant="body1" color="text.secondary" gutterBottom>
            Skeleton app
          </Typography>
          {error && <Chip label={`Error: ${error}`} color="error" />}
          {health && (
            <Box sx={{ mt: 2 }}>
              <Chip label={`API: ${health.status}`} color="success" sx={{ mr: 1 }} />
              <Chip
                label={`DB: ${health.db}`}
                color={health.db === 'ok' ? 'success' : 'warning'}
              />
            </Box>
          )}
        </Box>
      </Container>
    </ThemeProvider>
  )
}

export default App
