import { useState, type FormEvent } from 'react'
import {
  Alert,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
} from '@mui/material'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { useAsyncAction } from '../lib/useAsyncAction'
import type { Document } from '../api/types'

interface NewDocumentDialogProps {
  open: boolean
  onClose: () => void
}

// NewDocumentDialog prompts for a title (required) and url (optional), creates
// the document, then redirects to its page.
export default function NewDocumentDialog({ open, onClose }: NewDocumentDialogProps) {
  const navigate = useNavigate()
  const [title, setTitle] = useState('')
  const [url, setUrl] = useState('')
  const { busy: submitting, error, setError, run } = useAsyncAction()

  function handleClose() {
    if (submitting) return
    setTitle('')
    setUrl('')
    setError(null)
    onClose()
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    run(async () => {
      const doc = await api.post<Document>('/api/documents', {
        title,
        url: url.trim() || null,
      })
      navigate(`/documents/${doc.id}`)
    })
  }

  return (
    <Dialog open={open} onClose={handleClose} fullWidth maxWidth="sm">
      <form onSubmit={handleSubmit} noValidate>
        <DialogTitle>New document</DialogTitle>
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
              label="URL"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://… (optional)"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose} disabled={submitting} sx={{ color: 'text.secondary' }}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" disabled={submitting || !title.trim()}>
            {submitting ? (
              <>
                <CircularProgress size={16} color="inherit" sx={{ mr: 1 }} />
                Creating…
              </>
            ) : (
              'Create'
            )}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}
