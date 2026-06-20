import { useState, type FormEvent } from 'react'
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
} from '@mui/material'
import { api, ApiError } from '../api/client'
import type { RedditPost } from '../api/types'

interface RedditUrlDialogProps {
  open: boolean
  onClose: () => void
  onResolved: (post: RedditPost) => void
}

// RedditUrlDialog collects a Reddit post URL, fetches the post's title from the
// backend, then hands the resolved post off to the caller (which opens the
// create-document dialog).
export default function RedditUrlDialog({ open, onClose, onResolved }: RedditUrlDialogProps) {
  const [url, setUrl] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  function handleClose() {
    if (loading) return
    setUrl('')
    setError(null)
    onClose()
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const post = await api.get<RedditPost>(
        `/api/reddit/post?url=${encodeURIComponent(url.trim())}`,
      )
      setUrl('')
      onResolved(post)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong. Try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onClose={handleClose} fullWidth maxWidth="sm">
      <form onSubmit={handleSubmit} noValidate>
        <DialogTitle>Import a Reddit post</DialogTitle>
        <DialogContent>
          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label="Reddit post URL"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://www.reddit.com/r/…"
              autoFocus
              required
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose} disabled={loading} sx={{ color: 'text.secondary' }}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" disabled={loading || !url.trim()}>
            {loading ? 'Loading…' : 'Continue'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}
