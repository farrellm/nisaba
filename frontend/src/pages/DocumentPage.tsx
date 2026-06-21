import { useEffect, useRef, useState } from 'react'
import {
  Box,
  Container,
  Fab,
  IconButton,
  ListItemText,
  Menu,
  MenuItem,
  Typography,
} from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import MoreVertIcon from '@mui/icons-material/MoreVert'
import { useNavigate, useParams } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import type { Block, DocumentDetail, Mode } from '../api/types'
import Masthead from '../components/Masthead'
import AddBlockDialog from '../components/AddBlockDialog'
import BlockCard from '../components/BlockCard'
import DocumentAttributes from '../components/DocumentAttributes'
import ModelSelector from '../components/ModelSelector'
import { fonts } from '../theme'

// DocumentPage loads a document via GET /api/documents/:id and renders its
// blocks. Users add blocks (choosing a mode), edit each block's key/values, and
// run them.
export default function DocumentPage() {
  const { id } = useParams()
  const navigate = useNavigate()

  const [doc, setDoc] = useState<DocumentDetail | null>(null)
  const [modes, setModes] = useState<Mode[]>([])
  const [error, setError] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)

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

  async function handleArchive() {
    setBusy(true)
    setError(null)
    try {
      await api.put(`/api/documents/${id}`, { isArchived: true })
      navigate('/documents')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not archive. Try again.')
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

  async function createBlock(mode: string): Promise<Block> {
    const block = await api.post<Block>(`/api/documents/${id}/blocks`, { mode })
    setDoc((d) => (d ? { ...d, blocks: [...(d.blocks ?? []), block] } : d))
    return block
  }

  function replaceBlock(updated: Block) {
    setDoc((d) =>
      d ? { ...d, blocks: (d.blocks ?? []).map((b) => (b.id === updated.id ? updated : b)) } : d,
    )
  }

  function removeBlock(id: number) {
    setDoc((d) => (d ? { ...d, blocks: (d.blocks ?? []).filter((b) => b.id !== id) } : d))
  }

  // Running a block mutates the document's shared attributes, so reload it.
  function reloadDocument() {
    api.get<DocumentDetail>(`/api/documents/${id}`).then(setDoc).catch(() => {})
  }

  const modesByName = new Map(modes.map((m) => [m.name, m]))

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
              <Typography
                variant="h1"
                sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)', flexGrow: 1 }}
              >
                {doc.title || 'Untitled'}
              </Typography>
              <IconButton
                onClick={(e) => setAnchorEl(e.currentTarget)}
                aria-label="Document menu"
                aria-haspopup="true"
                aria-expanded={menuOpen ? 'true' : undefined}
                sx={{ color: 'text.secondary', mt: 1 }}
              >
                <MoreVertIcon />
              </IconButton>
              <Menu
                anchorEl={anchorEl}
                open={menuOpen}
                onClose={closeMenu}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                transformOrigin={{ vertical: 'top', horizontal: 'right' }}
              >
                <MenuItem onClick={handleArchive} disabled={busy}>
                  <ListItemText>Archive</ListItemText>
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
    </Box>
  )
}
