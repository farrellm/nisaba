import type { ReactNode } from 'react'
import { Typography, type SxProps, type Theme } from '@mui/material'
import { fonts } from '../theme'

interface StatusLineProps {
  // muted for loading/empty states, error for failures.
  tone?: 'muted' | 'error'
  // Per-site size/spacing (the mono face and tone color are the shared part).
  sx?: SxProps<Theme>
  children: ReactNode
}

// StatusLine is the small mono line used for loading / empty / error states
// throughout the app's ledger-style pages.
export default function StatusLine({ tone = 'muted', sx, children }: StatusLineProps) {
  return (
    <Typography
      sx={[
        { fontFamily: fonts.mono, color: tone === 'error' ? 'error.main' : 'text.secondary' },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      {children}
    </Typography>
  )
}
