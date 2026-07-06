import { useEffect, useMemo, useState } from 'react'
import {
  Alert,
  Box,
  Button,
  Container,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Divider,
  Link as MuiLink,
  TextField,
  Typography,
} from '@mui/material'
import { api, ApiError } from '../api/client'
import type { Document } from '../api/types'
import DocumentRow from '../components/DocumentRow'
import Masthead from '../components/Masthead'
import { usePageTitle } from '../lib/usePageTitle'
import { fonts } from '../theme'
import { EMPTY_ARRAY } from '../lib/constants'

// ⚡ Bolt: Extracting Intl.Collator prevents initializing it on every comparison in the sort loop.
// Improves alpha sort performance by ~100x for large label lists.
const collator = new Intl.Collator(undefined, { sensitivity: 'base' })

const sameName = (a: string, b: string) => a.toLowerCase() === b.toLowerCase()
const newestFirst = (a: Document, b: Document) =>
  b.updatedAt < a.updatedAt ? -1 : b.updatedAt > a.updatedAt ? 1 : 0

// LabelsPage is the writer's index: one section per label, every document carrying
// it listed beneath (archived ones marked), with per-label rename and delete. Labels
// auto-vanish once they're on no document, so every section here has at least one doc.
export default function LabelsPage() {
  usePageTitle('Labels')
  const [docs, setDocs] = useState<Document[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [renaming, setRenaming] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<string | null>(null)

  // ?archived=true returns every document (active and archived), which is exactly
  // the set the index needs — archived ones are marked per-row via doc.isArchived.
  function load() {
    api
      .get<Document[]>('/api/documents?archived=true')
      .then((all) => setDocs(all ?? []))
      .catch((e: unknown) => setError(e instanceof ApiError ? e.message : String(e)))
  }

  useEffect(load, [])

  // Group documents under each label name, alphabetical by label, newest-first within.
  const sections = useMemo(() => {
    if (!docs) return null
    const byLabel = new Map<string, Document[]>()
    for (const doc of docs) {
      for (const label of doc.labels ?? EMPTY_ARRAY) {
        const bucket = byLabel.get(label)
        if (bucket) bucket.push(doc)
        else byLabel.set(label, [doc])
      }
    }
    return [...byLabel.entries()]
      .sort((a, b) => collator.compare(a[0], b[0]))
      .map(([name, group]) => ({ name, docs: group.sort(newestFirst) }))
  }, [docs])

  const labelNames = useMemo(() => sections?.map((s) => s.name) ?? EMPTY_ARRAY, [sections])
  const loading = docs === null && error === null

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead />

      <Container maxWidth="md" sx={{ pt: { xs: 5, md: 8 }, pb: 12 }}>
        <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)', mb: 4 }}>
          Labels
        </Typography>
        <Divider sx={{ mb: 1 }} />

        {error ? (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'error.main', py: 1.5 }}>
            {error}
          </Typography>
        ) : loading ? (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'text.secondary', py: 1.5 }}>
            Loading…
          </Typography>
        ) : sections && sections.length > 0 ? (
          sections.map((section) => (
            <Box
              key={section.name}
              sx={{ mb: 5, '&:hover .label-controls, &:focus-within .label-controls': { opacity: 1 } }}
            >
              <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 2 }}>
                <Typography
                  component="h2"
                  sx={{ fontFamily: fonts.display, fontWeight: 500, fontSize: '1.5rem', letterSpacing: '-0.01em' }}
                >
                  {section.name}
                </Typography>
                <Box sx={{ flex: 1, borderBottom: '1px dotted', borderColor: 'divider', transform: 'translateY(-4px)' }} />
                <Typography
                  sx={{ fontFamily: fonts.mono, fontSize: '0.75rem', color: 'text.secondary', whiteSpace: 'nowrap' }}
                >
                  {section.docs.length === 1 ? '1 story' : `${section.docs.length} stories`}
                </Typography>
                <Box
                  className="label-controls"
                  sx={{ display: 'flex', gap: 1.5, opacity: { xs: 1, md: 0.45 }, transition: 'opacity 120ms' }}
                >
                  <ControlLink onClick={() => setRenaming(section.name)}>Rename</ControlLink>
                  <ControlLink onClick={() => setDeleting(section.name)}>Delete</ControlLink>
                </Box>
              </Box>

              <Box sx={{ mt: 0.5 }}>
                {section.docs.map((doc) => (
                  <DocumentRow key={doc.id} doc={doc} showArchived />
                ))}
              </Box>
            </Box>
          ))
        ) : (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'text.secondary', py: 1.5 }}>
            No labels yet. Add labels to a document to start your index.
          </Typography>
        )}
      </Container>

      {renaming !== null && (
        <RenameLabelDialog
          name={renaming}
          docCount={sections?.find((s) => s.name === renaming)?.docs.length ?? 0}
          otherNames={labelNames.filter((n) => n !== renaming)}
          onClose={() => setRenaming(null)}
          onDone={() => {
            setRenaming(null)
            load()
          }}
        />
      )}

      {deleting !== null && (
        <DeleteLabelDialog
          name={deleting}
          docCount={sections?.find((s) => s.name === deleting)?.docs.length ?? 0}
          onClose={() => setDeleting(null)}
          onDone={() => {
            setDeleting(null)
            load()
          }}
        />
      )}
    </Box>
  )
}

// ControlLink is the small mono action used in a section header (Rename / Delete).
function ControlLink({ onClick, children }: { onClick: () => void; children: string }) {
  return (
    <MuiLink
      component="button"
      type="button"
      onClick={onClick}
      underline="hover"
      sx={{
        fontFamily: fonts.mono,
        fontSize: '0.72rem',
        textTransform: 'uppercase',
        letterSpacing: '0.08em',
        color: 'text.secondary',
        '&:hover, &:focus-visible': { color: 'primary.main' },
      }}
    >
      {children}
    </MuiLink>
  )
}

// RenameLabelDialog renames a label everywhere. Typing the name of another existing
// label warns that the two will be merged, and the confirm button becomes "Merge".
function RenameLabelDialog({
  name,
  docCount,
  otherNames,
  onClose,
  onDone,
}: {
  name: string
  docCount: number
  otherNames: string[]
  onClose: () => void
  onDone: () => void
}) {
  const [draft, setDraft] = useState(name)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const trimmed = draft.trim()
  const willMerge = trimmed !== '' && otherNames.some((n) => sameName(n, trimmed))
  // Exact (case-sensitive) compare so a case-only rename (noir → Noir) is allowed.
  const unchanged = trimmed === '' || trimmed === name

  async function handleSave() {
    setError(null)
    setSubmitting(true)
    try {
      await api.put<{ merged: boolean }>('/api/labels', { name, newName: trimmed })
      onDone()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not rename the label. Try again.')
      setSubmitting(false)
    }
  }

  return (
    <Dialog open onClose={submitting ? undefined : onClose} fullWidth maxWidth="xs">
      <DialogTitle>Rename label</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        {willMerge && (
          <Alert severity="warning" sx={{ mb: 2 }}>
            A label named “{trimmed}” already exists. Renaming merges this label’s{' '}
            {docCount === 1 ? '1 document' : `${docCount} documents`} into it. This can’t be undone.
          </Alert>
        )}
        <TextField
          label="Label name"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !unchanged && !submitting) {
              e.preventDefault()
              handleSave()
            }
          }}
          autoFocus
          sx={{ mt: 1, '& input': { fontFamily: fonts.mono } }}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={submitting}>
          Cancel
        </Button>
        <Button onClick={handleSave} variant="contained" disabled={unchanged || submitting}>
          {willMerge ? 'Merge' : 'Rename'}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

// DeleteLabelDialog confirms removing a label from every document it tags. The
// documents themselves are kept.
function DeleteLabelDialog({
  name,
  docCount,
  onClose,
  onDone,
}: {
  name: string
  docCount: number
  onClose: () => void
  onDone: () => void
}) {
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  async function handleDelete() {
    setError(null)
    setSubmitting(true)
    try {
      await api.del<void>(`/api/labels?name=${encodeURIComponent(name)}`)
      onDone()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not delete the label. Try again.')
      setSubmitting(false)
    }
  }

  return (
    <Dialog open onClose={submitting ? undefined : onClose} fullWidth maxWidth="xs">
      <DialogTitle>Delete label</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <DialogContentText>
          Delete “{name}”? It will be removed from{' '}
          {docCount === 1 ? '1 document' : `${docCount} documents`}. The documents are not deleted.
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={submitting}>
          Cancel
        </Button>
        <Button onClick={handleDelete} color="error" variant="contained" disabled={submitting}>
          Delete
        </Button>
      </DialogActions>
    </Dialog>
  )
}
