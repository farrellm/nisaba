import { memo } from 'react'
import { Box, Chip, Divider, Link as MuiLink, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { timeAgo } from '../lib/relativeTime'
import { fonts } from '../theme'
import type { Document } from '../api/types'

interface DocumentRowProps {
  doc: Document
  // When true, an archived document is marked beside its timestamp. Off by default
  // so the Documents and Archive lists (single-state pages) stay unmarked.
  showArchived?: boolean
  // Route prefix the title links to; defaults to the live document view. The
  // read-only Anansi list passes '/anansi' to open the legacy viewer.
  basePath?: string
  // Hide the "time ago" stamp for sources without meaningful timestamps (the
  // Charlotte list, whose CLI only exposes file names). The archived marker remains.
  hideTime?: boolean
}

// DocumentRow is one ledger line in a list of documents: a serif title that links
// to the document, a dotted leader, and a mono "time ago" stamp. Shared by the
// Documents/Archive lists and the Labels index so rows look identical everywhere.
// ⚡ Bolt: Wrapping in React.memo prevents expensive re-renders when the list state changes.
const DocumentRow = memo(function DocumentRow({
  doc,
  showArchived = false,
  basePath = '/documents',
  hideTime = false,
}: DocumentRowProps) {
  // With hideTime, show only the bare "archived" marker (no timestamp), else the
  // usual "time ago" optionally prefixed with "archived · ".
  const stamp = hideTime
    ? showArchived && doc.isArchived
      ? 'archived'
      : ''
    : showArchived && doc.isArchived
      ? `archived · ${timeAgo(doc.updatedAt)}`
      : timeAgo(doc.updatedAt)
  return (
    <MuiLink
      component={RouterLink}
      to={`${basePath}/${doc.id}`}
      underline="none"
      color="inherit"
      sx={{
        display: 'block',
        '&:hover .doc-title, &:focus-visible .doc-title': { color: 'primary.main' },
      }}
    >
      <Box sx={{ py: 1.75 }}>
        <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 2 }}>
          <Typography
            className="doc-title"
            sx={{ fontFamily: fonts.display, fontSize: '1.15rem', transition: 'color 120ms' }}
          >
            {doc.title || 'Untitled'}
          </Typography>
          <Box
            sx={{
              flex: 1,
              borderBottom: '1px dotted',
              borderColor: 'divider',
              transform: 'translateY(-3px)',
            }}
          />
          <Typography
            sx={{
              fontFamily: fonts.mono,
              fontSize: '0.8rem',
              color: 'text.secondary',
              whiteSpace: 'nowrap',
            }}
          >
            {stamp}
          </Typography>
        </Box>
        {(doc.labels ?? []).length > 0 && (
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mt: 0.75 }}>
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
          </Box>
        )}
      </Box>
      <Divider sx={{ borderStyle: 'dotted' }} />
    </MuiLink>
  )
})

export default DocumentRow
