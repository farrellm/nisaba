import { useState } from 'react'
import { Box, Button, Stack, TextField, Typography } from '@mui/material'
import { api, ApiError } from '../api/client'
import type { Block, Mode } from '../api/types'
import { fonts } from '../theme'

interface BlockCardProps {
  block: Block
  mode: Mode | undefined
  onBlockUpdated: (block: Block) => void
  onAfterRun: () => void
  defaultOpen?: boolean
}

// BlockCard renders one block: its mode, editable key/values, a run action, and
// the responses produced so far. The body is a collapsible <details>; the mode
// header is the always-visible <summary>.
export default function BlockCard({ block, mode, onBlockUpdated, onAfterRun, defaultOpen }: BlockCardProps) {
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

  const dirty = keys.some((key) => (values[key] ?? '') !== (block.attributes[key] ?? ''))
  const busy = saving || copying || running

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

        <Stack direction="row" spacing={1.5} sx={{ mt: 2 }}>
          <Button variant="outlined" size="small" onClick={handleSave} disabled={!dirty || busy}>
            {saving ? 'Saving…' : 'Save'}
          </Button>
          <Button variant="outlined" size="small" onClick={handleCopy} disabled={busy}>
            {copying ? 'Copying…' : 'Copy to document'}
          </Button>
          <Button variant="contained" size="small" onClick={handleRun} disabled={busy}>
            {running ? 'Running…' : 'Run'}
          </Button>
        </Stack>

        {(block.responses ?? []).length > 0 && (
          <Stack spacing={1.5} sx={{ mt: 3 }}>
            {(block.responses ?? []).slice().reverse().map((response, idx) => (
              <Box key={response.id} component="details" {...(idx === 0 ? { open: true } : {})}>
                <Box
                  component="summary"
                  sx={{
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
