import { useEffect, useState } from 'react'
import { Box, Typography } from '@mui/material'
import type { StreamBuffer } from '../lib/streamBuffer'
import { fonts } from '../theme'

// StreamingPreview renders the in-flight run as a response-style <details>
// block. It mounts the instant a run starts (before any text arrives), so the
// new response's block shows up immediately — even with streaming off, where
// text only flushes at tool-call boundaries. It owns the accumulating text
// state, so the (frequent) per-delta re-renders stay confined to this small
// component rather than the whole BlockCard.
export default function StreamingPreview({ stream }: { stream: StreamBuffer }) {
  const [text, setText] = useState<string | null>(null)

  useEffect(() => stream.subscribe(setText), [stream])

  return (
    <Box component="details" open sx={{ mt: 3 }}>
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
          running…
        </Typography>
      </Box>
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
        {text ?? ''}
        <Box component="span" sx={{ opacity: 0.5 }}>
          ▌
        </Box>
      </Typography>
    </Box>
  )
}
