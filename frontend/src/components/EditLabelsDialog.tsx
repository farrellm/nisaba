import { useEffect, useState, type KeyboardEvent } from 'react'
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { api, ApiError } from '../api/client'
import type { DocumentDetail } from '../api/types'
import { fonts } from '../theme'

interface EditLabelsDialogProps {
  open: boolean
  doc: DocumentDetail
  onClose: () => void
  onChange: (doc: DocumentDetail) => void
}

const sameName = (a: string, b: string) => a.toLowerCase() === b.toLowerCase()

// EditLabelsDialog edits which of the user's labels apply to a document. Edits are
// local (filed/shelved pills) until Save commits them with a single PUT. Typing a
// name that already exists reuses that label rather than creating a duplicate.
export default function EditLabelsDialog({ open, doc, onChange, onClose }: EditLabelsDialogProps) {
  const [applied, setApplied] = useState<string[]>([])
  const [allLabels, setAllLabels] = useState<string[]>([])
  const [draft, setDraft] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  // Seed from the document and load the user's full label pool when opened.
  useEffect(() => {
    if (!open) return
    const seeded = doc.labels ?? []
    setApplied(seeded)
    setAllLabels(seeded)
    setDraft('')
    setError(null)
    api
      .get<string[]>('/api/labels')
      .then((names) => setAllLabels(names ?? []))
      .catch(() => {
        /* keep the seeded labels; the document's own labels still toggle */
      })
  }, [open, doc])

  function handleClose() {
    if (submitting) return
    onClose()
  }

  function applyLabel(name: string) {
    setApplied((prev) => (prev.some((l) => sameName(l, name)) ? prev : [...prev, name]))
  }

  function removeLabel(name: string) {
    setApplied((prev) => prev.filter((l) => !sameName(l, name)))
  }

  // Add the typed name, reusing an existing label (by its canonical name) when one
  // matches case-insensitively, so we never create a duplicate.
  function addDraft() {
    const name = draft.trim()
    if (!name) return
    const existing = allLabels.find((l) => sameName(l, name))
    if (!existing) {
      setAllLabels((prev) => [...prev, name].sort((a, b) => a.localeCompare(b)))
    }
    applyLabel(existing ?? name)
    setDraft('')
  }

  function handleDraftKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      addDraft()
    }
  }

  async function handleSave() {
    setError(null)
    setSubmitting(true)
    try {
      const updated = await api.put<DocumentDetail>(`/api/documents/${doc.id}`, { labels: applied })
      onChange(updated)
      onClose()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong. Try again.')
    } finally {
      setSubmitting(false)
    }
  }

  const others = allLabels.filter((l) => !applied.some((a) => sameName(a, l)))

  return (
    <Dialog open={open} onClose={handleClose} fullWidth maxWidth="sm">
      <DialogTitle>Edit labels</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <Stack spacing={3} sx={{ mt: 1 }}>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start' }}>
            <TextField
              label="New label"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={handleDraftKeyDown}
              autoFocus
            />
            <Button
              onClick={addDraft}
              variant="outlined"
              disabled={!draft.trim()}
              sx={{ flexShrink: 0, mt: 1 }}
            >
              Add
            </Button>
          </Box>

          <Box>
            <Typography variant="overline" sx={{ color: 'text.secondary', display: 'block', mb: 1 }}>
              On this document
            </Typography>
            {applied.length > 0 ? (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
                {applied.map((name) => (
                  <Chip
                    key={name}
                    label={name}
                    color="primary"
                    onDelete={() => removeLabel(name)}
                    sx={{ fontFamily: fonts.mono }}
                  />
                ))}
              </Box>
            ) : (
              <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.85rem', color: 'text.secondary' }}>
                No labels on this document yet.
              </Typography>
            )}
          </Box>

          <Box>
            <Typography variant="overline" sx={{ color: 'text.secondary', display: 'block', mb: 1 }}>
              Other labels
            </Typography>
            {others.length > 0 ? (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
                {others.map((name) => (
                  <Chip
                    key={name}
                    label={name}
                    variant="outlined"
                    onClick={() => applyLabel(name)}
                    sx={{ fontFamily: fonts.mono, color: 'text.secondary' }}
                  />
                ))}
              </Box>
            ) : (
              <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.85rem', color: 'text.secondary' }}>
                No other labels. Add one above.
              </Typography>
            )}
          </Box>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={submitting} sx={{ color: 'text.secondary' }}>
          Cancel
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={submitting}>
          {submitting ? 'Saving…' : 'Save'}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
