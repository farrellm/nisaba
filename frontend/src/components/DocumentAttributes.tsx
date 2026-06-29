import { useEffect, useRef, useState } from 'react'
import { Box, Button, IconButton, InputAdornment, Stack, TextField, Tooltip, Typography } from '@mui/material'
import UnfoldMore from '@mui/icons-material/UnfoldMore'
import OpenInNew from '@mui/icons-material/OpenInNew'
import { api, ApiError } from '../api/client'
import type { DocumentDetail } from '../api/types'
import { fonts } from '../theme'

interface DocumentAttributesProps {
  doc: DocumentDetail
  onChange: (doc: DocumentDetail) => void
}

// DocumentAttributes renders the document's shared attribute namespace: one
// editable value field per key (keys are created by running blocks, so they are
// fixed here), with a Save button at the top of the section.
export default function DocumentAttributes({ doc, onChange }: DocumentAttributesProps) {
  const attributes = doc.attributes ?? {}
  const keys = Object.keys(attributes).sort()
  const [values, setValues] = useState<Record<string, string>>(() => {
    const seed: Record<string, string> = {}
    for (const key of keys) seed[key] = attributes[key] ?? ''
    return seed
  })
  // Re-sync local values when the document's attributes change (e.g. after a
  // run promotes new values into the shared namespace). useState seeds only
  // once, so without this the fields keep showing stale text. Preserve any
  // unsaved edits by adopting the new server value only for keys the user
  // hasn't locally diverged on, tracked against the last server snapshot.
  const serverRef = useRef<Record<string, string>>(attributes)
  useEffect(() => {
    const server = doc.attributes ?? {}
    setValues((prev) => {
      const next = { ...prev }
      for (const key of Object.keys(server)) {
        const userEdited = (prev[key] ?? '') !== (serverRef.current[key] ?? '')
        if (!userEdited) next[key] = server[key]
      }
      return next
    })
    serverRef.current = server
  }, [doc.attributes])
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  function reveal(key: string) {
    setExpanded((prev) => new Set(prev).add(key))
  }

  const dirty = keys.some((key) => (values[key] ?? '') !== (attributes[key] ?? ''))

  async function handleSave() {
    setError(null)
    setSaving(true)
    try {
      const updated = await api.put<DocumentDetail>(`/api/documents/${doc.id}`, {
        attributes: values,
      })
      onChange(updated)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save. Try again.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Box component="section" sx={{ py: 4, borderTop: '1px dotted', borderColor: 'divider', mt: 2 }}>
      <Box
        component="details"
        sx={{ '&[open]': { borderBottom: '1px dotted', borderColor: 'divider', pb: 4 } }}
      >
        <Box
          component="summary"
          sx={{
            display: 'flex',
            alignItems: 'baseline',
            gap: 2,
            mb: 3,
            cursor: 'pointer',
            listStyle: 'none',
            '&::-webkit-details-marker': { display: 'none' },
          }}
        >
          <Typography
            variant="overline"
            sx={{ fontFamily: fonts.mono, color: 'primary.main', whiteSpace: 'nowrap' }}
          >
            Attributes
          </Typography>
          <Box sx={{ flex: 1, borderBottom: '1px dotted', borderColor: 'divider', transform: 'translateY(-3px)' }} />
        </Box>

        <Stack direction="row" spacing={1.5} sx={{ mb: 3 }}>
          <Button variant="outlined" size="small" onClick={handleSave} disabled={!dirty || saving}>
            {saving ? 'Saving…' : 'Save'}
          </Button>
        </Stack>

        {keys.length === 0 ? (
        <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.85rem', color: 'text.secondary' }}>
          No attributes yet.
        </Typography>
      ) : (
        <Stack spacing={2}>
          {keys.map((key) => {
            const value = values[key] ?? ''
            const collapsed = !expanded.has(key) && value.length > 80
            const field = collapsed ? (
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
            ) : (
              <TextField
                label={key}
                value={value}
                onChange={(e) => setValues((v) => ({ ...v, [key]: e.target.value }))}
                multiline
                minRows={1}
              />
            )
            return (
              <Stack key={key} direction="row" spacing={0.5} sx={{ alignItems: 'flex-start' }}>
                <Box sx={{ flex: 1 }}>{field}</Box>
                <Tooltip title="View as markdown">
                  <IconButton
                    component="a"
                    href={`/documents/${doc.id}/attributes/${encodeURIComponent(key)}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    aria-label={`View ${key} as markdown`}
                    size="small"
                    sx={{ mt: 1, color: 'text.secondary', '&:hover': { color: 'primary.main' } }}
                  >
                    <OpenInNew fontSize="small" />
                  </IconButton>
                </Tooltip>
              </Stack>
            )
          })}
        </Stack>
      )}

        {error && (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.8rem', color: 'error.main', mt: 1.5 }}>
            {error}
          </Typography>
        )}
      </Box>
    </Box>
  )
}
