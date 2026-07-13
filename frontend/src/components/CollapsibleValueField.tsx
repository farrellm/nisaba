import { InputAdornment, TextField } from '@mui/material'
import UnfoldMore from '@mui/icons-material/UnfoldMore'

// Values longer than this render collapsed until expanded.
const COLLAPSE_THRESHOLD = 80

interface CollapsibleValueFieldProps {
  label: string
  value: string
  expanded: boolean
  onExpand: () => void
  // When present the expanded field is editable; when absent it is read-only.
  onChange?: (value: string) => void
  // Collapsed preview: how many characters to keep and how many rows to show.
  previewLength?: number
  previewRows?: number
}

// CollapsibleValueField renders one key/value as a TextField, collapsing long
// values to a truncated read-only preview that expands on click (or Enter/
// Space). Once expanded a field stays expanded — collapse state is the
// caller's, tracked per key.
export default function CollapsibleValueField({
  label,
  value,
  expanded,
  onExpand,
  onChange,
  previewLength = 40,
  previewRows = 1,
}: CollapsibleValueFieldProps) {
  const collapsed = !expanded && value.length > COLLAPSE_THRESHOLD

  if (collapsed) {
    // Truncate in JS: iOS Safari won't apply -webkit-line-clamp to a
    // <textarea>, so a CSS-only ellipsis goes missing on iPhone.
    const multiline = previewRows > 1
    return (
      <TextField
        label={label}
        value={`${value.slice(0, previewLength)}…`}
        multiline={multiline}
        maxRows={multiline ? previewRows : undefined}
        onClick={onExpand}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onExpand()
          }
        }}
        inputProps={{ tabIndex: 0, 'aria-label': `Expand ${label}` }}
        InputProps={{
          readOnly: true,
          endAdornment: (
            <InputAdornment position="end" sx={{ color: 'text.secondary' }}>
              <UnfoldMore fontSize="small" />
            </InputAdornment>
          ),
          sx: {
            cursor: 'pointer',
            // Clip overflow beyond maxRows instead of showing a scrollbar.
            ...(multiline ? { '& textarea': { overflow: 'hidden !important' } } : {}),
            '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: 'primary.main' },
            '&:focus-within .MuiOutlinedInput-notchedOutline': {
              borderColor: 'primary.main',
              borderWidth: 2,
            },
          },
        }}
      />
    )
  }

  if (!onChange) {
    return (
      <TextField
        label={label}
        value={value}
        InputProps={{ readOnly: true }}
        multiline
        minRows={1}
      />
    )
  }

  return (
    <TextField
      label={label}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      // Once focused, the field counts as expanded so typing past the
      // collapse threshold can't swap the editor out mid-keystroke.
      onFocus={onExpand}
      multiline
      minRows={1}
    />
  )
}
