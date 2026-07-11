import { useEffect, useState } from 'react'
import { Box, Typography } from '@mui/material'
import { useParams } from 'react-router-dom'
import { api } from '../api/client'
import Markdown from '../components/Markdown'
import { fonts } from '../theme'

// AttributePage is a standalone, chrome-free view of a single document attribute
// value rendered as markdown. No masthead, no key label — just the content. Meant
// to be opened in its own tab from DocumentAttributes.
export default function AttributePage() {
  const { id, key = '' } = useParams()

  const [value, setValue] = useState<string | null>(null)
  const [title, setTitle] = useState<string | null>(null)
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // The tab title is just the document title (no " · Nisaba" suffix), so set
  // document.title directly rather than going through usePageTitle.
  useEffect(() => {
    if (title) document.title = title
  }, [title])

  useEffect(() => {
    api
      .get<{ value: string; title: string }>(
        `/api/public/documents/${id}/attributes/${encodeURIComponent(key)}`,
      )
      .then((r) => {
        setValue(r.value)
        setTitle(r.title)
      })
      .catch(() => setError('Could not load this attribute.'))
      .finally(() => setLoaded(true))
  }, [id, key])

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Box
        component="main"
        sx={{ maxWidth: 720, mx: 'auto', px: { xs: 3, md: 4 }, py: { xs: 6, md: 10 } }}
      >
        {error ? (
          <Notice>{error}</Notice>
        ) : !loaded ? null : value ? (
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
    <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.85rem', color: 'text.secondary' }}>
      {children}
    </Typography>
  )
}
