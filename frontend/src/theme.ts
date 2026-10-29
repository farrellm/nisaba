import { createTheme } from '@mui/material/styles'

// ── Modern-editorial token system ───────────────────────────────────────────
// A document tool named for the Sumerian goddess of scribes. The look borrows
// from print: a publication masthead in a high-contrast serif, mono utility
// labels that read like a ledger, hairline rules, and a single confident accent
// (ink-blue, deliberately not the terracotta/cream default).
//
// Dark mode is the page turned over: the ink becomes the ground (an indigo
// ink-wash, not a neutral black) and the paper becomes the type, with the
// ink-blue accent lifted so it still reads as the same pen.
const light = {
  ink: '#171410', // warm near-black
  paper: '#FBFAF7', // cool-warm white (not cream)
  muted: '#6B655B', // secondary text
  hairline: '#E5E1D8', // dividers / borders
  accent: '#2540E0', // ink-blue
  struck: '#B23A2E', // struck-through diff text (a printer's red)
  // Crepe editor tints, derived from paper/accent.
  surface: '#F3F1EA',
  surfaceLow: '#ECE9E0',
  selected: '#DFE3FB',
}
const dark = {
  ink: '#ECE7DC', // the paper, now as type
  paper: '#161A2B', // indigo ink-wash ground
  muted: '#A49D90',
  hairline: '#2D3349',
  accent: '#9AABFF', // ink-blue, lifted for the dark ground
  struck: '#F08A7E',
  surface: '#1D2236',
  surfaceLow: '#232941',
  selected: '#2E3866',
}
type Tokens = typeof light

const display = "'Fraunces', Georgia, serif"
const body = "'Inter', system-ui, sans-serif"
const mono = "'IBM Plex Mono', ui-monospace, monospace"

const schemePalette = (t: Tokens) => ({
  primary: { main: t.accent },
  background: { default: t.paper, paper: t.paper },
  text: { primary: t.ink, secondary: t.muted },
  divider: t.hairline,
})

// The tokens MUI's palette has no slot for, as CSS variables that flip with
// the scheme (see `palette` below).
const extraVars = (t: Tokens) => ({
  '--nisaba-struck': t.struck,
  '--nisaba-surface': t.surface,
  '--nisaba-surface-low': t.surfaceLow,
  '--nisaba-selected': t.selected,
})

// Light, dark, or following the device (the default). MUI stores the choice in
// localStorage under `mui-mode` and marks <html> with `data-color-scheme`; the
// inline script in index.html applies it before first paint, so keep the two
// in step.
const theme = createTheme({
  cssVariables: { colorSchemeSelector: 'data-color-scheme' },
  colorSchemes: {
    light: { palette: schemePalette(light) },
    dark: {
      palette: {
        ...schemePalette(dark),
        primary: { main: dark.accent, contrastText: dark.paper },
      },
    },
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
    MuiCssBaseline: {
      styleOverrides: {
        ':root': extraVars(light),
        ':root[data-color-scheme="dark"]': extraVars(dark),
      },
    },
    // Dark-mode MUI lightens raised surfaces with a white gradient; keep
    // dialogs and menus flat on the ink ground, as they are on paper.
    MuiPaper: {
      styleOverrides: { root: { backgroundImage: 'none' } },
    },
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
// Scheme-aware color references, for the rare spot that needs a literal (CSS
// variables handed to a non-MUI widget, diff strike color) — everything else
// should go through the MUI theme.
export const palette = {
  paper: 'var(--mui-palette-background-default)',
  accent: 'var(--mui-palette-primary-main)',
  struck: 'var(--nisaba-struck)',
  surface: 'var(--nisaba-surface)',
  surfaceLow: 'var(--nisaba-surface-low)',
  selected: 'var(--nisaba-selected)',
} as const
export default theme
