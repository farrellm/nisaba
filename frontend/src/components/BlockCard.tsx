import { memo, useEffect, useRef, useState } from 'react'
import {
  Box,
  CircularProgress,
  IconButton,
  InputAdornment,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
import CloseIcon from '@mui/icons-material/Close'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import DataObjectIcon from '@mui/icons-material/DataObject'
import EditOutlinedIcon from '@mui/icons-material/EditOutlined'
import DeleteIcon from '@mui/icons-material/Delete'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import Difference from '@mui/icons-material/Difference'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import ReplayIcon from '@mui/icons-material/Replay'
import SaveOutlinedIcon from '@mui/icons-material/SaveOutlined'
import UnfoldMore from '@mui/icons-material/UnfoldMore'
import { api, ApiError } from '../api/client'
import { useAuth } from '../auth/AuthContext'
import AuthorField from './AuthorField'
import Markdown from './Markdown'
import type { Block, Mode } from '../api/types'
import { parseResponseSegments } from '../lib/responseSegments'
import { fonts } from '../theme'

interface BlockCardProps {
  block: Block
  mode: Mode | undefined
  documentAttributes: Record<string, string>
  onBlockUpdated: (block: Block) => void
  onBlockDeleted: (id: number) => void
  onAfterRun: () => void
  defaultOpen?: boolean
}

// BlockCard renders one block: its mode, editable key/values, a run action, and
// the responses produced so far. The body is a collapsible <details>; the mode
// header is the always-visible <summary>.
const BlockCard = memo(function BlockCard({ block, mode, documentAttributes, onBlockUpdated, onBlockDeleted, onAfterRun, defaultOpen }: BlockCardProps) {
  const keys = mode?.keys ?? Object.keys(block.attributes)
  const [values, setValues] = useState<Record<string, string>>(() => {
    const seed: Record<string, string> = {}
    for (const key of keys) seed[key] = block.attributes[key] ?? ''
    return seed
  })
  const { user } = useAuth()
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [copying, setCopying] = useState(false)
  const [running, setRunning] = useState(false)
  // Transient text accumulated while a streamed run is in flight; null when not
  // streaming. Rendered as a live preview until the final block replaces it.
  const [streamingText, setStreamingText] = useState<string | null>(null)
  const [armed, setArmed] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [reparsingId, setReparsingId] = useState<number | null>(null)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editValue, setEditValue] = useState('')
  const [savingEditId, setSavingEditId] = useState<number | null>(null)
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [structured, setStructured] = useState<Set<number>>(() => {
    // The last block's newest response opens in the structured view by default.
    const responses = block.responses ?? []
    return defaultOpen && responses.length > 0
      ? new Set([responses[responses.length - 1].id])
      : new Set()
  })
  const armedTimer = useRef<ReturnType<typeof setTimeout>>()

  function reveal(key: string) {
    setExpanded((prev) => new Set(prev).add(key))
  }

  function toggleStructured(id: number) {
    setStructured((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  const dirty = keys.some((key) => (values[key] ?? '') !== (block.attributes[key] ?? ''))
  const busy = saving || copying || running || deleting || reparsingId !== null || savingEditId !== null

  // Save/Copy share one treatment: a quiet muted icon, ringed in accent when the
  // block has uncommitted edits. The clean-state border is transparent (not
  // absent) so the button's box never changes size and the row doesn't shift.
  const editActionSx = (active: boolean) => ({
    border: '1px solid',
    borderColor: active ? 'primary.main' : 'transparent',
    color: active ? 'primary.main' : 'text.disabled',
    '&:hover': { color: 'primary.main', bgcolor: active ? 'action.hover' : 'transparent' },
  })

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
      const updated = user?.streamingEnabled
        ? await api.postStream<Block>(
            `/api/documents/${block.documentId}/blocks/${block.id}/run/stream`,
            { attributes: values },
            (text) => setStreamingText((prev) => (prev ?? '') + text),
            'block',
          )
        : await api.post<Block>(
            `/api/documents/${block.documentId}/blocks/${block.id}/run`,
            { attributes: values },
          )
      // A freshly run response opens in the structured view by default — except
      // for streamed runs, which stay in the raw view the user just watched.
      const fresh = (updated.responses ?? [])[(updated.responses ?? []).length - 1]
      if (fresh && !user?.streamingEnabled) setStructured((prev) => new Set(prev).add(fresh.id))
      onBlockUpdated(updated)
      onAfterRun()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not run. Try again.')
    } finally {
      setStreamingText(null)
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

  function startEdit(responseId: number, value: string) {
    setError(null)
    setEditingId(responseId)
    setEditValue(value)
  }

  function cancelEdit() {
    setEditingId(null)
  }

  async function handleSaveEdit(responseId: number) {
    setError(null)
    setSavingEditId(responseId)
    try {
      const updated = await api.put<Block>(
        `/api/documents/${block.documentId}/blocks/${block.id}/responses/${responseId}`,
        { value: editValue },
      )
      onBlockUpdated(updated)
      onAfterRun() // editing auto-reparses into the document's shared attributes
      setEditingId(null)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save. Try again.')
    } finally {
      setSavingEditId(null)
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
          {keys.map((key) => {
            const value = values[key] ?? ''
            const collapsed = key !== 'author' && !expanded.has(key) && value.length > 80
            let field
            if (key === 'author') {
              field = (
                <AuthorField
                  value={value}
                  onChange={(v) => setValues((prev) => ({ ...prev, [key]: v }))}
                />
              )
            } else if (collapsed) {
              field = (
                <TextField
                  label={key}
                  value={`${value.slice(0, 40)}…`}
                  onClick={() => reveal(key)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      reveal(key)
                    }
                  }}
                  inputProps={{ tabIndex: 0 }}
                  InputProps={{
                    readOnly: true,
                    endAdornment: (
                      <InputAdornment position="end" sx={{ color: 'text.secondary' }}>
                        <UnfoldMore fontSize="small" />
                      </InputAdornment>
                    ),
                    sx: {
                      cursor: 'pointer',
                      '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: 'primary.main' },
                    },
                  }}
                />
              )
            } else {
              field = (
                <TextField
                  label={key}
                  value={value}
                  onChange={(e) => setValues((v) => ({ ...v, [key]: e.target.value }))}
                  multiline
                  minRows={1}
                />
              )
            }

            // Diff link mirrors the Save/Copy buttons (editActionSx): ringed in
            // accent when the saved block value diverges from the document
            // value, disabled when they match. Compares saved (not live-edited)
            // values to match what the diff page renders from the server.
            const differs = (block.attributes[key] ?? '') !== (documentAttributes[key] ?? '')
            return (
              <Stack key={key} direction="row" spacing={0.5} sx={{ alignItems: 'flex-start' }}>
                <Box sx={{ flex: 1 }}>{field}</Box>
                <Tooltip title={differs ? 'Compare with document' : 'Matches document'}>
                  <span>
                    <IconButton
                      component="a"
                      href={`/documents/${block.documentId}/blocks/${block.id}/attributes/${encodeURIComponent(key)}/diff`}
                      target="_blank"
                      rel="noopener"
                      disabled={!differs}
                      aria-label={`Compare ${key} with document`}
                      size="small"
                      sx={{ mt: 1, ...editActionSx(differs) }}
                    >
                      <Difference fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
              </Stack>
            )
          })}
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
                sx={editActionSx(dirty)}
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
                sx={editActionSx(dirty)}
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

        {streamingText !== null && (
          <Box sx={{ mt: 3 }}>
            <Typography
              variant="overline"
              sx={{ fontFamily: fonts.mono, color: 'text.secondary', fontSize: '0.7rem', display: 'block', mb: 1 }}
            >
              streaming…
            </Typography>
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
              {streamingText}
              <Box component="span" sx={{ opacity: 0.5 }}>▌</Box>
            </Typography>
          </Box>
        )}

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
                  {editingId === response.id ? (
                    <>
                      <Tooltip title="Save edit">
                        <span>
                          <IconButton
                            size="small"
                            disabled={busy}
                            aria-label="Save edited response"
                            onClick={(e) => {
                              e.preventDefault()
                              handleSaveEdit(response.id)
                            }}
                            sx={editActionSx(editValue !== response.value)}
                          >
                            {savingEditId === response.id ? (
                              <CircularProgress size={18} />
                            ) : (
                              <SaveOutlinedIcon fontSize="small" />
                            )}
                          </IconButton>
                        </span>
                      </Tooltip>
                      <Tooltip title="Cancel edit">
                        <span>
                          <IconButton
                            size="small"
                            disabled={busy}
                            aria-label="Cancel editing response"
                            onClick={(e) => {
                              e.preventDefault()
                              cancelEdit()
                            }}
                            sx={{ color: 'text.disabled', '&:hover': { color: 'primary.main' } }}
                          >
                            <CloseIcon fontSize="small" />
                          </IconButton>
                        </span>
                      </Tooltip>
                    </>
                  ) : (
                    <>
                      <Tooltip title={structured.has(response.id) ? 'Raw view' : 'Structured view'}>
                        <span>
                          <IconButton
                            size="small"
                            aria-label={structured.has(response.id) ? 'Show raw response' : 'Show structured response'}
                            onClick={(e) => {
                              e.preventDefault()
                              toggleStructured(response.id)
                            }}
                            sx={{
                              color: structured.has(response.id) ? 'primary.main' : 'text.disabled',
                              '&:hover': { color: 'primary.main' },
                            }}
                          >
                            <DataObjectIcon fontSize="small" />
                          </IconButton>
                        </span>
                      </Tooltip>
                      <Tooltip title="Edit response">
                        <span>
                          <IconButton
                            size="small"
                            disabled={busy}
                            aria-label="Edit response"
                            onClick={(e) => {
                              e.preventDefault()
                              startEdit(response.id, response.value)
                            }}
                            sx={{ color: 'text.disabled', '&:hover': { color: 'primary.main' } }}
                          >
                            <EditOutlinedIcon fontSize="small" />
                          </IconButton>
                        </span>
                      </Tooltip>
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
                    </>
                  )}
                </Box>
                {editingId === response.id ? (
                  <TextField
                    fullWidth
                    multiline
                    minRows={6}
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    InputProps={{
                      sx: { fontFamily: fonts.mono, fontSize: '0.85rem' },
                    }}
                  />
                ) : structured.has(response.id) ? (
                  <Box sx={{ bgcolor: 'action.hover', borderRadius: 2, p: 2 }}>
                    {parseResponseSegments(response.value).map((seg, segIdx) =>
                      seg.kind === 'text' ? (
                        <Markdown key={segIdx}>{seg.text}</Markdown>
                      ) : (
                        <Box
                          key={segIdx}
                          component="details"
                          open
                          sx={{ my: 1, '&:first-of-type': { mt: 0 }, '&:last-child': { mb: 0 } }}
                        >
                          <Box
                            component="summary"
                            sx={{
                              cursor: 'pointer',
                              fontFamily: fonts.mono,
                              fontSize: '0.8rem',
                              color: 'text.secondary',
                            }}
                          >
                            {seg.name}
                          </Box>
                          <Box
                            component="blockquote"
                            sx={{
                              my: 1,
                              ml: 0,
                              pl: 2.5,
                              borderLeft: '3px solid',
                              borderColor: 'divider',
                              color: 'text.secondary',
                            }}
                          >
                            {/* Escape '<' so nested tags render as literal text:
                                react-markdown drops raw HTML. */}
                            <Markdown>{seg.inner.split('<').join('\\<')}</Markdown>
                          </Box>
                        </Box>
                      ),
                    )}
                  </Box>
                ) : (
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
                )}
              </Box>
            ))}
          </Stack>
        )}
      </Box>
    </Box>
  )
})

export default BlockCard
