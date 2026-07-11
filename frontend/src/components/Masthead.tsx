import { Box, Link as MuiLink, Stack, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { fonts } from '../theme'
import { navLinkSx } from '../lib/styles'
import AccountMenu from './AccountMenu'

type Section = 'documents' | 'archive' | 'prompts'

const NAV: { key: Section; label: string; to: string }[] = [
  { key: 'documents', label: 'Documents', to: '/documents' },
  { key: 'archive', label: 'Archive', to: '/archive' },
  { key: 'prompts', label: 'Prompts', to: '/reddit' },
]

// Masthead is the shared top bar across the app: the Nisaba wordmark (links home),
// the global nav, and the account menu. The current section is shown muted and
// non-interactive so the user can see where they are.
export default function Masthead({ active }: { active?: Section }) {
  return (
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
            sx={{
              fontFamily: fonts.display,
              fontWeight: 600,
              fontSize: '1.5rem',
              letterSpacing: '-0.02em',
            }}
          >
            Nisaba
          </Typography>
        </MuiLink>
        {NAV.map((item) =>
          item.key === active ? (
            <Typography
              key={item.key}
              component="span"
              aria-current="page"
              sx={{ ...navLinkSx, color: 'text.secondary' }}
            >
              {item.label}
            </Typography>
          ) : (
            <MuiLink
              key={item.key}
              component={RouterLink}
              to={item.to}
              underline="hover"
              sx={navLinkSx}
            >
              {item.label}
            </MuiLink>
          ),
        )}
      </Stack>
      <AccountMenu />
    </Box>
  )
}
