import { memo, useEffect, useState } from 'react'
import { Box, CircularProgress, IconButton, Stack, Tooltip, Typography } from '@mui/material'
import CloseIcon from '@mui/icons-material/Close'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import EditOutlinedIcon from '@mui/icons-material/EditOutlined'
import DeleteIcon from '@mui/icons-material/Delete'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import Difference from '@mui/icons-material/Difference'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import ReplayIcon from '@mui/icons-material/Replay'
import SaveOutlinedIcon from '@mui/icons-material/SaveOutlined'
import { api, ApiError } from '../api/client'
import type { DeltaKind } from '../api/client'
import { errorMessage } from '../lib/errors'
import { useAuth } from '../auth/AuthContext'
import AuthorField from './AuthorField'
import CollapsibleValueField from './CollapsibleValueField'
import ResponseView from './ResponseView'
import StatusLine from './StatusLine'
import StreamingPreview from './StreamingPreview'
import type { Block, DocumentDetail, Mode } from '../api/types'
import { StreamBuffer } from '../lib/streamBuffer'
import {
  RunRecorder,
  clearRunEntry,
  isRunStale,
  loadRunEntry,
  type RunEntry,
} from '../lib/runState'
import { fonts } from '../theme'
import { leaderSx, summarySx } from '../lib/styles'
import { addToSet, setInSet, toggleSet } from '../lib/sets'
import { useArmedAction } from '../lib/useArmedAction'

interface BlockCardProps {
  block: Block
  mode: Mode | undefined
  documentAttributes: Record<string, string>
  onBlockUpdated: (block: Block) => void
  onBlockDeleted: (id: number) => void
  onAfterRun: () => void
  defaultOpen?: boolean
}

// How often a restored run polls the document for its saved response.
const RESUME_POLL_MS = 5000

function describeElapsed(startedAt: number): string {
  const mins = Math.floor((Date.now() - startedAt) / 60000)
  return mins < 1 ? 'moments ago' : mins === 1 ? '1 minute ago' : `${mins} minutes ago`
}

// BlockCard renders one block: its mode, editable key/values, a run action, and
// the responses produced so far. The body is a collapsible <details>; the mode
// header is the always-visible <summary>.
const BlockCard = memo(function BlockCard({
  block,
  mode,
  documentAttributes,
  onBlockUpdated,
  onBlockDeleted,
  onAfterRun,
  defaultOpen,
}: BlockCardProps) {
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
  // The in-flight streamed run's text buffer; null when not streaming. The
  // StreamingPreview below subscribes to it, so per-delta re-renders stay out
  // of this (large) component.
  const [stream, setStream] = useState<StreamBuffer | null>(null)
  // A run restored from localStorage after a reload; non-null while polling
  // the document for its saved response.
  const [resumeEntry, setResumeEntry] = useState<RunEntry | null>(null)
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
  // Which responses' <details> are expanded. Controlled so a run can collapse
  // the old ones on click and open just the new one on completion. Seeds with
  // the newest response open, matching how each list started before.
  const [openIds, setOpenIds] = useState<Set<number>>(() => {
    const responses = block.responses ?? []
    return responses.length > 0 ? new Set([responses[responses.length - 1].id]) : new Set()
  })

  // Restore a run that was in flight when the page was last unloaded. The run
  // keeps going server-side (detached), so re-show the running state and the
  // persisted partial text, then poll until its response lands. Mount-only:
  // document refetches update the block prop without remounting the card.
  useEffect(() => {
    const entry = loadRunEntry(block.documentId, block.id)
    if (!entry) return
    if ((block.responses ?? []).length > entry.baseResponseCount) {
      // The run finished while we were away; this page load already has it.
      clearRunEntry(block.documentId, block.id)
      return
    }
    const buffer = new StreamBuffer()
    if (entry.text !== '') buffer.push(entry.text)
    setStream(buffer)
    setRunning(true)
    setResumeEntry(entry)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // While restoring, poll the document until the run's response appears. A run
  // that failed while the tab was gone saved nothing and left no failure
  // signal to poll, so give up at the staleness bound; the dismiss button next
  // to the status line covers the impatient path.
  useEffect(() => {
    if (!resumeEntry) return
    let stopped = false

    const settle = () => {
      stopped = true
      clearRunEntry(block.documentId, block.id)
      setResumeEntry(null)
      setStream(null)
      setRunning(false)
    }

    const tick = async () => {
      if (stopped) return
      if (isRunStale(resumeEntry)) {
        settle()
        setError('The run did not complete within 15 minutes.')
        return
      }
      if (loadRunEntry(block.documentId, block.id) === null) {
        // Another tab finished or dismissed this run; stand down, refreshing
        // in case its response is already saved.
        settle()
        onAfterRun()
        return
      }
      try {
        const doc = await api.get<DocumentDetail>(`/api/documents/${block.documentId}`)
        if (stopped) return
        const fresh = (doc.blocks ?? []).find((b) => b.id === block.id)
        if (!fresh || (fresh.responses ?? []).length <= resumeEntry.baseResponseCount) return
        settle()
        // Same treatment as a live run completing (see handleRun).
        const newest = (fresh.responses ?? [])[(fresh.responses ?? []).length - 1]
        if (newest && !user?.streamingEnabled) setStructured((prev) => addToSet(prev, newest.id))
        if (newest) setOpenIds(new Set([newest.id]))
        onBlockUpdated(fresh)
        onAfterRun()
      } catch {
        // transient fetch error — keep polling; staleness bounds the wait
      }
    }

    void tick()
    const interval = setInterval(() => void tick(), RESUME_POLL_MS)
    return () => {
      stopped = true
      clearInterval(interval)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resumeEntry])

  // Stop waiting for a restored run without erroring. If the run does finish
  // later, its response is saved server-side and shows up on the next refresh.
  function dismissResume() {
    clearRunEntry(block.documentId, block.id)
    setResumeEntry(null)
    setStream(null)
    setRunning(false)
  }

  const reveal = (key: string) => setExpanded((prev) => addToSet(prev, key))
  const toggleStructured = (id: number) => setStructured((prev) => toggleSet(prev, id))

  const dirty = keys.some((key) => (values[key] ?? '') !== (block.attributes[key] ?? ''))
  const busy =
    saving || copying || running || deleting || reparsingId !== null || savingEditId !== null

  // Save/Copy share one treatment: a quiet muted icon, ringed in accent when the
  // block has uncommitted edits. The clean-state border is transparent (not
  // absent) so the button's box never changes size and the row doesn't shift.
  const editActionSx = (active: boolean) => ({
    border: '1px solid',
    borderColor: active ? 'primary.main' : 'transparent',
    color: active ? 'primary.main' : 'text.disabled',
    '&:hover': { color: 'primary.main', bgcolor: active ? 'action.hover' : 'transparent' },
  })

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
      setError(errorMessage(err, 'Could not save. Try again.'))
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
      setError(errorMessage(err, 'Could not copy. Try again.'))
    } finally {
      setCopying(false)
    }
  }

  // Deleting is a two-step action (see useArmedAction). Both clicks must not
  // toggle the surrounding <details>, so the caller stops propagation.
  const { armed, fire: fireDelete, disarm: disarmDelete } = useArmedAction(handleDelete)

  async function handleDelete() {
    setError(null)
    setDeleting(true)
    try {
      await api.del(`/api/documents/${block.documentId}/blocks/${block.id}`)
      onBlockDeleted(block.id)
    } catch (err) {
      setError(errorMessage(err, 'Could not delete. Try again.'))
      disarmDelete()
      setDeleting(false)
    }
  }

  async function handleRun() {
    setError(null)
    setRunning(true)
    // Collapse the old responses immediately; the in-flight preview block (and,
    // on completion, the new response) takes over the open slot.
    setOpenIds(new Set())
    const buffer = new StreamBuffer()
    setStream(buffer)
    // Remember the run in localStorage so a reload can restore the running
    // state and the preview text accumulated so far (the server finishes the
    // run either way — see the restore effect above).
    const recorder = new RunRecorder(block.documentId, block.id, (block.responses ?? []).length)
    // With streaming off, deltas accumulate unseen and flush into the preview
    // only when a tool-call block completes, so the output jumps to everything
    // produced so far (thinking, text, and the tool call with its result) at
    // each tool boundary. A run with no tool calls never flushes — no preview,
    // same as before.
    let unflushed = ''
    const onDelta = (text: string, kind: DeltaKind) => {
      unflushed += text
      if (user?.streamingEnabled || kind === 'tool') {
        buffer.push(unflushed)
        recorder.append(unflushed)
        unflushed = ''
      }
    }
    let updated: Block
    try {
      // Always stream so the server's keepalive pings keep the connection warm
      // through a long run (avoids proxy 504s). The streamingEnabled setting only
      // controls whether incoming text is displayed live or at tool boundaries.
      updated = await api.postStream<Block>(
        `/api/documents/${block.documentId}/blocks/${block.id}/run/stream`,
        { attributes: values },
        onDelta,
        'block',
      )
    } catch (err) {
      if (err instanceof ApiError) {
        // The server refused the run or reported a failure — nothing will be
        // saved, so forget the run.
        setError(errorMessage(err, 'Could not run. Try again.'))
        recorder.clear()
        setStream(null)
        setRunning(false)
        return
      }
      // The connection was interrupted (reload or app-switch teardown, network
      // blip) but the detached run continues server-side: keep the entry and
      // hand over to the resume/poll path. Notably, iOS Safari runs this catch
      // during page teardown on reload — clearing here would erase the entry
      // the reloaded page needs to restore.
      setResumeEntry(recorder.stop())
      return
    }
    recorder.clear()
    setStream(null)
    setRunning(false)
    // A freshly run response opens in the structured view by default — except
    // for streamed runs, which stay in the raw view the user just watched.
    const fresh = (updated.responses ?? [])[(updated.responses ?? []).length - 1]
    if (fresh && !user?.streamingEnabled) setStructured((prev) => addToSet(prev, fresh.id))
    if (fresh) setOpenIds(new Set([fresh.id]))
    onBlockUpdated(updated)
    onAfterRun()
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
      setError(errorMessage(err, 'Could not re-parse. Try again.'))
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
      setError(errorMessage(err, 'Could not save. Try again.'))
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
        <Box component="summary" sx={summarySx}>
          <Typography
            variant="overline"
            sx={{ fontFamily: fonts.mono, color: 'primary.main', whiteSpace: 'nowrap' }}
          >
            {mode?.label ?? block.mode}
          </Typography>
          <Box sx={leaderSx} />
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
                  fireDelete()
                }}
                sx={{
                  transform: 'translateY(-2px)',
                  ...(armed ? {} : { color: 'text.disabled', '&:hover': { color: 'error.main' } }),
                }}
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
            const field =
              key === 'author' ? (
                <AuthorField
                  value={value}
                  onChange={(v) => setValues((prev) => ({ ...prev, [key]: v }))}
                />
              ) : (
                <CollapsibleValueField
                  label={key}
                  value={value}
                  expanded={expanded.has(key)}
                  onExpand={() => reveal(key)}
                  onChange={(v) => setValues((prev) => ({ ...prev, [key]: v }))}
                />
              )

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
          <StatusLine tone="error" sx={{ fontSize: '0.8rem', mt: 1.5 }}>
            {error}
          </StatusLine>
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
                  '&.Mui-disabled': {
                    bgcolor: 'action.disabledBackground',
                    color: 'action.disabled',
                  },
                }}
              >
                {running ? (
                  <CircularProgress size={18} color="inherit" />
                ) : (
                  <PlayArrowIcon fontSize="small" />
                )}
              </IconButton>
            </span>
          </Tooltip>
        </Stack>

        {stream && <StreamingPreview stream={stream} />}

        {resumeEntry && (
          <StatusLine
            sx={{
              fontSize: '0.8rem',
              mt: 1.5,
              display: 'flex',
              alignItems: 'center',
              gap: 0.5,
            }}
          >
            Waiting for a run started {describeElapsed(resumeEntry.startedAt)} to finish…
            <Tooltip title="Stop waiting">
              <IconButton
                size="small"
                aria-label="Stop waiting for run"
                onClick={dismissResume}
                sx={{ color: 'text.disabled', '&:hover': { color: 'primary.main' } }}
              >
                <CloseIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </StatusLine>
        )}

        {(block.responses ?? []).length > 0 && (
          <Stack spacing={1.5} sx={{ mt: 3 }}>
            {(block.responses ?? [])
              .slice()
              .reverse()
              .map((response) => (
                <ResponseView
                  key={response.id}
                  response={response}
                  open={openIds.has(response.id)}
                  onToggle={(open) => setOpenIds((prev) => setInSet(prev, response.id, open))}
                  structured={structured.has(response.id)}
                  onToggleStructured={() => toggleStructured(response.id)}
                  actions={
                    <>
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
                  }
                  editing={
                    editingId === response.id
                      ? {
                          value: editValue,
                          onChange: setEditValue,
                          actions: (
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
                                    sx={{
                                      color: 'text.disabled',
                                      '&:hover': { color: 'primary.main' },
                                    }}
                                  >
                                    <CloseIcon fontSize="small" />
                                  </IconButton>
                                </span>
                              </Tooltip>
                            </>
                          ),
                        }
                      : undefined
                  }
                />
              ))}
          </Stack>
        )}
      </Box>
    </Box>
  )
})

export default BlockCard
