import { Box, Card, CardActionArea, Container, Stack, Typography } from '@mui/material'
import type { SvgIconComponent } from '@mui/icons-material'
import DescriptionOutlined from '@mui/icons-material/DescriptionOutlined'
import LocalOfferOutlined from '@mui/icons-material/LocalOfferOutlined'
import Inventory2Outlined from '@mui/icons-material/Inventory2Outlined'
import SearchOutlined from '@mui/icons-material/SearchOutlined'
import AutoAwesomeOutlined from '@mui/icons-material/AutoAwesomeOutlined'
import AutoStoriesOutlined from '@mui/icons-material/AutoStoriesOutlined'
import HistoryEduOutlined from '@mui/icons-material/HistoryEduOutlined'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import { Link as RouterLink } from 'react-router-dom'
import { fonts } from '../theme'
import AccountMenu from '../components/AccountMenu'
import { usePageTitle } from '../lib/usePageTitle'

type NavCard = {
  title: string
  to: string
  Icon: SvgIconComponent
  highlight?: boolean
}

const cards: NavCard[] = [
  { title: 'Documents', to: '/documents', Icon: DescriptionOutlined, highlight: true },
  { title: 'Archive', to: '/archive', Icon: Inventory2Outlined },
  { title: 'Search', to: '/search', Icon: SearchOutlined },
  { title: 'Labels', to: '/labels', Icon: LocalOfferOutlined },
  { title: 'Prompts', to: '/reddit', Icon: AutoAwesomeOutlined },
  { title: 'Anansi', to: '/anansi', Icon: AutoStoriesOutlined },
  { title: 'Charlotte', to: '/charlotte', Icon: HistoryEduOutlined },
]

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
          sx={{
            fontFamily: fonts.display,
            fontWeight: 600,
            fontSize: '1.5rem',
            letterSpacing: '-0.02em',
          }}
        >
          Nisaba
        </Typography>
        <AccountMenu />
      </Box>

      <Container maxWidth="sm" sx={{ pt: { xs: 7, md: 12 }, pb: 8 }}>
        <Stack spacing={2}>
          {cards.map(({ title, to, Icon, highlight }) => (
            <Card
              key={to}
              variant="outlined"
              sx={{
                borderColor: highlight ? 'primary.main' : 'divider',
                borderWidth: highlight ? 2 : 1,
              }}
            >
              <CardActionArea
                component={RouterLink}
                to={to}
                sx={{ display: 'flex', alignItems: 'center', px: 3, py: 2.5 }}
              >
                <Icon sx={{ color: 'primary.main', fontSize: 28, mr: 2.5 }} />
                <Typography sx={{ fontWeight: 600, fontSize: '1.1rem', color: 'text.primary' }}>
                  {title}
                </Typography>
                <ChevronRightIcon sx={{ color: 'text.secondary', ml: 'auto' }} />
              </CardActionArea>
            </Card>
          ))}
        </Stack>
      </Container>
    </Box>
  )
}
