import { fonts } from '../theme'

// Shared sx fragments for the recurring editorial motifs.

// The "Original post" / "Posted" permalink chips under a document title.
export const postLinkSx = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 0.5,
  color: 'primary.main',
  textDecoration: 'none',
  '&:hover': { textDecoration: 'underline' },
} as const

// Masthead-style nav links (mono, uppercase, tracked out).
export const navLinkSx = {
  fontFamily: fonts.mono,
  fontSize: '0.75rem',
  textTransform: 'uppercase',
  letterSpacing: '0.08em',
} as const

// Shared <summary> styling: a flex row whose default disclosure marker is
// hidden so a section header + dotted leader read as one ledger line.
export const summarySx = {
  display: 'flex',
  alignItems: 'baseline',
  gap: 2,
  mb: 2,
  cursor: 'pointer',
  listStyle: 'none',
  '&::-webkit-details-marker': { display: 'none' },
} as const

// The dotted leader that fills the space between a header and its trailing
// controls, nudged down to sit on the text baseline.
export const leaderSx = {
  flex: 1,
  borderBottom: '1px dotted',
  borderColor: 'divider',
  transform: 'translateY(-3px)',
} as const
