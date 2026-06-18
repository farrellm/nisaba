import { createTheme } from '@mui/material/styles'

// ── Modern-editorial token system ───────────────────────────────────────────
// A document tool named for the Sumerian goddess of scribes. The look borrows
// from print: a publication masthead in a high-contrast serif, mono utility
// labels that read like a ledger, hairline rules, and a single confident accent
// (ink-blue, deliberately not the terracotta/cream default).
const ink = '#171410' // warm near-black
const paper = '#FBFAF7' // cool-warm white (not cream)
const muted = '#6B655B' // secondary text
const hairline = '#E5E1D8' // dividers / borders
const accent = '#2540E0' // ink-blue

const display = "'Fraunces', Georgia, serif"
const body = "'Inter', system-ui, sans-serif"
const mono = "'IBM Plex Mono', ui-monospace, monospace"

const theme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: accent },
    background: { default: paper, paper: paper },
    text: { primary: ink, secondary: muted },
    divider: hairline,
  },
  shape: { borderRadius: 2 },
  typography: {
    fontFamily: body,
    h1: { fontFamily: display, fontWeight: 600, letterSpacing: '-0.02em', lineHeight: 1.02 },
    h2: { fontFamily: display, fontWeight: 600, letterSpacing: '-0.02em', lineHeight: 1.05 },
    h3: { fontFamily: display, fontWeight: 500, letterSpacing: '-0.01em' },
    h4: { fontFamily: display, fontWeight: 500 },
    button: { textTransform: 'none', fontWeight: 600, letterSpacing: '0.01em' },
    // Mono utility scale for eyebrows, field labels, and ledger data.
    overline: {
      fontFamily: mono,
      fontWeight: 500,
      fontSize: '0.7rem',
      letterSpacing: '0.18em',
      textTransform: 'uppercase',
    },
  },
  components: {
    MuiButton: {
      defaultProps: { disableElevation: true },
      styleOverrides: {
        root: { paddingTop: 10, paddingBottom: 10 },
      },
    },
    MuiTextField: {
      defaultProps: { variant: 'outlined', fullWidth: true },
    },
  },
})

export const fonts = { display, body, mono }
export default theme
