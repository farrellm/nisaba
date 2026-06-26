import { useEffect, useState, type FormEvent } from 'react'
import {
  Alert,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Radio,
  RadioGroup,
  Typography,
} from '@mui/material'
import { ApiError } from '../api/client'
import type { Block, Mode } from '../api/types'
import { fonts } from '../theme'

interface AddBlockDialogProps {
  open: boolean
  modes: Mode[]
  onClose: () => void
  onCreate: (mode: string) => Promise<Block>
}

// AddBlockDialog lets the user pick a mode and append a block. The block's
// fields are seeded server-side from the document's attributes.
export default function AddBlockDialog({ open, modes, onClose, onCreate }: AddBlockDialogProps) {
  const [selected, setSelected] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (open && !selected && modes.length > 0) setSelected(modes[0].name)
  }, [open, selected, modes])

  function handleClose() {
    if (submitting) return
    setError(null)
    onClose()
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      await onCreate(selected)
      onClose()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong. Try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      fullWidth
      maxWidth="sm"
      slotProps={{ paper: { component: 'form', onSubmit: handleSubmit } }}
    >
      <DialogTitle>Add a block</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <RadioGroup value={selected} onChange={(e) => setSelected(e.target.value)}>
          {modes.map((mode) => (
            <FormControlLabel
              key={mode.name}
              value={mode.name}
              control={<Radio />}
              sx={{ alignItems: 'flex-start', py: 0.5 }}
              label={
                <span>
                  <Typography component="span" sx={{ fontFamily: fonts.display, fontSize: '1.05rem' }}>
                    {mode.label}
                  </Typography>
                  <Typography
                    sx={{
                      fontFamily: fonts.mono,
                      fontSize: '0.75rem',
                      color: 'text.secondary',
                    }}
                  >
                    {mode.keys.join(' · ')}
                  </Typography>
                </span>
              }
            />
          ))}
        </RadioGroup>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={submitting} sx={{ color: 'text.secondary' }}>
          Cancel
        </Button>
        <Button type="submit" variant="contained" disabled={submitting || !selected}>
          {submitting ? (
            <>
              <CircularProgress size={16} color="inherit" sx={{ mr: 1 }} />
              Adding…
            </>
          ) : (
            'Add block'
          )}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
