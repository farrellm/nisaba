import { useState } from 'react'
import { Divider, IconButton, ListItemText, Menu, MenuItem, Typography } from '@mui/material'
import MoreVertIcon from '@mui/icons-material/MoreVert'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

// AccountMenu is the masthead's account dropdown: a generic more_vert button
// that opens a menu holding the username, a Settings link, and Log out. Shared
// by every page's masthead so the logout/navigation plumbing lives in one place.
export default function AccountMenu() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const open = Boolean(anchorEl)

  function close() {
    setAnchorEl(null)
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
      <IconButton
        onClick={(e) => setAnchorEl(e.currentTarget)}
        aria-label="Account menu"
        aria-haspopup="true"
        aria-expanded={open ? 'true' : undefined}
        sx={{ color: 'text.primary' }}
      >
        <MoreVertIcon />
      </IconButton>
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
