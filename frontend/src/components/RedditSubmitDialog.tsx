import { useEffect, useState, type FormEvent } from 'react'
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  InputAdornment,
  Link as MuiLink,
  Stack,
  TextField,
  CircularProgress,
} from '@mui/material'
import { api, ApiError } from '../api/client'
import type { DocumentDetail, RedditPost } from '../api/types'

interface RedditSubmitDialogProps {
  open: boolean
  doc: DocumentDetail
  onClose: () => void
}

// RedditSubmitDialog publishes the document's story back to Reddit as a self
// post. The body is seeded from the "story" attribute. The title is derived,
// best-effort, from the original prompt at doc.url: fetch its title, strip any
// "[WP]" tag, trim, and prefix "[PI] " (WritingPrompts -> Prompt Inspired). Both
// fields stay editable. Submitting posts to the user's configured subreddit via
// POST /api/reddit/submit.
export default function RedditSubmitDialog({ open, doc, onClose }: RedditSubmitDialogProps) {
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [titleLoading, setTitleLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [postedUrl, setPostedUrl] = useState<string | null>(null)

  // Seed the fields each time the dialog opens. The story is read directly; the
  // title is fetched from the original prompt and may arrive a moment later.
  useEffect(() => {
    if (!open) return
    setBody(doc.attributes?.story ?? '')
    setTitle('')
    setError(null)
    setPostedUrl(null)

    if (!doc.url) return
    let cancelled = false
    setTitleLoading(true)
    api
      .get<RedditPost>(`/api/reddit/post?url=${encodeURIComponent(doc.url)}`)
      .then((post) => {
        if (cancelled) return
        const stripped = post.title.replace(/\[wp\]/gi, '').trim()
        setTitle(`[PI] ${stripped}`)
      })
      .catch(() => {
        // Best-effort: leave the title blank for the user to fill in.
      })
      .finally(() => {
        if (!cancelled) setTitleLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, doc])

  function handleClose() {
    if (submitting) return
    onClose()
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const result = await api.post<{ url: string }>('/api/reddit/submit', {
        title,
        body,
      })
      setPostedUrl(result.url)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong. Try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onClose={handleClose} fullWidth maxWidth="sm">
      <form onSubmit={handleSubmit} noValidate>
        <DialogTitle>Post to Reddit</DialogTitle>
        <DialogContent>
          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}
          {postedUrl ? (
            <Alert severity="success" sx={{ mt: 1 }}>
              Posted to Reddit.{' '}
              <MuiLink href={postedUrl} target="_blank" rel="noopener noreferrer">
                View post ↗
              </MuiLink>
            </Alert>
          ) : (
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField
                label="Title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                autoFocus
                required
                InputProps={{
                  endAdornment: titleLoading ? (
                    <InputAdornment position="end">
                      <CircularProgress size={18} />
                    </InputAdornment>
                  ) : undefined,
                }}
              />
              <TextField
                label="Body"
                value={body}
                onChange={(e) => setBody(e.target.value)}
                multiline
                minRows={6}
              />
            </Stack>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose} disabled={submitting} sx={{ color: 'text.secondary' }}>
            {postedUrl ? 'Done' : 'Cancel'}
          </Button>
          {!postedUrl && (
            <Button type="submit" variant="contained" disabled={submitting || !title.trim()}>
              {submitting ? 'Posting…' : 'Post'}
            </Button>
          )}
        </DialogActions>
      </form>
    </Dialog>
  )
}
