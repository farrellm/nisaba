import { Button, CircularProgress, type ButtonProps } from '@mui/material'

interface SubmitButtonProps extends ButtonProps {
  busy: boolean
  // Label shown next to the spinner while busy (e.g. "Saving…").
  busyLabel: string
}

// SubmitButton is the confirm button of a submit-style form or dialog: a
// contained button whose label swaps to a small spinner + progress text while
// the action is in flight (and which disables itself while busy).
export default function SubmitButton({
  busy,
  busyLabel,
  disabled,
  children,
  ...rest
}: SubmitButtonProps) {
  return (
    <Button type="submit" variant="contained" disabled={disabled || busy} {...rest}>
      {busy ? (
        <>
          <CircularProgress size={16} color="inherit" sx={{ mr: 1 }} />
          {busyLabel}
        </>
      ) : (
        children
      )}
    </Button>
  )
}
