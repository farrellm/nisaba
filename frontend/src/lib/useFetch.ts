import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import { errorMessage } from './errors'

export interface Fetched<T> {
  data: T | null
  error: string | null
  loading: boolean
  // reload refetches the same path, keeping the current data on screen until
  // the fresh copy lands (so a list doesn't blank while it refreshes).
  reload: () => void
}

// useFetch GETs a JSON resource once on mount (and again whenever path
// changes, resetting to the loading state). It intentionally has no cache or
// dedupe — pages here fetch exactly one resource.
export function useFetch<T>(path: string): Fetched<T> {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [tick, setTick] = useState(0)

  // A new path is a new resource: drop the old one and show loading again
  // (state-reset-on-prop-change, done during render per the React docs).
  const [prevPath, setPrevPath] = useState(path)
  if (path !== prevPath) {
    setPrevPath(path)
    setData(null)
    setError(null)
  }

  useEffect(() => {
    let cancelled = false
    api
      .get<T>(path)
      .then((value) => {
        if (!cancelled) setData(value)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(errorMessage(e))
      })
    return () => {
      cancelled = true
    }
  }, [path, tick])

  const reload = useCallback(() => {
    setError(null)
    setTick((t) => t + 1)
  }, [])

  return { data, error, loading: data === null && error === null, reload }
}
