import { useEffect, useState, type FormEvent } from 'react'
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  InputAdornment,
  Stack,
  TextField,
  CircularProgress,
} from '@mui/material'
import { api } from '../api/client'
import { stripPromptTag } from '../lib/text'
import { useAsyncAction } from '../lib/useAsyncAction'
import SubmitButton from './SubmitButton'
import type { DocumentDetail, RedditPost } from '../api/types'

interface RedditSubmitDialogProps {
  open: boolean
  doc: DocumentDetail
  onClose: () => void
  // Called with the refreshed document after a successful post, so the parent
  // can reflect the newly-saved post URL.
  onPosted: (doc: DocumentDetail) => void
}

// RedditSubmitDialog publishes the document's story back to Reddit as a self
// post. The body is seeded from the "story" attribute, prefixed (once the
// original prompt is fetched) with a credit link to it and its author. The title
// is derived, best-effort, from the original prompt at doc.url: fetch its title, strip any
// "[WP]" tag, trim, and prefix "[PI] " (WritingPrompts -> Prompt Inspired). Both
// fields stay editable. Submitting posts to the user's configured subreddit via
// POST /api/documents/:id/reddit-submit, which saves the resulting permalink on
// the document and returns the refreshed document; on success the dialog closes
// (the saved permalink surfaces as a "Posted ↗" link in the document header).
export default function RedditSubmitDialog({
  open,
  doc,
  onClose,
  onPosted,
}: RedditSubmitDialogProps) {
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [titleLoading, setTitleLoading] = useState(false)
  const { busy: submitting, error, setError, run } = useAsyncAction()

  // Seed the fields each time the dialog opens. The story is read directly; the
  // title is fetched from the original prompt and may arrive a moment later.
  useEffect(() => {
    if (!open) return
    setBody(doc.attributes?.story ?? '')
    setTitle('')
    setError(null)

    if (!doc.url) return
    const url = doc.url
    let cancelled = false
    setTitleLoading(true)
    api
      .get<RedditPost>(`/api/reddit/post?url=${encodeURIComponent(url)}`)
      .then((post) => {
        if (cancelled) return
        const stripped = stripPromptTag(post.title)
        setTitle(`[PI] ${stripped}`)
        // Credit the original prompt above the story.
        const credit = `[Original post](${url}) by u/${post.author}.\n\n---\n\n`
        setBody(credit + (doc.attributes?.story ?? ''))
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
    // setError is a stable useState setter (via useAsyncAction).
  }, [open, doc, setError])

  function handleClose() {
    if (submitting) return
    onClose()
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    run(async () => {
      const updated = await api.post<DocumentDetail>(`/api/documents/${doc.id}/reddit-submit`, {
        title,
        body,
      })
      onPosted(updated)
      onClose()
    })
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
              // Cap growth (the story can be long) so the field scrolls
              // internally and the dialog's action buttons stay on screen.
              maxRows={12}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose} disabled={submitting} sx={{ color: 'text.secondary' }}>
            Cancel
          </Button>
          <SubmitButton busy={submitting} busyLabel="Posting…" disabled={!title.trim()}>
            Post
          </SubmitButton>
        </DialogActions>
      </form>
    </Dialog>
  )
}
