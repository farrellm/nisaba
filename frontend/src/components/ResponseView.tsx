import { useMemo, type ReactNode } from 'react'
import { Box, IconButton, TextField, Tooltip, Typography } from '@mui/material'
import DataObjectIcon from '@mui/icons-material/DataObject'
import Markdown from './Markdown'
import type { Response } from '../api/types'
import { useModels, modelLabel } from '../api/useModels'
import { parseResponseSegments } from '../lib/responseSegments'
import { fonts } from '../theme'

interface ResponseViewProps {
  response: Response
  // Whether this response's <details> is open. Controlled by the caller so it
  // can collapse older responses when a new run starts.
  open: boolean
  // Fires when the user toggles the <details>, with its new open state.
  onToggle: (open: boolean) => void
  // Structured view renders top-level tags as collapsible quoted sections;
  // raw view shows the verbatim mono text.
  structured: boolean
  onToggleStructured: () => void
  // Extra summary-row icon buttons after the structured/raw toggle (edit,
  // reparse). Read-only callers omit this.
  actions?: ReactNode
  // When set, the response is being edited in place: the summary shows
  // editing.actions (save/cancel) instead of the toggle + actions, and the
  // body is an editable mono TextField.
  editing?: {
    value: string
    onChange: (value: string) => void
    actions: ReactNode
  }
}

// ResponseView renders one model response as a collapsible <details> with the
// model name as its summary, a structured/raw toggle, and optional edit-in-
// place support. Shared by the live BlockCard and the read-only legacy
// document views.
export default function ResponseView({
  response,
  open,
  onToggle,
  structured,
  onToggleStructured,
  actions,
  editing,
}: ResponseViewProps) {
  // Re-parse only when the text changes, not on every toggle/render.
  const segments = useMemo(() => parseResponseSegments(response.value), [response.value])
  const models = useModels()

  return (
    <Box
      component="details"
      open={open}
      onToggle={(e) => onToggle((e.currentTarget as HTMLDetailsElement).open)}
    >
      <Box
        component="summary"
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          cursor: 'pointer',
          listStyle: 'none',
          '&::-webkit-details-marker': { display: 'none' },
          mb: 1,
        }}
      >
        <Typography
          variant="overline"
          sx={{ fontFamily: fonts.mono, color: 'text.secondary', fontSize: '0.7rem' }}
        >
          {modelLabel(models, response.model)}
        </Typography>
        <Box sx={{ flexGrow: 1 }} />
        {editing ? (
          editing.actions
        ) : (
          <>
            <Tooltip title={structured ? 'Raw view' : 'Structured view'}>
              <span>
                <IconButton
                  size="small"
                  aria-label={structured ? 'Show raw response' : 'Show structured response'}
                  onClick={(e) => {
                    e.preventDefault()
                    onToggleStructured()
                  }}
                  sx={{
                    color: structured ? 'primary.main' : 'text.disabled',
                    '&:hover': { color: 'primary.main' },
                  }}
                >
                  <DataObjectIcon fontSize="small" />
                </IconButton>
              </span>
            </Tooltip>
            {actions}
          </>
        )}
      </Box>
      {editing ? (
        <TextField
          fullWidth
          multiline
          minRows={6}
          value={editing.value}
          onChange={(e) => editing.onChange(e.target.value)}
          autoFocus
          inputProps={{ 'aria-label': 'Edit response value' }}
          InputProps={{
            sx: { fontFamily: fonts.mono, fontSize: '0.85rem' },
          }}
        />
      ) : structured ? (
        <Box sx={{ bgcolor: 'action.hover', borderRadius: 2, p: 2 }}>
          {segments.map((seg, segIdx) =>
            seg.kind === 'text' ? (
              <Markdown key={segIdx}>{seg.text}</Markdown>
            ) : (
              <Box
                key={segIdx}
                component="details"
                open
                sx={{ my: 1, '&:first-of-type': { mt: 0 }, '&:last-child': { mb: 0 } }}
              >
                <Box
                  component="summary"
                  sx={{
                    cursor: 'pointer',
                    fontFamily: fonts.mono,
                    fontSize: '0.8rem',
                    color: 'text.secondary',
                  }}
                >
                  {seg.name}
                </Box>
                <Box
                  component="blockquote"
                  sx={{
                    my: 1,
                    ml: 0,
                    pl: 2.5,
                    borderLeft: '3px solid',
                    borderColor: 'divider',
                    color: 'text.secondary',
                  }}
                >
                  {/* Escape '<' so nested tags render as literal text:
                      react-markdown drops raw HTML. */}
                  <Markdown>{seg.inner.split('<').join('\\<')}</Markdown>
                </Box>
              </Box>
            ),
          )}
        </Box>
      ) : (
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
          {response.value}
        </Typography>
      )}
    </Box>
  )
}
