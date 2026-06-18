// Document mirrors the backend model.Document summary shape returned by
// /api/documents. Only the fields the UI currently uses are declared.
export interface Document {
  id: number
  title: string
  url: string | null
  updatedAt: string
  isArchived: boolean
}
