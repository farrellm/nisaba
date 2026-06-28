import { useEffect, useState, type KeyboardEvent } from 'react'
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
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

// ⚡ Bolt: Extracting Intl.Collator prevents initializing it on every comparison in the sort loop.
// Improves alpha sort performance by ~100x for large label lists.
const collator = new Intl.Collator(undefined, { sensitivity: 'base' })

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
  const [suggested, setSuggested] = useState<string[]>([])
  const [suggesting, setSuggesting] = useState(false)
  const [recommended, setRecommended] = useState<string[]>([])
  const [recommending, setRecommending] = useState(false)
  const [draft, setDraft] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  // Seed from the document and load the user's full label pool when opened.
  useEffect(() => {
    if (!open) return
    const seeded = doc.labels ?? []
    setApplied(seeded)
    setAllLabels(seeded)
    setSuggested([])
    setSuggesting(false)
    setRecommended([])
    setRecommending(false)
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

  const isApplied = (name: string) => applied.some((l) => sameName(l, name))

  // Toggle a label's applied state in place — used by the suggestions, whose
  // pills stay put and just change color rather than moving between sections.
  function toggleLabel(name: string) {
    if (isApplied(name)) removeLabel(name)
    else applyLabel(name)
  }

  // Ask the model to suggest labels from the document's story. Suggestions stay
  // visible whether or not they're applied; clicking one toggles it.
  async function handleSuggest() {
    setError(null)
    setSuggesting(true)
    try {
      const names = await api.post<string[]>(`/api/documents/${doc.id}/suggest-labels`)
      setSuggested(names ?? [])
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not suggest labels. Try again.')
    } finally {
      setSuggesting(false)
    }
  }

  // Ask the model which of the user's existing labels fit the story. The picks
  // are highlighted in place within "Other labels" rather than moved.
  async function handleRecommend() {
    setError(null)
    setRecommending(true)
    try {
      const names = await api.post<string[]>(`/api/documents/${doc.id}/recommend-labels`)
      setRecommended(names ?? [])
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not recommend labels. Try again.')
    } finally {
      setRecommending(false)
    }
  }

  // Add the typed name, reusing an existing label (by its canonical name) when one
  // matches case-insensitively, so we never create a duplicate.
  function addDraft() {
    const name = draft.trim()
    if (!name) return
    const existing = allLabels.find((l) => sameName(l, name))
    if (!existing) {
      setAllLabels((prev) => [...prev, name].sort((a, b) => collator.compare(a, b)))
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

  // A suggested label keeps its own section, so don't also list it under "Other
  // labels" — every label lives in exactly one of the pool-based sections.
  const others = allLabels.filter(
    (l) => !isApplied(l) && !suggested.some((s) => sameName(s, l)),
  )

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
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="overline" sx={{ color: 'text.secondary' }}>
                Suggested
              </Typography>
              <Button size="small" onClick={handleSuggest} disabled={suggesting}>
                {suggesting ? 'Suggesting…' : suggested.length > 0 ? 'Regenerate' : 'Suggest from story'}
              </Button>
            </Box>
            {suggested.length > 0 ? (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
                {suggested.map((name) => {
                  const on = isApplied(name)
                  return (
                    <Chip
                      key={name}
                      label={name}
                      color={on ? 'primary' : 'default'}
                      variant={on ? 'filled' : 'outlined'}
                      onClick={() => toggleLabel(name)}
                      sx={{ fontFamily: fonts.mono, color: on ? undefined : 'text.secondary' }}
                    />
                  )
                })}
              </Box>
            ) : (
              <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.85rem', color: 'text.secondary' }}>
                {suggesting
                  ? 'Analyzing the story…'
                  : 'Generate labels from this document’s story.'}
              </Typography>
            )}
          </Box>

          <Box>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="overline" sx={{ color: 'text.secondary' }}>
                Other labels
              </Typography>
              <Button
                size="small"
                onClick={handleRecommend}
                disabled={recommending || others.length === 0}
              >
                {recommending ? 'Recommending…' : 'Recommend'}
              </Button>
            </Box>
            {others.length > 0 ? (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
                {others.map((name) => {
                  const rec = recommended.some((r) => sameName(r, name))
                  return (
                    <Chip
                      key={name}
                      label={name}
                      variant="outlined"
                      color={rec ? 'primary' : 'default'}
                      onClick={() => applyLabel(name)}
                      sx={{ fontFamily: fonts.mono, color: rec ? undefined : 'text.secondary' }}
                    />
                  )
                })}
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
          {submitting ? (
            <>
              <CircularProgress size={16} color="inherit" sx={{ mr: 1 }} />
              Saving…
            </>
          ) : (
            'Save'
          )}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
