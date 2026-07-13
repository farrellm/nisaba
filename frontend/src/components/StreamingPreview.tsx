import { useEffect, useState } from 'react'
import { Box, Typography } from '@mui/material'
import type { StreamBuffer } from '../lib/streamBuffer'
import { fonts } from '../theme'

// StreamingPreview renders the live text of an in-flight streamed run. It owns
// the accumulating text state, so the (frequent) per-delta re-renders are
// confined to this small component rather than the whole BlockCard. Nothing is
// shown until the first delta arrives, matching the pre-stream quiet.
export default function StreamingPreview({ stream }: { stream: StreamBuffer }) {
  const [text, setText] = useState<string | null>(null)

  useEffect(() => stream.subscribe(setText), [stream])

  if (text === null) return null
  return (
    <Box sx={{ mt: 3 }}>
      <Typography
        variant="overline"
        sx={{
          fontFamily: fonts.mono,
          color: 'text.secondary',
          fontSize: '0.7rem',
          display: 'block',
          mb: 1,
        }}
      >
        streaming…
      </Typography>
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
        {text}
        <Box component="span" sx={{ opacity: 0.5 }}>
          ▌
        </Box>
      </Typography>
    </Box>
  )
}
