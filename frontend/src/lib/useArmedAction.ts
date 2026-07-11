import { useCallback, useEffect, useRef, useState } from 'react'

// useArmedAction implements the two-step destructive action shared by block
// and document deletion: the first fire() arms the control (and starts a
// timer that quietly disarms it), the second fire() confirms by calling
// onConfirm. disarm() cancels an armed state (e.g. when a menu closes); the
// pending timer is also cleared on unmount.
export function useArmedAction(onConfirm: () => void, disarmMs = 4000) {
  const [armed, setArmed] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout>>()

  useEffect(() => () => clearTimeout(timer.current), [])

  const disarm = useCallback(() => {
    clearTimeout(timer.current)
    setArmed(false)
  }, [])

  const fire = useCallback(() => {
    if (!armed) {
      setArmed(true)
      timer.current = setTimeout(() => setArmed(false), disarmMs)
      return
    }
    clearTimeout(timer.current)
    onConfirm()
  }, [armed, onConfirm, disarmMs])

  return { armed, fire, disarm }
}
