import { useState } from 'react'
import { Box, Button, InputAdornment, Stack, TextField, Typography } from '@mui/material'
import UnfoldMore from '@mui/icons-material/UnfoldMore'
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
      <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 2, mb: 3 }}>
        <Typography
          variant="overline"
          sx={{ fontFamily: fonts.mono, color: 'primary.main', whiteSpace: 'nowrap' }}
        >
          Attributes
        </Typography>
        <Box sx={{ flex: 1, borderBottom: '1px dotted', borderColor: 'divider', transform: 'translateY(-3px)' }} />
        <Button variant="outlined" size="small" onClick={handleSave} disabled={!dirty || saving}>
          {saving ? 'Saving…' : 'Save'}
        </Button>
      </Box>

      {keys.length === 0 ? (
        <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.85rem', color: 'text.secondary' }}>
          No attributes yet.
        </Typography>
      ) : (
        <Stack spacing={2}>
          {keys.map((key) => {
            const value = values[key] ?? ''
            const collapsed = !expanded.has(key) && value.length > 30
            if (collapsed) {
              return (
                <TextField
                  key={key}
                  label={key}
                  value={`${value.slice(0, 28)}…`}
                  onClick={() => reveal(key)}
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
            }
            return (
              <TextField
                key={key}
                label={key}
                value={value}
                onChange={(e) => setValues((v) => ({ ...v, [key]: e.target.value }))}
                multiline
                minRows={1}
              />
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
  )
}
