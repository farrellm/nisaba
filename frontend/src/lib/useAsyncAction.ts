import { useCallback, useState } from 'react'
import { errorMessage } from './errors'

export interface AsyncAction {
  busy: boolean
  error: string | null
  setError: (e: string | null) => void
  // run clears the error, sets busy, awaits fn, and reports failures via
  // errorMessage(err, fallback) (default fallback: "Something went wrong.
  // Try again."). keepBusyOnSuccess leaves busy set after fn resolves, for
  // handlers that navigate away (so the control doesn't re-enable during the
  // transition).
  run: (fn: () => Promise<void>, opts?: { fallback?: string; keepBusyOnSuccess?: boolean }) => void
}

// useAsyncAction owns the setError(null)/setBusy(true)/try/catch/finally
// choreography shared by every submit-style dialog handler.
export function useAsyncAction(): AsyncAction {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const run = useCallback<AsyncAction['run']>((fn, opts) => {
    setError(null)
    setBusy(true)
    void (async () => {
      try {
        await fn()
        if (!opts?.keepBusyOnSuccess) setBusy(false)
      } catch (err) {
        setError(errorMessage(err, opts?.fallback ?? 'Something went wrong. Try again.'))
        setBusy(false)
      }
    })()
  }, [])

  return { busy, error, setError, run }
}
