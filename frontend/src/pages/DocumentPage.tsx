import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Box,
  Container,
  Fab,
  IconButton,
  Link as MuiLink,
  ListItemText,
  Menu,
  MenuItem,
  Tooltip,
  Typography,
} from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import MoreVertIcon from '@mui/icons-material/MoreVert'
import { useNavigate, useParams } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import type { Block, DocumentDetail, Mode } from '../api/types'
import Masthead from '../components/Masthead'
import AddBlockDialog from '../components/AddBlockDialog'
import EditLabelsDialog from '../components/EditLabelsDialog'
import BlockCard from '../components/BlockCard'
import DocumentAttributes from '../components/DocumentAttributes'
import ModelSelector from '../components/ModelSelector'
import { usePageTitle } from '../lib/usePageTitle'
import { fonts } from '../theme'

// DocumentPage loads a document via GET /api/documents/:id and renders its
// blocks. Users add blocks (choosing a mode), edit each block's key/values, and
// run them.
export default function DocumentPage() {
  const { id } = useParams()
  const navigate = useNavigate()

  const [doc, setDoc] = useState<DocumentDetail | null>(null)
  usePageTitle(doc ? doc.title || 'Untitled' : null)
  const [modes, setModes] = useState<Mode[]>([])
  const [error, setError] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [labelsDialogOpen, setLabelsDialogOpen] = useState(false)

  // Document overflow menu (archive / delete). Delete uses an arm/confirm step,
  // matching BlockCard: the first click arms (and starts a disarm timer), the
  // second confirms.
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const [armed, setArmed] = useState(false)
  const [busy, setBusy] = useState(false)
  const armedTimer = useRef<ReturnType<typeof setTimeout>>()
  const menuOpen = Boolean(anchorEl)

  function closeMenu() {
    setAnchorEl(null)
    setArmed(false)
    clearTimeout(armedTimer.current)
  }

  async function handleToggleArchive() {
    if (!doc) return
    const archive = !doc.isArchived
    setBusy(true)
    setError(null)
    try {
      const updated = await api.put<DocumentDetail>(`/api/documents/${id}`, {
        isArchived: archive,
      })
      // Archiving removes the doc from the active list, so leave for it.
      // Unarchiving keeps you on the page; just reflect the new state.
      if (archive) {
        navigate('/documents')
      } else {
        setDoc(updated)
        setBusy(false)
        closeMenu()
      }
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message
          : `Could not ${archive ? 'archive' : 'unarchive'}. Try again.`,
      )
      setBusy(false)
      closeMenu()
    }
  }

  function handleDeleteClick() {
    if (!armed) {
      setArmed(true)
      armedTimer.current = setTimeout(() => setArmed(false), 4000)
      return
    }
    clearTimeout(armedTimer.current)
    handleDelete()
  }

  async function handleDelete() {
    setBusy(true)
    setError(null)
    try {
      await api.del(`/api/documents/${id}`)
      navigate('/documents', { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not delete. Try again.')
      setBusy(false)
      closeMenu()
    }
  }

  useEffect(() => {
    api
      .get<DocumentDetail>(`/api/documents/${id}`)
      .then(setDoc)
      .catch((e: unknown) => setError(String(e)))
    api.get<Mode[]>('/api/modes').then(setModes).catch(() => setModes([]))
  }, [id])

  const createBlock = useCallback(async (mode: string): Promise<Block> => {
    const block = await api.post<Block>(`/api/documents/${id}/blocks`, { mode })
    setDoc((d) => (d ? { ...d, blocks: [...(d.blocks ?? []), block] } : d))
    return block
  }, [id])

  const replaceBlock = useCallback((updated: Block) => {
    setDoc((d) =>
      d ? { ...d, blocks: (d.blocks ?? []).map((b) => (b.id === updated.id ? updated : b)) } : d,
    )
  }, [])

  const removeBlock = useCallback((blockId: number) => {
    setDoc((d) => (d ? { ...d, blocks: (d.blocks ?? []).filter((b) => b.id !== blockId) } : d))
  }, [])

  // Running a block mutates the document's shared attributes, so reload it.
  const reloadDocument = useCallback(() => {
    api.get<DocumentDetail>(`/api/documents/${id}`).then(setDoc).catch(() => {})
  }, [id])

  const modesByName = useMemo(() => new Map(modes.map((m) => [m.name, m])), [modes])

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead />

      <Container maxWidth="md" sx={{ pt: { xs: 7, md: 12 }, pb: 12 }}>
        {error ? (
          <Typography sx={{ fontFamily: fonts.mono, color: 'error.main' }}>{error}</Typography>
        ) : !doc ? (
          <Typography sx={{ fontFamily: fonts.mono, color: 'text.secondary' }}>Loading…</Typography>
        ) : (
          <>
            <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1, mb: 4 }}>
              <Box sx={{ flexGrow: 1 }}>
                <Typography
                  variant="h1"
                  sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)' }}
                >
                  {doc.title || 'Untitled'}
                </Typography>
                {doc.url && (
                  <MuiLink
                    href={doc.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    variant="overline"
                    sx={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: 0.5,
                      mt: 1.5,
                      color: 'primary.main',
                      textDecoration: 'none',
                      '&:hover': { textDecoration: 'underline' },
                    }}
                  >
                    Original post ↗
                  </MuiLink>
                )}
              </Box>
              <Tooltip title="Document menu">
                <IconButton
                  onClick={(e) => setAnchorEl(e.currentTarget)}
                  aria-label="Document menu"
                  aria-haspopup="true"
                  aria-expanded={menuOpen ? 'true' : undefined}
                  sx={{ color: 'text.secondary', mt: 1 }}
                >
                  <MoreVertIcon />
                </IconButton>
              </Tooltip>
              <Menu
                anchorEl={anchorEl}
                open={menuOpen}
                onClose={closeMenu}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                transformOrigin={{ vertical: 'top', horizontal: 'right' }}
              >
                <MenuItem
                  onClick={() => {
                    setLabelsDialogOpen(true)
                    closeMenu()
                  }}
                >
                  <ListItemText>Edit labels…</ListItemText>
                </MenuItem>
                <MenuItem onClick={handleToggleArchive} disabled={busy}>
                  <ListItemText>{doc.isArchived ? 'Unarchive' : 'Archive'}</ListItemText>
                </MenuItem>
                <MenuItem onClick={handleDeleteClick} disabled={busy}>
                  <ListItemText sx={armed ? { color: 'error.main' } : undefined}>
                    {armed ? 'Confirm delete' : 'Delete document'}
                  </ListItemText>
                </MenuItem>
              </Menu>
            </Box>

            {(doc.blocks ?? []).length === 0 ? (
              <Typography sx={{ fontFamily: fonts.mono, color: 'text.secondary' }}>
                No blocks yet. Add one to get started.
              </Typography>
            ) : (
              (doc.blocks ?? []).map((block, i, arr) => (
                <BlockCard
                  key={block.id}
                  block={block}
                  mode={modesByName.get(block.mode)}
                  documentAttributes={doc.attributes ?? {}}
                  onBlockUpdated={replaceBlock}
                  onBlockDeleted={removeBlock}
                  onAfterRun={reloadDocument}
                  defaultOpen={i === arr.length - 1}
                />
              ))
            )}

            <DocumentAttributes doc={doc} onChange={setDoc} />
          </>
        )}
      </Container>

      {doc && <ModelSelector doc={doc} onChange={setDoc} />}
      <Fab
        color="primary"
        aria-label="Add block"
        onClick={() => setDialogOpen(true)}
        sx={{ position: 'fixed', bottom: 32, right: 32, borderRadius: '50%' }}
      >
        <AddIcon />
      </Fab>
      <AddBlockDialog
        open={dialogOpen}
        modes={modes}
        onClose={() => setDialogOpen(false)}
        onCreate={createBlock}
      />
      {doc && (
        <EditLabelsDialog
          open={labelsDialogOpen}
          doc={doc}
          onClose={() => setLabelsDialogOpen(false)}
          onChange={setDoc}
        />
      )}
    </Box>
  )
}
