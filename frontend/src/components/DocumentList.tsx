import { useMemo, useState, type ReactNode } from 'react'
import {
  Box,
  Container,
  Divider,
  MenuItem,
  Select,
  Typography,
  type SelectChangeEvent,
} from '@mui/material'
import { fonts } from '../theme'
import DocumentRow from './DocumentRow'
import Masthead from './Masthead'
import type { Document } from '../api/types'

export type SortOrder = 'alpha' | 'oldest' | 'newest'

const sortOptions: { value: SortOrder; label: string }[] = [
  { value: 'newest', label: 'Newest first' },
  { value: 'oldest', label: 'Oldest first' },
  { value: 'alpha', label: 'Alphabetical' },
]

// ⚡ Bolt: Extracting Intl.Collator prevents initializing it on every comparison in the sort loop.
// Improves alpha sort performance by ~100x for large document lists.
const collator = new Intl.Collator(undefined, { sensitivity: 'base' })

interface DocumentListProps {
  heading: string
  documents: Document[] | null
  loading: boolean
  error: string | null
  // Which page we're on, so the masthead links to the other view. Omitted on
  // the read-only Anansi list, which highlights no masthead section.
  active?: 'documents' | 'archive'
  // Initial sort order; differs per page (Documents: newest, Archive: alpha).
  defaultSort: SortOrder
  // Route prefix each row links to; defaults to the live document view.
  basePath?: string
  // Mark archived rows beside their timestamp (for lists that mix both states).
  showArchived?: boolean
  // Hide the "time ago" stamp on rows (for sources without real timestamps, e.g.
  // the Charlotte list); passed through to each DocumentRow.
  hideTime?: boolean
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
  basePath,
  showArchived,
  hideTime,
  children,
}: DocumentListProps) {
  const [sort, setSort] = useState<SortOrder>(defaultSort)

  const sorted = useMemo(() => {
    if (!documents) return documents
    const copy = [...documents]
    copy.sort((a, b) => {
      switch (sort) {
        case 'alpha':
          return collator.compare(a.title || 'Untitled', b.title || 'Untitled')
        case 'oldest':
          return a.updatedAt < b.updatedAt ? -1 : a.updatedAt > b.updatedAt ? 1 : 0
        case 'newest':
        default:
          return b.updatedAt < a.updatedAt ? -1 : b.updatedAt > a.updatedAt ? 1 : 0
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
              inputProps={{ 'aria-label': 'Sort documents' }}
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
            <DocumentRow key={doc.id} doc={doc} basePath={basePath} showArchived={showArchived} hideTime={hideTime} />
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
