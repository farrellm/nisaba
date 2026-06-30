import ReadOnlyDocumentPage from '../components/ReadOnlyDocumentPage'

// CharlotteDocumentPage renders a single legacy charlotte-cli document read-only via
// the shared ReadOnlyDocumentPage.
export default function CharlotteDocumentPage() {
  return <ReadOnlyDocumentPage apiBase="/api/charlotte/documents" />
}
