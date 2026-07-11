import { useEffect, useState } from 'react'
import { api } from '../api/client'
import { errorMessage } from '../lib/errors'
import type { Document } from '../api/types'
import DocumentList from '../components/DocumentList'
import { usePageTitle } from '../lib/usePageTitle'

export default function ArchivePage() {
  usePageTitle('Archive')
  const [documents, setDocuments] = useState<Document[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .get<Document[]>('/api/documents?archived=true')
      .then(setDocuments)
      .catch((e: unknown) => setError(errorMessage(e)))
  }, [])

  return (
    <DocumentList
      heading="Archive"
      documents={documents}
      loading={documents === null && error === null}
      error={error}
      active="archive"
      defaultSort="alpha"
    />
  )
}
