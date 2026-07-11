import { useEffect, useMemo, useState } from 'react'
import { Box, Divider, Typography } from '@mui/material'
import { useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { DocumentDetail } from '../api/types'
import StatusLine from '../components/StatusLine'
import { usePageTitle } from '../lib/usePageTitle'
import { wordCount, wordDiff } from '../lib/wordDiff'
import { fonts } from '../theme'

const noticeSx = { fontSize: '0.85rem' } as const

// A muted brick red for deletions — desaturated so it sits inside the warm
// editorial palette rather than shouting like the MUI error color.
const STRUCK = '#B23A2E'

// BlockAttributeDiffPage is a standalone, chrome-free "collation": a full-page,
// unified word-level diff between one block's value for an attribute (the
// baseline) and the document's shared value for the same key (the variant).
// Opened in its own tab from the diff icon in BlockCard.
export default function BlockAttributeDiffPage() {
  const { id, blockId, key = '' } = useParams()
  usePageTitle(key ? `${key} · diff` : null)

  const [doc, setDoc] = useState<DocumentDetail | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .get<DocumentDetail>(`/api/documents/${id}`)
      .then(setDoc)
      .catch(() => setError('Could not load this comparison.'))
  }, [id])

  const block = (doc?.blocks ?? []).find((b) => b.id === Number(blockId))
  const before = block?.attributes?.[key] ?? '' // this block (baseline)
  const after = doc?.attributes?.[key] ?? '' // document (variant)

  const segments = useMemo(() => wordDiff(before, after), [before, after])
  const removed = useMemo(
    () => segments.filter((s) => s.type === 'remove').reduce((n, s) => n + wordCount(s.text), 0),
    [segments],
  )
  const added = useMemo(
    () => segments.filter((s) => s.type === 'add').reduce((n, s) => n + wordCount(s.text), 0),
    [segments],
  )

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Box
        component="main"
        sx={{ maxWidth: 760, mx: 'auto', px: { xs: 3, md: 4 }, py: { xs: 6, md: 10 } }}
      >
        {error ? (
          <StatusLine sx={noticeSx}>{error}</StatusLine>
        ) : !doc ? null : !block ? (
          <StatusLine sx={noticeSx}>This block no longer exists.</StatusLine>
        ) : (
          <>
            {/* Collation header */}
            <Typography variant="overline" sx={{ color: 'primary.main', display: 'block' }}>
              {key}
            </Typography>
            <Typography
              sx={{
                fontFamily: fonts.mono,
                fontSize: '0.8rem',
                color: 'text.secondary',
                mt: 0.5,
              }}
            >
              This block&nbsp;&rarr;&nbsp;Document
            </Typography>
            <Typography sx={{ fontFamily: fonts.mono, fontSize: '0.8rem', mt: 0.75 }}>
              <Box component="span" sx={{ color: removed ? STRUCK : 'text.disabled' }}>
                &minus;{removed} {removed === 1 ? 'word' : 'words'}
              </Box>
              <Box component="span" sx={{ color: 'text.disabled', mx: 1.25 }}>
                /
              </Box>
              <Box component="span" sx={{ color: added ? 'primary.main' : 'text.disabled' }}>
                +{added} {added === 1 ? 'word' : 'words'}
              </Box>
            </Typography>
            <Divider sx={{ mt: 2.5, mb: 4 }} />

            {/* Body */}
            {before === '' && after === '' ? (
              <StatusLine sx={noticeSx}>Nothing to compare — neither side has a value.</StatusLine>
            ) : before === after ? (
              <StatusLine sx={noticeSx}>
                No differences — the document matches this block.
              </StatusLine>
            ) : (
              <>
                {after === '' && (
                  <StatusLine sx={{ ...noticeSx, mb: 3 }}>
                    No document value for this key yet.
                  </StatusLine>
                )}
                <Box
                  sx={{
                    fontFamily: fonts.body,
                    fontSize: '1.05rem',
                    lineHeight: 1.7,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    color: 'text.primary',
                  }}
                >
                  {segments.map((seg, i) => {
                    if (seg.type === 'add') {
                      return (
                        <Box
                          component="ins"
                          key={i}
                          sx={{
                            color: 'primary.main',
                            textDecoration: 'underline',
                            textUnderlineOffset: '0.18em',
                            bgcolor: 'transparent',
                          }}
                        >
                          {seg.text}
                        </Box>
                      )
                    }
                    if (seg.type === 'remove') {
                      return (
                        <Box
                          component="del"
                          key={i}
                          sx={{ color: STRUCK, textDecoration: 'line-through' }}
                        >
                          {seg.text}
                        </Box>
                      )
                    }
                    return <span key={i}>{seg.text}</span>
                  })}
                </Box>
              </>
            )}
          </>
        )}
      </Box>
    </Box>
  )
}
