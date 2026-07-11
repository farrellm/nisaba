import { useEffect, useState, type FormEvent } from 'react'
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
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { stripPromptTag } from '../lib/text'
import { useAsyncAction } from '../lib/useAsyncAction'
import SubmitButton from './SubmitButton'
import type { Document, DocumentDetail, RedditPost } from '../api/types'

interface RedditPromptDialogProps {
  open: boolean
  post: RedditPost | null
  onClose: () => void
}

// RedditPromptDialog turns a Reddit post into a new document: the title starts
// empty and the prompt is seeded from the post title (with any "[WP]" tag
// stripped and the result trimmed). On submit it creates the document
// (url = the post permalink), merges in the "prompt" attribute, then redirects
// to the new document.
export default function RedditPromptDialog({ open, post, onClose }: RedditPromptDialogProps) {
  const navigate = useNavigate()
  const [title, setTitle] = useState('')
  const [prompt, setPrompt] = useState('')
  const { busy: submitting, error, setError, run } = useAsyncAction()

  // Seed the prompt from the post title (and clear the title) whenever the
  // dialog opens for a different post. Strip any "[WP]" tag (case-insensitive)
  // and trim surrounding whitespace.
  useEffect(() => {
    setTitle('')
    setPrompt(stripPromptTag(post?.title ?? ''))
    setError(null)
    // setError is a stable useState setter (via useAsyncAction).
  }, [post, setError])

  function handleClose() {
    if (submitting) return
    onClose()
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!post) return
    run(
      async () => {
        const doc = await api.post<Document>('/api/documents', {
          title,
          url: post.url,
        })
        await api.put<DocumentDetail>(`/api/documents/${doc.id}`, {
          attributes: { prompt },
        })
        navigate(`/documents/${doc.id}`)
      },
      { keepBusyOnSuccess: true },
    )
  }

  return (
    <Dialog open={open} onClose={handleClose} fullWidth maxWidth="sm">
      <form onSubmit={handleSubmit} noValidate>
        <DialogTitle>New document from prompt</DialogTitle>
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
            />
            <TextField
              label="Prompt"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              multiline
              minRows={3}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose} disabled={submitting} sx={{ color: 'text.secondary' }}>
            Cancel
          </Button>
          <SubmitButton busy={submitting} busyLabel="Creating…" disabled={!title.trim()}>
            Create
          </SubmitButton>
        </DialogActions>
      </form>
    </Dialog>
  )
}
