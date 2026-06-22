import { useEffect, useRef, useState } from 'react'
import {
  Box,
  CircularProgress,
  IconButton,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import DeleteIcon from '@mui/icons-material/Delete'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import ReplayIcon from '@mui/icons-material/Replay'
import SaveOutlinedIcon from '@mui/icons-material/SaveOutlined'
import { api, ApiError } from '../api/client'
import type { Block, Mode } from '../api/types'
import { fonts } from '../theme'

interface BlockCardProps {
  block: Block
  mode: Mode | undefined
  onBlockUpdated: (block: Block) => void
  onBlockDeleted: (id: number) => void
  onAfterRun: () => void
  defaultOpen?: boolean
}

// BlockCard renders one block: its mode, editable key/values, a run action, and
// the responses produced so far. The body is a collapsible <details>; the mode
// header is the always-visible <summary>.
export default function BlockCard({ block, mode, onBlockUpdated, onBlockDeleted, onAfterRun, defaultOpen }: BlockCardProps) {
  const keys = mode?.keys ?? Object.keys(block.attributes)
  const [values, setValues] = useState<Record<string, string>>(() => {
    const seed: Record<string, string> = {}
    for (const key of keys) seed[key] = block.attributes[key] ?? ''
    return seed
  })
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [copying, setCopying] = useState(false)
  const [running, setRunning] = useState(false)
  const [armed, setArmed] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [reparsingId, setReparsingId] = useState<number | null>(null)
  const armedTimer = useRef<ReturnType<typeof setTimeout>>()

  const dirty = keys.some((key) => (values[key] ?? '') !== (block.attributes[key] ?? ''))
  const busy = saving || copying || running || deleting || reparsingId !== null

  // Clear the pending revert timer if the card unmounts.
  useEffect(() => () => clearTimeout(armedTimer.current), [])

  async function handleSave() {
    setError(null)
    setSaving(true)
    try {
      const updated = await api.put<Block>(
        `/api/documents/${block.documentId}/blocks/${block.id}`,
        { attributes: values },
      )
      onBlockUpdated(updated)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save. Try again.')
    } finally {
      setSaving(false)
    }
  }

  async function handleCopy() {
    setError(null)
    setCopying(true)
    try {
      const updated = await api.post<Block>(
        `/api/documents/${block.documentId}/blocks/${block.id}/copy`,
        { attributes: values },
      )
      onBlockUpdated(updated)
      onAfterRun()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not copy. Try again.')
    } finally {
      setCopying(false)
    }
  }

  // Deleting is a two-step action: the first click arms the control (and starts
  // a timer to disarm it), the second confirms. Both clicks must not toggle the
  // surrounding <details>, so the caller stops propagation.
  function handleDeleteClick() {
    if (!armed) {
      setArmed(true)
      armedTimer.current = setTimeout(() => setArmed(false), 4000)
      return
    }
    clearTimeout(armedTimer.current)
    handleDelete()
  }

  async function handleDelete() {
    setError(null)
    setDeleting(true)
    try {
      await api.del(`/api/documents/${block.documentId}/blocks/${block.id}`)
      onBlockDeleted(block.id)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not delete. Try again.')
      setArmed(false)
      setDeleting(false)
    }
  }

  async function handleRun() {
    setError(null)
    setRunning(true)
    try {
      const updated = await api.post<Block>(
        `/api/documents/${block.documentId}/blocks/${block.id}/run`,
        { attributes: values },
      )
      onBlockUpdated(updated)
      onAfterRun()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not run. Try again.')
    } finally {
      setRunning(false)
    }
  }

  // Re-derive the document's shared attributes from a stored response, without
  // calling the model again. Same refresh path as a run.
  async function handleReparse(responseId: number) {
    setError(null)
    setReparsingId(responseId)
    try {
      const updated = await api.post<Block>(
        `/api/documents/${block.documentId}/blocks/${block.id}/responses/${responseId}/reparse`,
      )
      onBlockUpdated(updated)
      onAfterRun()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not re-parse. Try again.')
    } finally {
      setReparsingId(null)
    }
  }

  return (
    <Box component="section" sx={{ pt: 4 }}>
      <Box
        component="details"
        {...(defaultOpen ? { open: true } : {})}
        sx={{ '&[open]': { borderBottom: '1px dotted', borderColor: 'divider', pb: 4 } }}
      >
        <Box
          component="summary"
          sx={{
            display: 'flex',
            alignItems: 'baseline',
            gap: 2,
            mb: 2,
            cursor: 'pointer',
            listStyle: 'none',
            '&::-webkit-details-marker': { display: 'none' },
          }}
        >
          <Typography
            variant="overline"
            sx={{ fontFamily: fonts.mono, color: 'primary.main', whiteSpace: 'nowrap' }}
          >
            {mode?.label ?? block.mode}
          </Typography>
          <Box sx={{ flex: 1, borderBottom: '1px dotted', borderColor: 'divider', transform: 'translateY(-3px)' }} />
          <Tooltip title={deleting ? '' : armed ? 'Click again to confirm' : 'Delete block'}>
            <span>
              <IconButton
                size="small"
                edge="end"
                color={armed ? 'error' : undefined}
                disabled={busy && !deleting}
                aria-label={armed ? 'Confirm delete' : 'Delete block'}
                onClick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  handleDeleteClick()
                }}
                sx={{ transform: 'translateY(-2px)', ...(armed ? {} : { color: 'text.disabled', '&:hover': { color: 'error.main' } }) }}
              >
                {deleting ? (
                  <CircularProgress size={18} color="error" />
                ) : armed ? (
                  <DeleteIcon fontSize="small" />
                ) : (
                  <DeleteOutlineIcon fontSize="small" />
                )}
              </IconButton>
            </span>
          </Tooltip>
        </Box>

        <Stack spacing={2}>
          {keys.map((key) => (
            <TextField
              key={key}
              label={key}
              value={values[key] ?? ''}
              onChange={(e) => setValues((v) => ({ ...v, [key]: e.target.value }))}
              multiline
              minRows={1}
            />
          ))}
        </Stack>

        {error && (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.8rem', color: 'error.main', mt: 1.5 }}>
            {error}
          </Typography>
        )}

        <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 2 }}>
          <Tooltip title="Save">
            <span>
              <IconButton
                size="small"
                onClick={handleSave}
                disabled={!dirty || busy}
                aria-label="Save"
                sx={{ color: 'text.disabled', '&:hover': { color: 'primary.main' } }}
              >
                {saving ? <CircularProgress size={18} /> : <SaveOutlinedIcon fontSize="small" />}
              </IconButton>
            </span>
          </Tooltip>
          <Tooltip title="Copy to document">
            <span>
              <IconButton
                size="small"
                onClick={handleCopy}
                disabled={busy}
                aria-label="Copy to document"
                sx={{ color: 'text.disabled', '&:hover': { color: 'primary.main' } }}
              >
                {copying ? <CircularProgress size={18} /> : <ContentCopyIcon fontSize="small" />}
              </IconButton>
            </span>
          </Tooltip>
          <Box sx={{ flexGrow: 1 }} />
          <Tooltip title="Run">
            <span>
              <IconButton
                size="small"
                onClick={handleRun}
                disabled={busy}
                aria-label="Run"
                sx={{
                  bgcolor: 'primary.main',
                  color: 'primary.contrastText',
                  '&:hover': { bgcolor: 'primary.dark' },
                  '&.Mui-disabled': { bgcolor: 'action.disabledBackground', color: 'action.disabled' },
                }}
              >
                {running ? <CircularProgress size={18} color="inherit" /> : <PlayArrowIcon fontSize="small" />}
              </IconButton>
            </span>
          </Tooltip>
        </Stack>

        {(block.responses ?? []).length > 0 && (
          <Stack spacing={1.5} sx={{ mt: 3 }}>
            {(block.responses ?? []).slice().reverse().map((response, idx) => (
              <Box key={response.id} component="details" {...(idx === 0 ? { open: true } : {})}>
                <Box
                  component="summary"
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1,
                    cursor: 'pointer',
                    listStyle: 'none',
                    '&::-webkit-details-marker': { display: 'none' },
                    mb: 1,
                  }}
                >
                  <Typography
                    variant="overline"
                    sx={{ fontFamily: fonts.mono, color: 'text.secondary', fontSize: '0.7rem' }}
                  >
                    {response.model || 'no model'}
                  </Typography>
                  <Box sx={{ flexGrow: 1 }} />
                  <Tooltip title="Re-parse into document attributes">
                    <span>
                      <IconButton
                        size="small"
                        disabled={busy}
                        aria-label="Re-parse response into document attributes"
                        onClick={(e) => {
                          e.preventDefault()
                          handleReparse(response.id)
                        }}
                        sx={{ color: 'text.disabled', '&:hover': { color: 'primary.main' } }}
                      >
                        {reparsingId === response.id ? (
                          <CircularProgress size={18} />
                        ) : (
                          <ReplayIcon fontSize="small" />
                        )}
                      </IconButton>
                    </span>
                  </Tooltip>
                </Box>
                <Typography
                  sx={{
                    fontFamily: fonts.mono,
                    fontSize: '0.85rem',
                    whiteSpace: 'pre-wrap',
                    bgcolor: 'action.hover',
                    borderRadius: 2,
                    p: 2,
                  }}
                >
                  {response.value}
                </Typography>
              </Box>
            ))}
          </Stack>
        )}
      </Box>
    </Box>
  )
}
