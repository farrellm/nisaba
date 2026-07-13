import type { Document } from '../api/types'
import DocumentList from '../components/DocumentList'
import { useFetch } from '../lib/useFetch'
import { usePageTitle } from '../lib/usePageTitle'

export default function ArchivePage() {
  usePageTitle('Archive')
  const { data: documents, error, loading } = useFetch<Document[]>('/api/documents?archived=true')

  return (
    <DocumentList
      heading="Archive"
      documents={documents}
      loading={loading}
      error={error}
      active="archive"
      defaultSort="alpha"
    />
  )
}
