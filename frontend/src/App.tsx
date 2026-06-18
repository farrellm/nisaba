import { Navigate, Route, Routes } from 'react-router-dom'
import RequireAuth from './auth/RequireAuth'
import IndexPage from './pages/IndexPage'
import LoginPage from './pages/LoginPage'
import CreateUserPage from './pages/CreateUserPage'

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
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
