import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { Document } from '../api/types'
import DocumentList from '../components/DocumentList'
import { usePageTitle } from '../lib/usePageTitle'

// CharlottePage lists the legacy charlotte-cli documents (read-only). The CLI only
// exposes file names, so rows show the name as the title, sort alphabetically, and
// hide timestamps; archived (archive/*) rows are still marked.
export default function CharlottePage() {
  usePageTitle('Charlotte')
  const [documents, setDocuments] = useState<Document[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .get<Document[]>('/api/charlotte/documents')
      .then(setDocuments)
      .catch((e: unknown) => setError(String(e)))
  }, [])

  return (
    <DocumentList
      heading="Charlotte"
      documents={documents}
      loading={documents === null && error === null}
      error={error}
      defaultSort="alpha"
      basePath="/charlotte"
      showArchived
      hideTime
    />
  )
}
