import { Box, Divider, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import { fonts } from '../theme'

// Editorial masthead split used by the login and create-account pages. On wide
// screens the identity sits left of a hairline rule with the form on the right;
// it stacks on mobile.
export default function AuthLayout({
  eyebrow,
  children,
}: {
  eyebrow: string
  children: ReactNode
}) {
  return (
    <Box
      sx={{
        minHeight: '100vh',
        bgcolor: 'background.default',
        display: 'grid',
        placeItems: 'center',
        px: 3,
        py: { xs: 6, md: 0 },
      }}
    >
      <Box
        sx={{
          width: '100%',
          maxWidth: 880,
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
          columnGap: { md: 7 },
          rowGap: { xs: 5, md: 0 },
          alignItems: 'center',
        }}
      >
        {/* Identity / masthead */}
        <Box>
          <Typography variant="overline" sx={{ color: 'primary.main', display: 'block', mb: 2 }}>
            {eyebrow}
          </Typography>
          <Typography
            sx={{
              fontFamily: fonts.display,
              fontWeight: 600,
              letterSpacing: '-0.03em',
              lineHeight: 0.95,
              fontSize: 'clamp(3.5rem, 9vw, 5.5rem)',
            }}
          >
            Nisaba
          </Typography>
          <Divider sx={{ my: 3, maxWidth: 280 }} />
          <Typography variant="body1" color="text.secondary" sx={{ maxWidth: 320 }}>
            A scribe's table for your documents — write, label, and keep them in order.
          </Typography>
        </Box>

        {/* Form */}
        <Box
          sx={{
            borderLeft: { md: '1px solid' },
            borderColor: { md: 'divider' },
            pl: { md: 7 },
          }}
        >
          {children}
        </Box>
      </Box>
    </Box>
  )
}
