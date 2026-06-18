import { useEffect, useState } from 'react'
import { Box, Button, Container, Fab, Link as MuiLink, Stack, Typography } from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Block, DocumentDetail, Mode } from '../api/types'
import { useAuth } from '../auth/AuthContext'
import AddBlockDialog from '../components/AddBlockDialog'
import BlockCard from '../components/BlockCard'
import { fonts } from '../theme'

// DocumentPage loads a document via GET /api/documents/:id and renders its
// blocks. Users add blocks (choosing a mode), edit each block's key/values, and
// run them.
export default function DocumentPage() {
  const { id } = useParams()
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const [doc, setDoc] = useState<DocumentDetail | null>(null)
  const [modes, setModes] = useState<Mode[]>([])
  const [error, setError] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)

  useEffect(() => {
    api
      .get<DocumentDetail>(`/api/documents/${id}`)
      .then(setDoc)
      .catch((e: unknown) => setError(String(e)))
    api.get<Mode[]>('/api/modes').then(setModes).catch(() => setModes([]))
  }, [id])

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

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

  // Running a block mutates the document's shared attributes, so reload it.
  function reloadDocument() {
    api.get<DocumentDetail>(`/api/documents/${id}`).then(setDoc).catch(() => {})
  }

  const modesByName = new Map(modes.map((m) => [m.name, m]))

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Box
        component="header"
        sx={{
          borderBottom: '1px solid',
          borderColor: 'divider',
          px: { xs: 3, md: 5 },
          py: 2,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <Stack direction="row" spacing={3} alignItems="baseline">
          <MuiLink component={RouterLink} to="/" underline="none" color="inherit">
            <Typography
              sx={{ fontFamily: fonts.display, fontWeight: 600, fontSize: '1.5rem', letterSpacing: '-0.02em' }}
            >
              Nisaba
            </Typography>
          </MuiLink>
          <MuiLink
            component={RouterLink}
            to="/documents"
            underline="hover"
            sx={{ fontFamily: fonts.mono, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.08em' }}
          >
            Documents
          </MuiLink>
        </Stack>
        <Stack direction="row" spacing={2} alignItems="center">
          <Typography variant="overline" sx={{ color: 'text.secondary' }}>
            {user?.username}
          </Typography>
          <Button variant="text" size="small" onClick={handleLogout} sx={{ color: 'text.primary' }}>
            Log out
          </Button>
        </Stack>
      </Box>

      <Container maxWidth="md" sx={{ pt: { xs: 7, md: 12 }, pb: 12 }}>
        {error ? (
          <Typography sx={{ fontFamily: fonts.mono, color: 'error.main' }}>{error}</Typography>
        ) : !doc ? (
          <Typography sx={{ fontFamily: fonts.mono, color: 'text.secondary' }}>Loading…</Typography>
        ) : (
          <>
            <Typography variant="overline" sx={{ color: 'primary.main', display: 'block', mb: 2 }}>
              {doc.selectedModel || 'No model selected'}
            </Typography>
            <Typography variant="h1" sx={{ fontSize: 'clamp(2.25rem, 6vw, 3.5rem)', mb: 4 }}>
              {doc.title || 'Untitled'}
            </Typography>

            {(doc.blocks ?? []).length === 0 ? (
              <Typography sx={{ fontFamily: fonts.mono, color: 'text.secondary' }}>
                No blocks yet. Add one to get started.
              </Typography>
            ) : (
              (doc.blocks ?? []).map((block) => (
                <BlockCard
                  key={block.id}
                  block={block}
                  mode={modesByName.get(block.mode)}
                  onBlockUpdated={replaceBlock}
                  onAfterRun={reloadDocument}
                />
              ))
            )}
          </>
        )}
      </Container>

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
