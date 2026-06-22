import { useMemo, useState, type ReactNode } from 'react'
import {
  Box,
  Chip,
  Container,
  Divider,
  Link as MuiLink,
  MenuItem,
  Select,
  Typography,
  type SelectChangeEvent,
} from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { timeAgo } from '../lib/relativeTime'
import { fonts } from '../theme'
import Masthead from './Masthead'
import type { Document } from '../api/types'

export type SortOrder = 'alpha' | 'oldest' | 'newest'

const sortOptions: { value: SortOrder; label: string }[] = [
  { value: 'newest', label: 'Newest first' },
  { value: 'oldest', label: 'Oldest first' },
  { value: 'alpha', label: 'Alphabetical' },
]

interface DocumentListProps {
  heading: string
  documents: Document[] | null
  loading: boolean
  error: string | null
  // Which page we're on, so the masthead links to the other view.
  active: 'documents' | 'archive'
  // Initial sort order; differs per page (Documents: newest, Archive: alpha).
  defaultSort: SortOrder
  // Optional extra content (e.g. a floating action button).
  children?: ReactNode
}

// DocumentList renders the masthead and a ledger-style list of documents shared
// by the Documents and Archive pages.
export default function DocumentList({
  heading,
  documents,
  loading,
  error,
  active,
  defaultSort,
  children,
}: DocumentListProps) {
  const [sort, setSort] = useState<SortOrder>(defaultSort)

  const sorted = useMemo(() => {
    if (!documents) return documents
    const copy = [...documents]
    copy.sort((a, b) => {
      switch (sort) {
        case 'alpha':
          return (a.title || 'Untitled').localeCompare(b.title || 'Untitled', undefined, {
            sensitivity: 'base',
          })
        case 'oldest':
          return new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime()
        case 'newest':
        default:
          return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
      }
    })
    return copy
  }, [documents, sort])

  const hasDocuments = sorted !== null && sorted.length > 0

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead active={active} />

      <Container maxWidth="md" sx={{ pt: { xs: 5, md: 8 }, pb: 12 }}>
        <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)', mb: 4 }}>
          {heading}
        </Typography>

        {hasDocuments && (
          <Box
            sx={{
              display: 'flex',
              alignItems: 'baseline',
              justifyContent: 'flex-end',
              gap: 1.25,
              mb: 0.5,
            }}
          >
            <Typography variant="overline" sx={{ color: 'text.secondary' }}>
              Sort
            </Typography>
            <Select
              value={sort}
              onChange={(e: SelectChangeEvent) => setSort(e.target.value as SortOrder)}
              size="small"
              variant="standard"
              disableUnderline
              sx={{ fontFamily: fonts.mono, fontSize: '0.8rem' }}
            >
              {sortOptions.map((o) => (
                <MenuItem key={o.value} value={o.value} sx={{ fontFamily: fonts.mono, fontSize: '0.8rem' }}>
                  {o.label}
                </MenuItem>
              ))}
            </Select>
          </Box>
        )}

        <Divider sx={{ mb: 1 }} />

        {error ? (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'error.main', py: 1.5 }}>
            {error}
          </Typography>
        ) : loading ? (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'text.secondary', py: 1.5 }}>
            Loading…
          </Typography>
        ) : sorted && sorted.length > 0 ? (
          sorted.map((doc) => (
            <MuiLink
              key={doc.id}
              component={RouterLink}
              to={`/documents/${doc.id}`}
              underline="none"
              color="inherit"
              sx={{ display: 'block', '&:hover .doc-title': { color: 'primary.main' } }}
            >
              <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 2, py: 1.75 }}>
                <Typography
                  className="doc-title"
                  sx={{ fontFamily: fonts.display, fontSize: '1.15rem', transition: 'color 120ms' }}
                >
                  {doc.title || 'Untitled'}
                </Typography>
                {(doc.labels ?? []).map((label) => (
                  <Chip
                    key={label}
                    label={label}
                    size="small"
                    variant="outlined"
                    sx={{
                      fontFamily: fonts.mono,
                      fontSize: '0.7rem',
                      height: 20,
                      color: 'text.secondary',
                      borderColor: 'divider',
                    }}
                  />
                ))}
                <Box
                  sx={{ flex: 1, borderBottom: '1px dotted', borderColor: 'divider', transform: 'translateY(-3px)' }}
                />
                <Typography
                  sx={{ fontFamily: fonts.mono, fontSize: '0.8rem', color: 'text.secondary', whiteSpace: 'nowrap' }}
                >
                  {timeAgo(doc.updatedAt)}
                </Typography>
              </Box>
              <Divider sx={{ borderStyle: 'dotted' }} />
            </MuiLink>
          ))
        ) : (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'text.secondary', py: 1.5 }}>
            Nothing here yet.
          </Typography>
        )}
      </Container>

      {children}
    </Box>
  )
}
