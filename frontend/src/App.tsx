import { Navigate, Route, Routes } from 'react-router-dom'
import RequireAuth from './auth/RequireAuth'
import IndexPage from './pages/IndexPage'
import LoginPage from './pages/LoginPage'
import CreateUserPage from './pages/CreateUserPage'
import DocumentsPage from './pages/DocumentsPage'
import LabelsPage from './pages/LabelsPage'
import DocumentPage from './pages/DocumentPage'
import AttributePage from './pages/AttributePage'
import BlockAttributeDiffPage from './pages/BlockAttributeDiffPage'
import ArchivePage from './pages/ArchivePage'
import RedditPostsPage from './pages/RedditPostsPage'
import SettingsPage from './pages/SettingsPage'
import AnansiPage from './pages/AnansiPage'
import AnansiDocumentPage from './pages/AnansiDocumentPage'
import CharlottePage from './pages/CharlottePage'
import CharlotteDocumentPage from './pages/CharlotteDocumentPage'

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/signup" element={<CreateUserPage />} />
      <Route
        path="/"
        element={
          <RequireAuth>
            <IndexPage />
          </RequireAuth>
        }
      />
      <Route
        path="/documents"
        element={
          <RequireAuth>
            <DocumentsPage />
          </RequireAuth>
        }
      />
      <Route
        path="/labels"
        element={
          <RequireAuth>
            <LabelsPage />
          </RequireAuth>
        }
      />
      <Route
        path="/documents/:id"
        element={
          <RequireAuth>
            <DocumentPage />
          </RequireAuth>
        }
      />
      <Route path="/documents/:id/attributes/:key" element={<AttributePage />} />
      <Route
        path="/documents/:id/blocks/:blockId/attributes/:key/diff"
        element={
          <RequireAuth>
            <BlockAttributeDiffPage />
          </RequireAuth>
        }
      />
      <Route
        path="/archive"
        element={
          <RequireAuth>
            <ArchivePage />
          </RequireAuth>
        }
      />
      <Route
        path="/reddit"
        element={
          <RequireAuth>
            <RedditPostsPage />
          </RequireAuth>
        }
      />
      <Route
        path="/settings"
        element={
          <RequireAuth>
            <SettingsPage />
          </RequireAuth>
        }
      />
      <Route
        path="/anansi"
        element={
          <RequireAuth>
            <AnansiPage />
          </RequireAuth>
        }
      />
      <Route
        path="/anansi/:id"
        element={
          <RequireAuth>
            <AnansiDocumentPage />
          </RequireAuth>
        }
      />
      <Route
        path="/charlotte"
        element={
          <RequireAuth>
            <CharlottePage />
          </RequireAuth>
        }
      />
      <Route
        path="/charlotte/:id"
        element={
          <RequireAuth>
            <CharlotteDocumentPage />
          </RequireAuth>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
