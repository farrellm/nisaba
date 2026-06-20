import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { Document } from '../api/types'
import DocumentList from '../components/DocumentList'

export default function ArchivePage() {
  const [documents, setDocuments] = useState<Document[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .get<Document[]>('/api/documents?archived=true')
      .then(setDocuments)
      .catch((e: unknown) => setError(String(e)))
  }, [])

  return (
    <DocumentList
      heading="Archive"
      documents={documents}
      loading={documents === null && error === null}
      error={error}
      active="archive"
    />
  )
}
