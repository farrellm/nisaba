import { Box, Button, Container, Stack, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { fonts } from '../theme'
import AccountMenu from '../components/AccountMenu'
import { usePageTitle } from '../lib/usePageTitle'

export default function IndexPage() {
  usePageTitle()
  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      {/* Masthead bar */}
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
        <Typography
          sx={{ fontFamily: fonts.display, fontWeight: 600, fontSize: '1.5rem', letterSpacing: '-0.02em' }}
        >
          Nisaba
        </Typography>
        <AccountMenu />
      </Box>

      <Container maxWidth="md" sx={{ pt: { xs: 7, md: 12 }, pb: 8 }}>
        <Stack direction="row" sx={{ flexWrap: 'wrap', gap: 2 }}>
          <Button component={RouterLink} to="/documents" variant="contained" size="large">
            Documents
          </Button>
          <Button component={RouterLink} to="/archive" variant="outlined" size="large">
            Archive
          </Button>
          <Button component={RouterLink} to="/reddit" variant="outlined" size="large">
            Prompts
          </Button>
        </Stack>
      </Container>
    </Box>
  )
}
