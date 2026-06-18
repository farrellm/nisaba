import { Navigate, Route, Routes } from 'react-router-dom'
import RequireAuth from './auth/RequireAuth'
import IndexPage from './pages/IndexPage'
import LoginPage from './pages/LoginPage'
import CreateUserPage from './pages/CreateUserPage'
import DocumentsPage from './pages/DocumentsPage'
import DocumentPage from './pages/DocumentPage'
import ArchivePage from './pages/ArchivePage'

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
        path="/documents/:id"
        element={
          <RequireAuth>
            <DocumentPage />
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
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
