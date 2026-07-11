import { useEffect, useState, type FormEvent } from 'react'
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Radio,
  RadioGroup,
  Typography,
} from '@mui/material'
import { useAsyncAction } from '../lib/useAsyncAction'
import SubmitButton from './SubmitButton'
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
  const { busy: submitting, error, setError, run } = useAsyncAction()

  useEffect(() => {
    if (open && !selected && modes.length > 0) setSelected(modes[0].name)
  }, [open, selected, modes])

  function handleClose() {
    if (submitting) return
    setError(null)
    onClose()
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    run(async () => {
      await onCreate(selected)
      onClose()
    })
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
              control={<Radio inputProps={{ 'aria-label': `Select mode ${mode.label}` }} />}
              sx={{ alignItems: 'flex-start', py: 0.5 }}
              label={
                <span>
                  <Typography
                    component="span"
                    sx={{ fontFamily: fonts.display, fontSize: '1.05rem' }}
                  >
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
        <SubmitButton busy={submitting} busyLabel="Adding…" disabled={!selected}>
          Add block
        </SubmitButton>
      </DialogActions>
    </Dialog>
  )
}
