import { useState } from 'react'
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
} from '@mui/material'
import { api } from '../api/client'
import type { LabelRenameResult } from '../api/types'
import { sameName } from '../lib/text'
import { useAsyncAction } from '../lib/useAsyncAction'
import { fonts } from '../theme'

interface RenameLabelDialogProps {
  name: string
  docCount: number
  otherNames: string[]
  onClose: () => void
  onDone: () => void
}

// RenameLabelDialog renames a label everywhere. Typing the name of another existing
// label warns that the two will be merged, and the confirm button becomes "Merge".
export default function RenameLabelDialog({
  name,
  docCount,
  otherNames,
  onClose,
  onDone,
}: RenameLabelDialogProps) {
  const [draft, setDraft] = useState(name)
  const { busy: submitting, error, run } = useAsyncAction()

  const trimmed = draft.trim()
  const willMerge = trimmed !== '' && otherNames.some((n) => sameName(n, trimmed))
  // Exact (case-sensitive) compare so a case-only rename (noir → Noir) is allowed.
  const unchanged = trimmed === '' || trimmed === name

  function handleSave() {
    run(
      async () => {
        await api.put<LabelRenameResult>('/api/labels', { name, newName: trimmed })
        onDone()
      },
      { fallback: 'Could not rename the label. Try again.', keepBusyOnSuccess: true },
    )
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
