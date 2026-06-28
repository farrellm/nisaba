import { useEffect, useState } from 'react'
import { MenuItem, Paper, Select, Typography, type SelectChangeEvent } from '@mui/material'
import { api } from '../api/client'
import type { DocumentDetail, LLMModel } from '../api/types'
import { fonts } from '../theme'

interface ModelSelectorProps {
  doc: DocumentDetail
  onChange: (doc: DocumentDetail) => void
}

// ModelSelector is a fixed lower-left widget that lists the available models and
// auto-saves the document's choice on change.
export default function ModelSelector({ doc, onChange }: ModelSelectorProps) {
  const [models, setModels] = useState<LLMModel[]>([])
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(false)

  useEffect(() => {
    api.get<LLMModel[]>('/api/models').then(setModels).catch(() => setModels([]))
  }, [])

  async function handleChange(e: SelectChangeEvent) {
    const selectedModel = e.target.value
    setSaving(true)
    setError(false)
    try {
      const updated = await api.put<DocumentDetail>(`/api/documents/${doc.id}`, { selectedModel })
      onChange(updated)
    } catch {
      setError(true)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Paper
      elevation={3}
      sx={{
        position: 'fixed',
        bottom: 32,
        left: 32,
        zIndex: (theme) => theme.zIndex.fab,
        px: 2,
        py: 1.25,
        display: 'flex',
        alignItems: 'center',
        gap: 1.5,
        borderRadius: 2,
      }}
    >
      <Typography
        sx={{
          fontFamily: fonts.mono,
          fontSize: '0.65rem',
          textTransform: 'uppercase',
          letterSpacing: '0.08em',
          color: error ? 'error.main' : 'text.secondary',
        }}
      >
        {error ? 'Save failed' : saving ? 'Saving…' : 'Model'}
      </Typography>
      <Select
        value={doc.selectedModel || ''}
        onChange={handleChange}
        size="small"
        displayEmpty
        disabled={saving}
        variant="standard"
        inputProps={{ 'aria-label': 'Select model' }}
        sx={{ fontFamily: fonts.body, fontSize: '0.85rem', minWidth: 160 }}
      >
        <MenuItem value="" disabled>
          Select a model…
        </MenuItem>
        {models.map((m) => (
          <MenuItem key={m.id} value={m.id}>
            {m.label}
          </MenuItem>
        ))}
      </Select>
    </Paper>
  )
}
