import { useEffect, useState } from 'react'
import { Box, Container, Divider, InputAdornment, TextField, Typography } from '@mui/material'
import SearchOutlined from '@mui/icons-material/SearchOutlined'
import { api, ApiError } from '../api/client'
import type { Document } from '../api/types'
import DocumentRow from '../components/DocumentRow'
import Masthead from '../components/Masthead'
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
        setError(e instanceof ApiError ? e.message : String(e))
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
          <Helper color="error.main">{error}</Helper>
        ) : searching ? (
          <Helper>Searching…</Helper>
        ) : query === '' ? (
          <Helper>Press space or enter to search your stories.</Helper>
        ) : results.length > 0 ? (
          results.map((doc) => <DocumentRow key={doc.id} doc={doc} showArchived />)
        ) : (
          <Helper>No stories match “{query}”.</Helper>
        )}
      </Container>
    </Box>
  )
}

// Helper is the small mono status line used for loading / empty / error states.
function Helper({
  children,
  color = 'text.secondary',
}: {
  children: React.ReactNode
  color?: string
}) {
  return (
    <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color, py: 1.5 }}>
      {children}
    </Typography>
  )
}
