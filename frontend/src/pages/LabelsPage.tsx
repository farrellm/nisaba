import { useMemo, useState } from 'react'
import { Box, Container, Divider, Link as MuiLink, Typography } from '@mui/material'
import type { Document } from '../api/types'
import DeleteLabelDialog from '../components/DeleteLabelDialog'
import DocumentRow from '../components/DocumentRow'
import Masthead from '../components/Masthead'
import RenameLabelDialog from '../components/RenameLabelDialog'
import StatusLine from '../components/StatusLine'
import { listStatusSx } from '../lib/styles'
import { collator } from '../lib/text'
import { useFetch } from '../lib/useFetch'
import { usePageTitle } from '../lib/usePageTitle'
import { fonts } from '../theme'

const newestFirst = (a: Document, b: Document) =>
  b.updatedAt < a.updatedAt ? -1 : b.updatedAt > a.updatedAt ? 1 : 0

// LabelsPage is the writer's index: one section per label, every document carrying
// it listed beneath (archived ones marked), with per-label rename and delete. Labels
// auto-vanish once they're on no document, so every section here has at least one doc.
export default function LabelsPage() {
  usePageTitle('Labels')
  // ?archived=true returns every document (active and archived), which is exactly
  // the set the index needs — archived ones are marked per-row via doc.isArchived.
  const {
    data: docs,
    error,
    loading,
    reload,
  } = useFetch<Document[]>('/api/documents?archived=true')
  const [renaming, setRenaming] = useState<string | null>(null)
  const [deleting, setDeleting] = useState<string | null>(null)

  // Group documents under each label name, alphabetical by label, newest-first within.
  const sections = useMemo(() => {
    if (!docs) return null
    const byLabel = new Map<string, Document[]>()
    for (const doc of docs) {
      for (const label of doc.labels ?? []) {
        const bucket = byLabel.get(label)
        if (bucket) bucket.push(doc)
        else byLabel.set(label, [doc])
      }
    }
    return [...byLabel.entries()]
      .sort((a, b) => collator.compare(a[0], b[0]))
      .map(([name, group]) => ({ name, docs: group.sort(newestFirst) }))
  }, [docs])

  const labelNames = useMemo(() => sections?.map((s) => s.name) ?? [], [sections])

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead />

      <Container maxWidth="md" sx={{ pt: { xs: 5, md: 8 }, pb: 12 }}>
        <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)', mb: 4 }}>
          Labels
        </Typography>
        <Divider sx={{ mb: 1 }} />

        {error ? (
          <StatusLine tone="error" sx={listStatusSx}>
            {error}
          </StatusLine>
        ) : loading ? (
          <StatusLine sx={listStatusSx}>Loading…</StatusLine>
        ) : sections && sections.length > 0 ? (
          sections.map((section) => (
            <Box
              key={section.name}
              sx={{
                mb: 5,
                '&:hover .label-controls, &:focus-within .label-controls': { opacity: 1 },
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 2 }}>
                <Typography
                  component="h2"
                  sx={{
                    fontFamily: fonts.display,
                    fontWeight: 500,
                    fontSize: '1.5rem',
                    letterSpacing: '-0.01em',
                  }}
                >
                  {section.name}
                </Typography>
                <Box
                  sx={{
                    flex: 1,
                    borderBottom: '1px dotted',
                    borderColor: 'divider',
                    transform: 'translateY(-4px)',
                  }}
                />
                <Typography
                  sx={{
                    fontFamily: fonts.mono,
                    fontSize: '0.75rem',
                    color: 'text.secondary',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {section.docs.length === 1 ? '1 story' : `${section.docs.length} stories`}
                </Typography>
                <Box
                  className="label-controls"
                  sx={{
                    display: 'flex',
                    gap: 1.5,
                    opacity: { xs: 1, md: 0.45 },
                    transition: 'opacity 120ms',
                  }}
                >
                  <ControlLink onClick={() => setRenaming(section.name)}>Rename</ControlLink>
                  <ControlLink onClick={() => setDeleting(section.name)}>Delete</ControlLink>
                </Box>
              </Box>

              <Box sx={{ mt: 0.5 }}>
                {section.docs.map((doc) => (
                  <DocumentRow key={doc.id} doc={doc} showArchived />
                ))}
              </Box>
            </Box>
          ))
        ) : (
          <StatusLine sx={listStatusSx}>
            No labels yet. Add labels to a document to start your index.
          </StatusLine>
        )}
      </Container>

      {renaming !== null && (
        <RenameLabelDialog
          name={renaming}
          docCount={sections?.find((s) => s.name === renaming)?.docs.length ?? 0}
          otherNames={labelNames.filter((n) => n !== renaming)}
          onClose={() => setRenaming(null)}
          onDone={() => {
            setRenaming(null)
            reload()
          }}
        />
      )}

      {deleting !== null && (
        <DeleteLabelDialog
          name={deleting}
          docCount={sections?.find((s) => s.name === deleting)?.docs.length ?? 0}
          onClose={() => setDeleting(null)}
          onDone={() => {
            setDeleting(null)
            reload()
          }}
        />
      )}
    </Box>
  )
}

// ControlLink is the small mono action used in a section header (Rename / Delete).
function ControlLink({ onClick, children }: { onClick: () => void; children: string }) {
  return (
    <MuiLink
      component="button"
      type="button"
      onClick={onClick}
      underline="hover"
      sx={{
        fontFamily: fonts.mono,
        fontSize: '0.72rem',
        textTransform: 'uppercase',
        letterSpacing: '0.08em',
        color: 'text.secondary',
        '&:hover, &:focus-visible': { color: 'primary.main' },
      }}
    >
      {children}
    </MuiLink>
  )
}
