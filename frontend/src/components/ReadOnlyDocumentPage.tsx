import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  Box,
  Container,
  IconButton,
  InputAdornment,
  Link as MuiLink,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
import DataObjectIcon from '@mui/icons-material/DataObject'
import UnfoldMore from '@mui/icons-material/UnfoldMore'
import { api } from '../api/client'
import type { Block, DocumentDetail, Response } from '../api/types'
import Masthead from './Masthead'
import Markdown from './Markdown'
import { parseResponseSegments } from '../lib/responseSegments'
import { usePageTitle } from '../lib/usePageTitle'
import { fonts } from '../theme'

const EMPTY_ATTRIBUTES: Record<string, string> = {}

const postLinkSx = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 0.5,
  color: 'primary.main',
  textDecoration: 'none',
  '&:hover': { textDecoration: 'underline' },
} as const

// Shared <summary> styling: a flex row whose default disclosure marker is hidden
// so the mode/attribute header + dotted leader read like the live document page.
const summarySx = {
  display: 'flex',
  alignItems: 'baseline',
  gap: 2,
  mb: 2,
  cursor: 'pointer',
  listStyle: 'none',
  '&::-webkit-details-marker': { display: 'none' },
} as const

const leaderSx = {
  flex: 1,
  borderBottom: '1px dotted',
  borderColor: 'divider',
  transform: 'translateY(-3px)',
} as const

// ReadOnlyDocumentPage renders a single document read-only, mirroring the live
// document page's structure (collapsible block cards with mode headers,
// structured/raw response views, and a collapsible Attributes section) but with no
// editing affordances. It is shared by the legacy "Anansi" (reflex.db) and
// "Charlotte" (charlotte-cli) browsers; the only difference is the API base path it
// fetches the document by id from.
export default function ReadOnlyDocumentPage({ apiBase }: { apiBase: string }) {
  const { id } = useParams<{ id: string }>()
  const [doc, setDoc] = useState<DocumentDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  usePageTitle(doc ? doc.title || 'Untitled' : null)

  useEffect(() => {
    setDoc(null)
    setError(null)
    api
      .get<DocumentDetail>(`${apiBase}/${id}`)
      .then(setDoc)
      .catch((e: unknown) => setError(String(e)))
  }, [apiBase, id])

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

            {(doc.blocks ?? []).map((block, i, arr) => (
              <BlockSection key={block.id} block={block} defaultOpen={i === arr.length - 1} />
            ))}

            <Attributes attributes={doc.attributes ?? EMPTY_ATTRIBUTES} />
          </>
        )}
      </Container>
    </Box>
  )
}

// ReadOnlyField renders a key/value as a read-only TextField, collapsing long
// values to a truncated preview that expands on click (mirroring BlockCard /
// DocumentAttributes).
function ReadOnlyField({
  fieldKey,
  value,
  expanded,
  onReveal,
}: {
  fieldKey: string
  value: string
  expanded: boolean
  onReveal: () => void
}) {
  const collapsed = !expanded && value.length > 80
  if (collapsed) {
    return (
      <TextField
        label={fieldKey}
        value={`${value.slice(0, 40)}…`}
        onClick={onReveal}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onReveal()
          }
        }}
        inputProps={{ tabIndex: 0 }}
        InputProps={{
          readOnly: true,
          endAdornment: (
            <InputAdornment position="end" sx={{ color: 'text.secondary' }}>
              <UnfoldMore fontSize="small" />
            </InputAdornment>
          ),
          sx: {
            cursor: 'pointer',
            '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: 'primary.main' },
            '&:focus-within .MuiOutlinedInput-notchedOutline': { borderColor: 'primary.main', borderWidth: 2 },
          },
        }}
      />
    )
  }
  return (
    <TextField
      label={fieldKey}
      value={value}
      InputProps={{ readOnly: true }}
      multiline
      minRows={1}
    />
  )
}

// BlockSection is a read-only port of BlockCard: a collapsible card with the
// mode header, the block's input key/values, and its responses.
function BlockSection({ block, defaultOpen }: { block: Block; defaultOpen: boolean }) {
  const keys = Object.keys(block.attributes)
  const responses = block.responses ?? []
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  // The last block's newest response opens in the structured view by default.
  const [structured, setStructured] = useState<Set<number>>(() =>
    defaultOpen && responses.length > 0
      ? new Set([responses[responses.length - 1].id])
      : new Set(),
  )

  function reveal(key: string) {
    setExpanded((prev) => new Set(prev).add(key))
  }

  function toggleStructured(id: number) {
    setStructured((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  return (
    <Box component="section" sx={{ pt: 4 }}>
      <Box
        component="details"
        {...(defaultOpen ? { open: true } : {})}
        sx={{ '&[open]': { borderBottom: '1px dotted', borderColor: 'divider', pb: 4 } }}
      >
        <Box component="summary" sx={summarySx}>
          <Typography
            variant="overline"
            sx={{ fontFamily: fonts.mono, color: 'primary.main', whiteSpace: 'nowrap' }}
          >
            {block.mode}
          </Typography>
          <Box sx={leaderSx} />
        </Box>

        {keys.length > 0 && (
          <Stack spacing={2}>
            {keys.map((key) => (
              <ReadOnlyField
                key={key}
                fieldKey={key}
                value={block.attributes[key] ?? ''}
                expanded={expanded.has(key)}
                onReveal={() => reveal(key)}
              />
            ))}
          </Stack>
        )}

        {responses.length > 0 && (
          <Stack spacing={1.5} sx={{ mt: 3 }}>
            {responses
              .slice()
              .reverse()
              .map((response, idx) => (
                <ResponseDetails
                  key={response.id}
                  response={response}
                  open={idx === 0}
                  structured={structured.has(response.id)}
                  onToggle={() => toggleStructured(response.id)}
                />
              ))}
          </Stack>
        )}
      </Box>
    </Box>
  )
}

// ResponseDetails renders one response as a collapsible with the model as its
// header and a structured/raw toggle (read-only port of BlockCard's response).
function ResponseDetails({
  response,
  open,
  structured,
  onToggle,
}: {
  response: Response
  open: boolean
  structured: boolean
  onToggle: () => void
}) {
  return (
    <Box component="details" {...(open ? { open: true } : {})}>
      <Box
        component="summary"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          cursor: 'pointer',
          listStyle: 'none',
          '&::-webkit-details-marker': { display: 'none' },
          mb: 1,
        }}
      >
        <Typography
          variant="overline"
          sx={{ fontFamily: fonts.mono, color: 'text.secondary', fontSize: '0.7rem' }}
        >
          {response.model || 'no model'}
        </Typography>
        <Box sx={{ flexGrow: 1 }} />
        <Tooltip title={structured ? 'Raw view' : 'Structured view'}>
          <IconButton
            size="small"
            aria-label={structured ? 'Show raw response' : 'Show structured response'}
            onClick={(e) => {
              e.preventDefault()
              onToggle()
            }}
            sx={{
              color: structured ? 'primary.main' : 'text.disabled',
              '&:hover': { color: 'primary.main' },
            }}
          >
            <DataObjectIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Box>
      {structured ? (
        <Box sx={{ bgcolor: 'action.hover', borderRadius: 2, p: 2 }}>
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
      ) : (
        <Typography
          sx={{
            fontFamily: fonts.mono,
            fontSize: '0.85rem',
            whiteSpace: 'pre-wrap',
            bgcolor: 'action.hover',
            borderRadius: 2,
            p: 2,
          }}
        >
          {response.value}
        </Typography>
      )}
    </Box>
  )
}

// Attributes renders the document's shared key/value namespace read-only,
// mirroring DocumentAttributes' collapsible section.
function Attributes({ attributes }: { attributes: Record<string, string> }) {
  // ⚡ Bolt: Memoize sorted keys to prevent allocating and sorting an array on every re-render.
  const keys = useMemo(() => Object.keys(attributes).sort(), [attributes])
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  if (keys.length === 0) return null
  return (
    <Box component="section" sx={{ py: 4, borderTop: '1px dotted', borderColor: 'divider', mt: 2 }}>
      <Box component="details" sx={{ '&[open]': { borderBottom: '1px dotted', borderColor: 'divider', pb: 4 } }}>
        <Box component="summary" sx={{ ...summarySx, mb: 3 }}>
          <Typography
            variant="overline"
            sx={{ fontFamily: fonts.mono, color: 'primary.main', whiteSpace: 'nowrap' }}
          >
            Attributes
          </Typography>
          <Box sx={leaderSx} />
        </Box>

        <Stack spacing={2}>
          {keys.map((key) => (
            <ReadOnlyField
              key={key}
              fieldKey={key}
              value={attributes[key] ?? ''}
              expanded={expanded.has(key)}
              onReveal={() => setExpanded((prev) => new Set(prev).add(key))}
            />
          ))}
        </Stack>
      </Box>
    </Box>
  )
}
