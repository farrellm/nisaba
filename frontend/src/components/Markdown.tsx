import { memo } from 'react'
import { Box } from '@mui/material'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { fonts } from '../theme'

const remarkPlugins = [remarkGfm]

// Markdown renders free-form markdown (typically LLM output) with the app's
// editorial typography. Styling lives in one place via descendant selectors on a
// wrapping Box rather than per-element component overrides, so the renderer stays
// reusable.
const Markdown = memo(function Markdown({ children }: { children: string }) {
  return (
    <Box
      sx={{
        fontFamily: fonts.body,
        color: 'text.primary',
        fontSize: '1.05rem',
        lineHeight: 1.7,
        wordBreak: 'break-word',
        '& > :first-of-type': { mt: 0 },
        '& > :last-child': { mb: 0 },
        '& h1, & h2, & h3, & h4, & h5, & h6': {
          fontFamily: fonts.display,
          fontWeight: 600,
          letterSpacing: '-0.02em',
          lineHeight: 1.15,
          mt: 4,
          mb: 1.5,
        },
        '& h1': { fontSize: '2.2rem' },
        '& h2': { fontSize: '1.7rem' },
        '& h3': { fontSize: '1.35rem' },
        '& h4, & h5, & h6': { fontSize: '1.1rem' },
        '& p': { my: 2 },
        '& a': {
          color: 'primary.main',
          textDecoration: 'underline',
          textUnderlineOffset: '2px',
        },
        '& ul, & ol': { my: 2, pl: 3.5 },
        '& li': { mb: 0.5 },
        '& li > ul, & li > ol': { my: 0.5 },
        '& blockquote': {
          my: 2,
          ml: 0,
          pl: 2.5,
          borderLeft: '3px solid',
          borderColor: 'divider',
          color: 'text.secondary',
          fontStyle: 'italic',
        },
        '& code': {
          fontFamily: fonts.mono,
          fontSize: '0.88em',
          bgcolor: 'rgba(0,0,0,0.05)',
          px: 0.6,
          py: 0.2,
          borderRadius: 1,
        },
        '& pre': {
          my: 2,
          p: 2,
          overflowX: 'auto',
          bgcolor: 'rgba(0,0,0,0.05)',
          borderRadius: 1,
          border: '1px solid',
          borderColor: 'divider',
        },
        '& pre code': {
          bgcolor: 'transparent',
          p: 0,
          fontSize: '0.85rem',
          lineHeight: 1.6,
        },
        '& hr': {
          my: 4,
          border: 'none',
          borderTop: '1px solid',
          borderColor: 'divider',
        },
        '& img': { maxWidth: '100%', height: 'auto' },
        '& table': {
          my: 2,
          borderCollapse: 'collapse',
          width: '100%',
          fontSize: '0.95rem',
        },
        '& th, & td': {
          border: '1px solid',
          borderColor: 'divider',
          px: 1.5,
          py: 0.75,
          textAlign: 'left',
        },
        '& th': { fontFamily: fonts.mono, fontWeight: 600, fontSize: '0.8rem' },
      }}
    >
      <ReactMarkdown remarkPlugins={remarkPlugins}>{children}</ReactMarkdown>
    </Box>
  )
})

export default Markdown
