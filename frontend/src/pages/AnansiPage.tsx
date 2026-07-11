import { useEffect, useState } from 'react'
import { api } from '../api/client'
import { errorMessage } from '../lib/errors'
import type { Document } from '../api/types'
import DocumentList from '../components/DocumentList'
import { usePageTitle } from '../lib/usePageTitle'

// AnansiPage lists the legacy reflex.db documents (read-only). It reuses the
// shared ledger UI, linking each row into the read-only Anansi viewer and
// marking archived rows since the list mixes archived and active documents.
export default function AnansiPage() {
  usePageTitle('Anansi')
  const [documents, setDocuments] = useState<Document[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .get<Document[]>('/api/anansi/documents')
      .then(setDocuments)
      .catch((e: unknown) => setError(errorMessage(e)))
  }, [])

  return (
    <DocumentList
      heading="Anansi"
      documents={documents}
      loading={documents === null && error === null}
      error={error}
      defaultSort="newest"
      basePath="/anansi"
      showArchived
    />
  )
}
