import { useCallback, useRef, useState } from 'react'
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
  const crepeRef = useRef<Crepe | null>(null)
  const readyRef = useRef<Promise<unknown> | null>(null)
  // Seed once: the dialog is mounted fresh per edit, so initialValue is stable
  // for its lifetime. Reading from a ref keeps the callback ref identity stable
  // (empty deps) so it isn't torn down and recreated on unrelated re-renders.
  const initialValueRef = useRef(initialValue)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Create the editor via a callback ref rather than an effect: MUI's Dialog
  // portals its content in *after* this component's mount effects run, so a
  // rootRef would still be null when an effect fired (leaving a blank editor).
  // A callback ref fires exactly when the node attaches (and again with null on
  // detach), regardless of the portal/transition timing. Destroy runs only
  // after create() resolves so a fast attach→detach can't race an in-flight
  // create (e.g. React StrictMode's double-invoke in dev).
  const editorRootRef = useCallback((node: HTMLDivElement | null) => {
    if (node) {
      const crepe = new Crepe({
        root: node,
        defaultValue: initialValueRef.current,
        features: {
          // TopBar is the persistent formatting toolbar (headings, bold,
          // italic, lists, quote); Toolbar is the selection bubble.
          [Crepe.Feature.TopBar]: true,
          [Crepe.Feature.Toolbar]: true,
        },
        featureConfigs: {
          [Crepe.Feature.Placeholder]: { text: 'Start writing…', mode: 'block' },
          [Crepe.Feature.TopBar]: {
            // Trim the toolbar: drop the "insert" (link, image, table) and
            // "block" (code block, math) groups, plus the checklist from the
            // list group. Features stay enabled so any existing content using
            // those nodes still renders in the editor.
            buildTopBar: (builder) => {
              builder.getGroup('insert').clear()
              builder.getGroup('block').clear()
              const list = builder.getGroup('list')
              list.group.items = list.group.items.filter((item) => item.key !== 'task-list')
            },
          },
        },
      })
      crepeRef.current = crepe
      readyRef.current = crepe.create().catch((err: unknown) => {
        setError(err instanceof Error ? err.message : 'Could not open the editor.')
      })
    } else {
      const crepe = crepeRef.current
      const ready = readyRef.current
      crepeRef.current = null
      readyRef.current = null
      if (crepe) ready?.then(() => crepe.destroy()).catch(() => {})
    }
  }, [])

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
            // Collapse the stray dividers left by the emptied toolbar groups
            // (the top bar renders a divider before every group, even empty).
            '& .top-bar-divider + .top-bar-divider': { display: 'none' },
          }}
        >
          <div ref={editorRootRef} />
        </Box>
      </DialogContent>
    </Dialog>
  )
}
