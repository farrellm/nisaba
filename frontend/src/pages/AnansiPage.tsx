import type { Document } from '../api/types'
import DocumentList from '../components/DocumentList'
import { useFetch } from '../lib/useFetch'
import { usePageTitle } from '../lib/usePageTitle'

// AnansiPage lists the legacy reflex.db documents (read-only). It reuses the
// shared ledger UI, linking each row into the read-only Anansi viewer and
// marking archived rows since the list mixes archived and active documents.
export default function AnansiPage() {
  usePageTitle('Anansi')
  const { data: documents, error, loading } = useFetch<Document[]>('/api/anansi/documents')

  return (
    <DocumentList
      heading="Anansi"
      documents={documents}
      loading={loading}
      error={error}
      defaultSort="newest"
      basePath="/anansi"
      showArchived
    />
  )
}
