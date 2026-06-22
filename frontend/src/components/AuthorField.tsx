import { useState } from 'react'
import { Autocomplete, TextField } from '@mui/material'
import { api } from '../api/client'

interface AuthorFieldProps {
  value: string
  onChange: (value: string) => void
}

// AuthorField is a free-solo autocomplete for the block-level `author` attribute.
// Its suggestions are the current user's past author values (fetched lazily on
// first open). Typing a brand new name is allowed and flows through onChange just
// like the plain text field it replaces.
export default function AuthorField({ value, onChange }: AuthorFieldProps) {
  const [options, setOptions] = useState<string[]>([])
  const [loaded, setLoaded] = useState(false)

  async function loadOptions() {
    if (loaded) return
    setLoaded(true)
    try {
      const past = await api.get<string[]>('/api/attribute-values?key=author')
      setOptions(past ?? [])
    } catch {
      // Suggestions are a convenience; on failure the field still accepts typing.
    }
  }

  return (
    <Autocomplete
      freeSolo
      options={options}
      value={value}
      inputValue={value}
      onOpen={loadOptions}
      // Free-solo: drive state from the input string so typing a new name and
      // picking a suggestion both flow through the same path.
      onInputChange={(_, next) => onChange(next)}
      renderInput={(params) => <TextField {...params} label="author" />}
    />
  )
}
