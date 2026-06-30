import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Box, Container, Divider, Link as MuiLink, Typography } from '@mui/material'
import { api } from '../api/client'
import type { Block, DocumentDetail } from '../api/types'
import Masthead from '../components/Masthead'
import Markdown from '../components/Markdown'
import { parseResponseSegments } from '../lib/responseSegments'
import { usePageTitle } from '../lib/usePageTitle'
import { fonts } from '../theme'

const postLinkSx = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 0.5,
  color: 'primary.main',
  textDecoration: 'none',
  '&:hover': { textDecoration: 'underline' },
} as const

const labelSx = {
  fontFamily: fonts.mono,
  fontSize: '0.75rem',
  color: 'text.secondary',
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
} as const

// AnansiDocumentPage renders a single legacy reflex.db document read-only:
// title, each block's responses (parsed and rendered as markdown, mirroring the
// live BlockCard), and the document's attributes. It has no editing affordances.
export default function AnansiDocumentPage() {
  const { id } = useParams<{ id: string }>()
  const [doc, setDoc] = useState<DocumentDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  usePageTitle(doc ? doc.title || 'Untitled' : null)

  useEffect(() => {
    setDoc(null)
    setError(null)
    api
      .get<DocumentDetail>(`/api/anansi/documents/${id}`)
      .then(setDoc)
      .catch((e: unknown) => setError(String(e)))
  }, [id])

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead />
      <Container maxWidth="md" sx={{ pt: { xs: 7, md: 12 }, pb: 12 }}>
        {error ? (
          <Typography sx={{ fontFamily: fonts.mono, color: 'error.main' }}>{error}</Typography>
        ) : !doc ? (
          <Typography sx={{ fontFamily: fonts.mono, color: 'text.secondary' }}>Loading…</Typography>
        ) : (
          <>
            <Box sx={{ mb: 4 }}>
              <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)' }}>
                {doc.title || 'Untitled'}
              </Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 2, mt: 1.5 }}>
                {doc.url && (
                  <MuiLink
                    href={doc.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    variant="overline"
                    sx={postLinkSx}
                  >
                    Original post ↗
                  </MuiLink>
                )}
                {doc.isArchived && (
                  <Typography variant="overline" sx={{ color: 'text.secondary' }}>
                    Archived
                  </Typography>
                )}
                <Typography variant="overline" sx={{ color: 'text.secondary' }}>
                  Read-only
                </Typography>
              </Box>
            </Box>

            {(doc.blocks ?? []).map((block) => (
              <BlockSection key={block.id} block={block} />
            ))}

            <Attributes attributes={doc.attributes ?? {}} />
          </>
        )}
      </Container>
    </Box>
  )
}

// BlockSection renders one block's mode/model heading and each of its responses.
function BlockSection({ block }: { block: Block }) {
  const responses = block.responses ?? []
  const model = responses.length > 0 ? responses[0].model : ''
  return (
    <Box component="section" sx={{ py: 3, borderTop: '1px dotted', borderColor: 'divider' }}>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1.5, mb: 1.5, ...labelSx }}>
        <span>{block.mode}</span>
        {model && <span style={{ opacity: 0.7 }}>{model}</span>}
      </Box>
      {responses.length === 0 ? (
        <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.85rem', color: 'text.secondary' }}>
          No response.
        </Typography>
      ) : (
        responses.map((response) => (
          <Box key={response.id} sx={{ bgcolor: 'action.hover', borderRadius: 2, p: 2, mb: 2 }}>
            {parseResponseSegments(response.value).map((seg, segIdx) =>
              seg.kind === 'text' ? (
                <Markdown key={segIdx}>{seg.text}</Markdown>
              ) : (
                <Box
                  key={segIdx}
                  component="details"
                  open
                  sx={{ my: 1, '&:first-of-type': { mt: 0 }, '&:last-child': { mb: 0 } }}
                >
                  <Box
                    component="summary"
                    sx={{ cursor: 'pointer', fontFamily: fonts.mono, fontSize: '0.8rem', color: 'text.secondary' }}
                  >
                    {seg.name}
                  </Box>
                  <Box
                    component="blockquote"
                    sx={{ my: 1, ml: 0, pl: 2.5, borderLeft: '3px solid', borderColor: 'divider', color: 'text.secondary' }}
                  >
                    {/* Escape '<' so nested tags render as literal text:
                        react-markdown drops raw HTML. */}
                    <Markdown>{seg.inner.split('<').join('\\<')}</Markdown>
                  </Box>
                </Box>
              ),
            )}
          </Box>
        ))
      )}
    </Box>
  )
}

// Attributes renders the document's shared key/value namespace read-only.
function Attributes({ attributes }: { attributes: Record<string, string> }) {
  const keys = Object.keys(attributes).sort()
  if (keys.length === 0) return null
  return (
    <Box component="section" sx={{ py: 4, borderTop: '1px dotted', borderColor: 'divider', mt: 2 }}>
      <Typography variant="overline" sx={{ color: 'text.secondary' }}>
        Attributes
      </Typography>
      {keys.map((key) => (
        <Box key={key} sx={{ mt: 2 }}>
          <Typography sx={labelSx}>{key}</Typography>
          <Divider sx={{ my: 1, borderStyle: 'dotted' }} />
          <Markdown>{attributes[key].split('<').join('\\<')}</Markdown>
        </Box>
      ))}
    </Box>
  )
}
