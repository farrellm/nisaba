import { useEffect, useState } from 'react'
import { Box, Container, Divider, InputAdornment, TextField, Typography } from '@mui/material'
import SearchOutlined from '@mui/icons-material/SearchOutlined'
import { api } from '../api/client'
import { errorMessage } from '../lib/errors'
import { listStatusSx } from '../lib/styles'
import type { Document } from '../api/types'
import DocumentRow from '../components/DocumentRow'
import Masthead from '../components/Masthead'
import StatusLine from '../components/StatusLine'
import { usePageTitle } from '../lib/usePageTitle'
import { fonts } from '../theme'

// SearchPage is a full-text search over every document's `story` attribute. The
// query commits on word boundaries — Space (a word just completed) or Enter —
// rather than on every keystroke, matching the word/lexeme nature of Postgres FTS.
export default function SearchPage() {
  usePageTitle('Search')
  const [input, setInput] = useState('')
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Document[]>([])
  const [searching, setSearching] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (query === '') {
      setResults([])
      setSearching(false)
      setError(null)
      return
    }
    let stale = false
    setSearching(true)
    setError(null)
    api
      .get<Document[]>(`/api/documents/search?q=${encodeURIComponent(query)}`)
      .then((docs) => {
        if (stale) return
        setResults(docs ?? [])
        setSearching(false)
      })
      .catch((e: unknown) => {
        if (stale) return
        setError(errorMessage(e))
        setSearching(false)
      })
    return () => {
      stale = true
    }
  }, [query])

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead />

      <Container maxWidth="md" sx={{ pt: { xs: 5, md: 8 }, pb: 12 }}>
        <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)', mb: 4 }}>
          Search
        </Typography>

        <TextField
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              if (e.key === 'Enter') e.preventDefault()
              setQuery(input.trim())
            }
          }}
          fullWidth
          autoFocus
          placeholder="Search stories — press space or enter…"
          inputProps={{ 'aria-label': 'Search stories' }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchOutlined sx={{ color: 'text.secondary' }} />
              </InputAdornment>
            ),
          }}
          sx={{ mb: 3, '& input': { fontFamily: fonts.mono } }}
        />
        <Divider sx={{ mb: 1 }} />

        {error ? (
          <StatusLine tone="error" sx={listStatusSx}>
            {error}
          </StatusLine>
        ) : searching ? (
          <StatusLine sx={listStatusSx}>Searching…</StatusLine>
        ) : query === '' ? (
          <StatusLine sx={listStatusSx}>Press space or enter to search your stories.</StatusLine>
        ) : results.length > 0 ? (
          results.map((doc) => <DocumentRow key={doc.id} doc={doc} showArchived />)
        ) : (
          <StatusLine sx={listStatusSx}>No stories match “{query}”.</StatusLine>
        )}
      </Container>
    </Box>
  )
}
