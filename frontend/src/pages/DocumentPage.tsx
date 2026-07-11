import { useCallback, useEffect, useMemo, useState } from 'react'
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
import { api } from '../api/client'
import { errorMessage } from '../lib/errors'
import { EMPTY_ATTRIBUTES, type Block, type DocumentDetail, type Mode } from '../api/types'
import Masthead from '../components/Masthead'
import AddBlockDialog from '../components/AddBlockDialog'
import EditLabelsDialog from '../components/EditLabelsDialog'
import RedditSubmitDialog from '../components/RedditSubmitDialog'
import BlockCard from '../components/BlockCard'
import DocumentAttributes from '../components/DocumentAttributes'
import ModelSelector from '../components/ModelSelector'
import StatusLine from '../components/StatusLine'
import { useArmedAction } from '../lib/useArmedAction'
import { usePageTitle } from '../lib/usePageTitle'
import { postLinkSx } from '../lib/styles'

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
  const [submitDialogOpen, setSubmitDialogOpen] = useState(false)

  // Document overflow menu (archive / delete). Delete uses an arm/confirm step,
  // matching BlockCard: the first click arms (and starts a disarm timer), the
  // second confirms.
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const [busy, setBusy] = useState(false)
  const { armed, fire: fireDelete, disarm } = useArmedAction(handleDelete)
  const menuOpen = Boolean(anchorEl)

  function closeMenu() {
    setAnchorEl(null)
    disarm()
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
      setError(errorMessage(err, `Could not ${archive ? 'archive' : 'unarchive'}. Try again.`))
      setBusy(false)
      closeMenu()
    }
  }

  async function handleDelete() {
    setBusy(true)
    setError(null)
    try {
      await api.del(`/api/documents/${id}`)
      navigate('/documents', { replace: true })
    } catch (err) {
      setError(errorMessage(err, 'Could not delete. Try again.'))
      setBusy(false)
      closeMenu()
    }
  }

  useEffect(() => {
    api
      .get<DocumentDetail>(`/api/documents/${id}`)
      .then(setDoc)
      .catch((e: unknown) => setError(errorMessage(e)))
    api
      .get<Mode[]>('/api/modes')
      .then(setModes)
      .catch(() => setModes([]))
  }, [id])

  const createBlock = useCallback(
    async (mode: string): Promise<Block> => {
      const block = await api.post<Block>(`/api/documents/${id}/blocks`, { mode })
      setDoc((d) => (d ? { ...d, blocks: [...(d.blocks ?? []), block] } : d))
      return block
    },
    [id],
  )

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
    api
      .get<DocumentDetail>(`/api/documents/${id}`)
      .then(setDoc)
      .catch(() => {})
  }, [id])

  const modesByName = useMemo(() => new Map(modes.map((m) => [m.name, m])), [modes])

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Masthead />

      <Container maxWidth="md" sx={{ pt: { xs: 7, md: 12 }, pb: 12 }}>
        {error ? (
          <StatusLine tone="error">{error}</StatusLine>
        ) : !doc ? (
          <StatusLine>Loading…</StatusLine>
        ) : (
          <>
            <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1, mb: 4 }}>
              <Box sx={{ flexGrow: 1 }}>
                <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)' }}>
                  {doc.title || 'Untitled'}
                </Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2, mt: 1.5 }}>
                  {doc.url && (
                    <MuiLink
                      href={doc.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      variant="overline"
                      sx={postLinkSx}
                    >
                      Original post ↗
                    </MuiLink>
                  )}
                  {(doc.postUrls ?? []).map((postUrl, i) => (
                    <MuiLink
                      key={postUrl}
                      href={postUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      variant="overline"
                      sx={postLinkSx}
                    >
                      {(doc.postUrls ?? []).length > 1 ? `Posted #${i + 1} ↗` : 'Posted ↗'}
                    </MuiLink>
                  ))}
                </Box>
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
                <MenuItem
                  onClick={() => {
                    setSubmitDialogOpen(true)
                    closeMenu()
                  }}
                >
                  <ListItemText>Post to Reddit…</ListItemText>
                </MenuItem>
                <MenuItem onClick={handleToggleArchive} disabled={busy}>
                  <ListItemText>{doc.isArchived ? 'Unarchive' : 'Archive'}</ListItemText>
                </MenuItem>
                <MenuItem onClick={fireDelete} disabled={busy}>
                  <ListItemText sx={armed ? { color: 'error.main' } : undefined}>
                    {armed ? 'Confirm delete' : 'Delete document'}
                  </ListItemText>
                </MenuItem>
              </Menu>
            </Box>

            {(doc.blocks ?? []).length === 0 ? (
              <StatusLine>No blocks yet. Add one to get started.</StatusLine>
            ) : (
              (doc.blocks ?? []).map((block, i, arr) => (
                <BlockCard
                  key={block.id}
                  block={block}
                  mode={modesByName.get(block.mode)}
                  documentAttributes={doc.attributes ?? EMPTY_ATTRIBUTES}
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
      <Tooltip title="Add block" placement="left">
        <Fab
          color="primary"
          aria-label="Add block"
          onClick={() => setDialogOpen(true)}
          sx={{ position: 'fixed', bottom: 32, right: 32, borderRadius: '50%' }}
        >
          <AddIcon />
        </Fab>
      </Tooltip>
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
      {doc && (
        <RedditSubmitDialog
          open={submitDialogOpen}
          doc={doc}
          onClose={() => setSubmitDialogOpen(false)}
          onPosted={setDoc}
        />
      )}
    </Box>
  )
}
