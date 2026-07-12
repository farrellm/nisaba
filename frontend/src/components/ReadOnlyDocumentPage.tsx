import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Alert,
  Box,
  CircularProgress,
  Container,
  Fab,
  Link as MuiLink,
  Snackbar,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material'
import SaveAltIcon from '@mui/icons-material/SaveAlt'
import { api } from '../api/client'
import { errorMessage } from '../lib/errors'
import { EMPTY_ATTRIBUTES, type Block, type Document, type DocumentDetail } from '../api/types'
import CollapsibleValueField from './CollapsibleValueField'
import Masthead from './Masthead'
import ResponseView from './ResponseView'
import StatusLine from './StatusLine'
import { usePageTitle } from '../lib/usePageTitle'
import { fonts } from '../theme'
import { leaderSx, postLinkSx, summarySx } from '../lib/styles'
import { addToSet, toggleSet } from '../lib/sets'
import { useAsyncAction } from '../lib/useAsyncAction'

// ReadOnlyDocumentPage renders a single document read-only, mirroring the live
// document page's structure (collapsible block cards with mode headers,
// structured/raw response views, and a collapsible Attributes section) but with no
// editing affordances. It is shared by the legacy "Anansi" (reflex.db) and
// "Charlotte" (charlotte-cli) browsers; the only difference is the API base path it
// fetches the document by id from.
export default function ReadOnlyDocumentPage({ apiBase }: { apiBase: string }) {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [doc, setDoc] = useState<DocumentDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  const {
    busy: importing,
    error: importError,
    setError: setImportError,
    run: runImport,
  } = useAsyncAction()
  usePageTitle(doc ? doc.title || 'Untitled' : null)

  useEffect(() => {
    setDoc(null)
    setError(null)
    api
      .get<DocumentDetail>(`${apiBase}/${id}`)
      .then(setDoc)
      .catch((e: unknown) => setError(errorMessage(e)))
  }, [apiBase, id])

  function handleImport() {
    runImport(
      async () => {
        const imported = await api.post<Document>(`${apiBase}/${id}/import`)
        navigate(`/documents/${imported.id}`)
      },
      { fallback: 'Could not import document', keepBusyOnSuccess: true },
    )
  }

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead />
      <Container maxWidth="md" sx={{ pt: { xs: 7, md: 12 }, pb: 12 }}>
        {error ? (
          <StatusLine tone="error">{error}</StatusLine>
        ) : !doc ? (
          <StatusLine>Loading…</StatusLine>
        ) : (
          <>
            <Box sx={{ mb: 4 }}>
              <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)' }}>
                {doc.title || 'Untitled'}
              </Typography>
              <Box
                sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 2, mt: 1.5 }}
              >
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

      {doc && (
        <Tooltip title="Copy this document into your Nisaba documents" placement="left">
          <Fab
            variant="extended"
            color="primary"
            aria-label="Import into Nisaba"
            disabled={importing}
            onClick={handleImport}
            sx={{ position: 'fixed', bottom: 32, right: 32 }}
          >
            {importing ? (
              <CircularProgress size={20} color="inherit" sx={{ mr: 1 }} />
            ) : (
              <SaveAltIcon sx={{ mr: 1 }} />
            )}
            Import into Nisaba
          </Fab>
        </Tooltip>
      )}

      <Snackbar
        open={importError !== null}
        autoHideDuration={8000}
        onClose={() => setImportError(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity="error" onClose={() => setImportError(null)} sx={{ maxWidth: 560 }}>
          {importError}
        </Alert>
      </Snackbar>
    </Box>
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
    defaultOpen && responses.length > 0 ? new Set([responses[responses.length - 1].id]) : new Set(),
  )

  const reveal = (key: string) => setExpanded((prev) => addToSet(prev, key))
  const toggleStructured = (id: number) => setStructured((prev) => toggleSet(prev, id))

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
              <CollapsibleValueField
                key={key}
                label={key}
                value={block.attributes[key] ?? ''}
                expanded={expanded.has(key)}
                onExpand={() => reveal(key)}
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
                <ResponseView
                  key={response.id}
                  response={response}
                  defaultOpen={idx === 0}
                  structured={structured.has(response.id)}
                  onToggleStructured={() => toggleStructured(response.id)}
                />
              ))}
          </Stack>
        )}
      </Box>
    </Box>
  )
}

// Attributes renders the document's shared key/value namespace read-only,
// mirroring DocumentAttributes' collapsible section.
function Attributes({ attributes }: { attributes: Record<string, string> }) {
  const keys = useMemo(() => Object.keys(attributes).sort(), [attributes])
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  if (keys.length === 0) return null
  return (
    <Box component="section" sx={{ py: 4, borderTop: '1px dotted', borderColor: 'divider', mt: 2 }}>
      <Box
        component="details"
        sx={{ '&[open]': { borderBottom: '1px dotted', borderColor: 'divider', pb: 4 } }}
      >
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
            <CollapsibleValueField
              key={key}
              label={key}
              value={attributes[key] ?? ''}
              expanded={expanded.has(key)}
              onExpand={() => setExpanded((prev) => addToSet(prev, key))}
            />
          ))}
        </Stack>
      </Box>
    </Box>
  )
}
