import { useState, type FormEvent } from 'react'
import { Alert, Box, Button, CircularProgress, Link as MuiLink, Stack, TextField, Typography } from '@mui/material'
import { Link as RouterLink, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { ApiError } from '../api/client'
import AuthLayout from '../components/AuthLayout'
import { usePageTitle } from '../lib/usePageTitle'

const MIN_PASSWORD = 8

export default function CreateUserPage() {
  usePageTitle('Sign up')
  const { user, register } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  if (user) return <Navigate to="/" replace />

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (password.length < MIN_PASSWORD) {
      setError(`Password must be at least ${MIN_PASSWORD} characters.`)
      return
    }
    if (password !== confirm) {
      setError('Passwords do not match.')
      return
    }
    setSubmitting(true)
    try {
      await register(username, password)
      navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong. Try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthLayout eyebrow="Create account">
      <Box component="form" onSubmit={handleSubmit} noValidate>
        <Typography variant="h4" sx={{ mb: 0.5 }}>
          Claim your table
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          Pick a username and a password to get started.
        </Typography>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        <Stack spacing={2}>
          <TextField
            label="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            autoFocus
            required
          />
          <TextField
            label="Password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="new-password"
            helperText={`At least ${MIN_PASSWORD} characters.`}
            required
          />
          <TextField
            label="Confirm password"
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            autoComplete="new-password"
            required
          />
          <Button type="submit" variant="contained" size="large" disabled={submitting}>
            {submitting ? (
              <>
                <CircularProgress size={16} color="inherit" sx={{ mr: 1 }} />
                Creating account…
              </>
            ) : (
              'Create account'
            )}
          </Button>
        </Stack>

        <Typography variant="body2" color="text.secondary" sx={{ mt: 3 }}>
          Already have an account?{' '}
          <MuiLink component={RouterLink} to="/login" underline="hover" sx={{ fontWeight: 600 }}>
            Sign in
          </MuiLink>
        </Typography>
      </Box>
    </AuthLayout>
  )
}
