import type { Document } from '../api/types'
import DocumentList from '../components/DocumentList'
import { useFetch } from '../lib/useFetch'
import { usePageTitle } from '../lib/usePageTitle'

// CharlottePage lists the legacy charlotte-cli documents (read-only). The CLI only
// exposes file names, so rows show the name as the title, sort alphabetically, and
// hide timestamps; archived (archive/*) rows are still marked.
export default function CharlottePage() {
  usePageTitle('Charlotte')
  const { data: documents, error, loading } = useFetch<Document[]>('/api/charlotte/documents')

  return (
    <DocumentList
      heading="Charlotte"
      documents={documents}
      loading={loading}
      error={error}
      defaultSort="alpha"
      basePath="/charlotte"
      showArchived
      hideTime
    />
  )
}
