import { useEffect, useState } from 'react'
import { Box, Typography } from '@mui/material'
import { useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { DocumentDetail } from '../api/types'
import Markdown from '../components/Markdown'
import { usePageTitle } from '../lib/usePageTitle'
import { fonts } from '../theme'

// AttributePage is a standalone, chrome-free view of a single document attribute
// value rendered as markdown. No masthead, no key label — just the content. Meant
// to be opened in its own tab from DocumentAttributes.
export default function AttributePage() {
  const { id, key = '' } = useParams()
  usePageTitle(key || null)

  const [doc, setDoc] = useState<DocumentDetail | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .get<DocumentDetail>(`/api/documents/${id}`)
      .then(setDoc)
      .catch(() => setError('Could not load this attribute.'))
  }, [id])

  const value = doc?.attributes?.[key]

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Box
        component="main"
        sx={{ maxWidth: 720, mx: 'auto', px: { xs: 3, md: 4 }, py: { xs: 6, md: 10 } }}
      >
        {error ? (
          <Notice>{error}</Notice>
        ) : !doc ? null : value ? (
          <Markdown>{value}</Markdown>
        ) : (
          <Notice>No value for this attribute.</Notice>
        )}
      </Box>
    </Box>
  )
}

function Notice({ children }: { children: string }) {
  return (
    <Typography
      sx={{ fontFamily: fonts.mono, fontSize: '0.85rem', color: 'text.secondary' }}
    >
      {children}
    </Typography>
  )
}
