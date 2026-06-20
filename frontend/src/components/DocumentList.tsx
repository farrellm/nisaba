import type { ReactNode } from 'react'
import { Box, Container, Divider, Link as MuiLink, Stack, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { timeAgo } from '../lib/relativeTime'
import { fonts } from '../theme'
import AccountMenu from './AccountMenu'
import type { Document } from '../api/types'

interface DocumentListProps {
  eyebrow: string
  heading: string
  documents: Document[] | null
  loading: boolean
  error: string | null
  // Which page we're on, so the masthead links to the other view.
  active: 'documents' | 'archive'
  // Optional extra content (e.g. a floating action button).
  children?: ReactNode
}

// DocumentList renders the masthead and a ledger-style list of documents shared
// by the Documents and Archive pages.
export default function DocumentList({
  eyebrow,
  heading,
  documents,
  loading,
  error,
  active,
  children,
}: DocumentListProps) {
  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      {/* Masthead bar */}
      <Box
        component="header"
        sx={{
          borderBottom: '1px solid',
          borderColor: 'divider',
          px: { xs: 3, md: 5 },
          py: 2,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <Stack direction="row" spacing={3} alignItems="baseline">
          <MuiLink component={RouterLink} to="/" underline="none" color="inherit">
            <Typography
              sx={{ fontFamily: fonts.display, fontWeight: 600, fontSize: '1.5rem', letterSpacing: '-0.02em' }}
            >
              Nisaba
            </Typography>
          </MuiLink>
          <MuiLink
            component={RouterLink}
            to={active === 'documents' ? '/archive' : '/documents'}
            underline="hover"
            sx={{ fontFamily: fonts.mono, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.08em' }}
          >
            {active === 'documents' ? 'Archive' : 'Documents'}
          </MuiLink>
          <MuiLink
            component={RouterLink}
            to="/reddit"
            underline="hover"
            sx={{ fontFamily: fonts.mono, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.08em' }}
          >
            Prompts
          </MuiLink>
        </Stack>
        <AccountMenu />
      </Box>

      <Container maxWidth="md" sx={{ pt: { xs: 5, md: 8 }, pb: 12 }}>
        <Typography variant="overline" sx={{ color: 'primary.main', display: 'block', mb: 2 }}>
          {eyebrow}
        </Typography>
        <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)', mb: 4 }}>
          {heading}
        </Typography>

        <Divider sx={{ mb: 1 }} />

        {error ? (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'error.main', py: 1.5 }}>
            {error}
          </Typography>
        ) : loading ? (
          <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.9rem', color: 'text.secondary', py: 1.5 }}>
            Loading…
          </Typography>
        ) : documents && documents.length > 0 ? (
          documents.map((doc) => (
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
