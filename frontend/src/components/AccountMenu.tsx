import { useState } from 'react'
import {
  Divider,
  IconButton,
  ListItemText,
  Menu,
  MenuItem,
  Switch,
  Tooltip,
  Typography,
} from '@mui/material'
import MoreVertIcon from '@mui/icons-material/MoreVert'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../auth/AuthContext'

// AccountMenu is the masthead's account dropdown: a generic more_vert button
// that opens a menu holding the username, a Settings link, and Log out. Shared
// by every page's masthead so the logout/navigation plumbing lives in one place.
export default function AccountMenu() {
  const { user, logout, refresh } = useAuth()
  const navigate = useNavigate()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const [savingStreaming, setSavingStreaming] = useState(false)
  const open = Boolean(anchorEl)

  function close() {
    setAnchorEl(null)
  }

  async function toggleStreaming() {
    if (savingStreaming) return
    setSavingStreaming(true)
    try {
      await api.put('/api/auth/me', { streamingEnabled: !user?.streamingEnabled })
      await refresh()
    } finally {
      setSavingStreaming(false)
    }
  }

  function goToSettings() {
    close()
    navigate('/settings')
  }

  async function handleLogout() {
    close()
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <>
      <Tooltip title="Account menu">
        <IconButton
          onClick={(e) => setAnchorEl(e.currentTarget)}
          aria-label="Account menu"
          aria-haspopup="true"
          aria-expanded={open ? 'true' : undefined}
          sx={{ color: 'text.primary' }}
        >
          <MoreVertIcon />
        </IconButton>
      </Tooltip>
      <Menu
        anchorEl={anchorEl}
        open={open}
        onClose={close}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <Typography variant="overline" sx={{ color: 'text.secondary', px: 2, py: 0.5, display: 'block' }}>
          {user?.username}
        </Typography>
        <Divider />
        <MenuItem onClick={toggleStreaming} disabled={savingStreaming}>
          <ListItemText>Streaming</ListItemText>
          <Switch
            edge="end"
            size="small"
            checked={user?.streamingEnabled ?? false}
            tabIndex={-1}
            disableRipple
            inputProps={{ 'aria-label': 'Toggle streaming' }}
          />
        </MenuItem>
        <MenuItem onClick={goToSettings}>
          <ListItemText>Settings</ListItemText>
        </MenuItem>
        <MenuItem onClick={handleLogout}>
          <ListItemText>Log out</ListItemText>
        </MenuItem>
      </Menu>
    </>
  )
}
