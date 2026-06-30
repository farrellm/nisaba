import ReadOnlyDocumentPage from '../components/ReadOnlyDocumentPage'

// AnansiDocumentPage renders a single legacy reflex.db document read-only via the
// shared ReadOnlyDocumentPage.
export default function AnansiDocumentPage() {
  return <ReadOnlyDocumentPage apiBase="/api/anansi/documents" />
}
