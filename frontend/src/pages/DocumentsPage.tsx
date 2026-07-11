import { useEffect, useState } from 'react'
import { Fab, Tooltip } from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import { api } from '../api/client'
import { errorMessage } from '../lib/errors'
import type { Document } from '../api/types'
import DocumentList from '../components/DocumentList'
import NewDocumentDialog from '../components/NewDocumentDialog'
import { usePageTitle } from '../lib/usePageTitle'

export default function DocumentsPage() {
  usePageTitle('Documents')
  const [documents, setDocuments] = useState<Document[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)

  useEffect(() => {
    api
      .get<Document[]>('/api/documents')
      .then(setDocuments)
      .catch((e: unknown) => setError(errorMessage(e)))
  }, [])

  return (
    <DocumentList
      heading="Documents"
      documents={documents}
      loading={documents === null && error === null}
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
