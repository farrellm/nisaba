import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'
import { api } from '../api/client'
import { useAsyncAction } from '../lib/useAsyncAction'

interface DeleteLabelDialogProps {
  name: string
  docCount: number
  onClose: () => void
  onDone: () => void
}

// DeleteLabelDialog confirms removing a label from every document it tags. The
// documents themselves are kept.
export default function DeleteLabelDialog({
  name,
  docCount,
  onClose,
  onDone,
}: DeleteLabelDialogProps) {
  const { busy: submitting, error, run } = useAsyncAction()

  function handleDelete() {
    run(
      async () => {
        await api.del<void>(`/api/labels?name=${encodeURIComponent(name)}`)
        onDone()
      },
      { fallback: 'Could not delete the label. Try again.', keepBusyOnSuccess: true },
    )
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
