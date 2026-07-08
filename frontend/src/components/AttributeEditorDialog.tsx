import { useEffect, useRef, useState } from 'react'
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogContent,
  Stack,
  Typography,
} from '@mui/material'
import { Crepe } from '@milkdown/crepe'
import '@milkdown/crepe/theme/common/style.css'
import '@milkdown/crepe/theme/frame.css'
import { ApiError } from '../api/client'
import { fonts } from '../theme'

interface AttributeEditorDialogProps {
  open: boolean
  attributeKey: string
  initialValue: string
  onClose: () => void
  onSave: (markdown: string) => Promise<void>
}

// AttributeEditorDialog is a full-screen WYSIWYG markdown editor (Milkdown Crepe)
// for a single document attribute value. It opens as an overlay at the current
// URL rather than navigating. Saving reads the editor's markdown and hands it to
// onSave, which persists it; the parent owns the PUT and local-state sync.
export default function AttributeEditorDialog({
  open,
  attributeKey,
  initialValue,
  onClose,
  onSave,
}: AttributeEditorDialogProps) {
  const rootRef = useRef<HTMLDivElement>(null)
  const crepeRef = useRef<Crepe | null>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Seed the editor once per open. Deliberately keyed on `open` only, not
  // `initialValue`, so keystrokes never re-seed and blow away the user's edits.
  // Destroy runs only after create() resolves so React 18 StrictMode's
  // mount→unmount→remount can't race a destroy ahead of an in-flight create.
  useEffect(() => {
    if (!open) return
    const root = rootRef.current
    if (!root) return
    const crepe = new Crepe({
      root,
      defaultValue: initialValue,
      features: { [Crepe.Feature.Toolbar]: true },
      featureConfigs: {
        [Crepe.Feature.Placeholder]: { text: 'Start writing…', mode: 'block' },
      },
    })
    crepeRef.current = crepe
    const ready = crepe.create()
    return () => {
      crepeRef.current = null
      ready.then(() => crepe.destroy()).catch(() => {})
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  async function handleSave() {
    setError(null)
    setSaving(true)
    try {
      await onSave(crepeRef.current?.getMarkdown() ?? '')
      onClose()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save. Try again.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onClose={saving ? undefined : onClose} fullScreen>
      <Stack
        direction="row"
        spacing={2}
        sx={{
          alignItems: 'center',
          px: 3,
          py: 1.5,
          borderBottom: '1px dotted',
          borderColor: 'divider',
        }}
      >
        <Typography
          variant="overline"
          sx={{ fontFamily: fonts.mono, color: 'primary.main', whiteSpace: 'nowrap' }}
        >
          Edit · {attributeKey}
        </Typography>
        <Box sx={{ flex: 1 }} />
        <Button onClick={onClose} disabled={saving} sx={{ color: 'text.secondary' }}>
          Cancel
        </Button>
        <Button variant="contained" onClick={handleSave} disabled={saving}>
          {saving ? (
            <>
              <CircularProgress size={16} color="inherit" sx={{ mr: 1 }} />
              Saving…
            </>
          ) : (
            'Save'
          )}
        </Button>
      </Stack>

      <DialogContent sx={{ p: 0, bgcolor: 'background.default' }}>
        {error && (
          <Alert severity="error" sx={{ m: 3, mb: 0 }}>
            {error}
          </Alert>
        )}
        <Box
          sx={{
            maxWidth: 760,
            mx: 'auto',
            px: { xs: 2, sm: 4 },
            py: 3,
            // Tune Crepe's frame theme to the app's ink-blue editorial palette.
            '& .milkdown': {
              '--crepe-color-primary': '#2540E0',
              '--crepe-color-background': '#FBFAF7',
              '--crepe-color-surface': '#F3F1EA',
              '--crepe-color-surface-low': '#ECE9E0',
              '--crepe-color-hover': '#ECE9E0',
              '--crepe-color-selected': '#DFE3FB',
              '--crepe-color-inline-area': '#DFE3FB',
              '--crepe-font-default': fonts.body,
              '--crepe-font-title': fonts.display,
              '--crepe-font-code': fonts.mono,
              minHeight: '60vh',
              padding: 0,
            },
            '& .milkdown .ProseMirror': { padding: 0 },
          }}
        >
          <div ref={rootRef} />
        </Box>
      </DialogContent>
    </Dialog>
  )
}
