import { useState } from 'react'
import { Fab, Tooltip } from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import type { Document } from '../api/types'
import DocumentList from '../components/DocumentList'
import NewDocumentDialog from '../components/NewDocumentDialog'
import { useFetch } from '../lib/useFetch'
import { usePageTitle } from '../lib/usePageTitle'

export default function DocumentsPage() {
  usePageTitle('Documents')
  const { data: documents, error, loading } = useFetch<Document[]>('/api/documents')
  const [dialogOpen, setDialogOpen] = useState(false)

  return (
    <DocumentList
      heading="Documents"
      documents={documents}
      loading={loading}
      error={error}
      active="documents"
      defaultSort="newest"
    >
      <Tooltip title="New document" placement="left">
        <Fab
          color="primary"
          aria-label="New document"
          onClick={() => setDialogOpen(true)}
          sx={{ position: 'fixed', bottom: 32, right: 32, borderRadius: '50%' }}
        >
          <AddIcon />
        </Fab>
      </Tooltip>
      <NewDocumentDialog open={dialogOpen} onClose={() => setDialogOpen(false)} />
    </DocumentList>
  )
}
