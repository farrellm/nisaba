import { useEffect, useState } from 'react'
import { api } from './client'
import type { LLMModel } from './types'

// The model list is a fixed, build-time constant on the server, so fetch it once
// per page load and share the promise across every caller.
let cached: Promise<LLMModel[]> | null = null

function fetchModels(): Promise<LLMModel[]> {
  if (!cached) {
    cached = api.get<LLMModel[]>('/api/models').catch(() => {
      // Don't cache a failure — let the next mount retry.
      cached = null
      return []
    })
  }
  return cached
}

// useModels returns the available models, or [] until they load. Used both by
// the selector and by ResponseView, which renders inside the read-only Anansi
// and Charlotte pages where no model list is otherwise in scope.
export function useModels(): LLMModel[] {
  const [models, setModels] = useState<LLMModel[]>([])

  useEffect(() => {
    let active = true
    fetchModels().then((m) => {
      if (active) setModels(m)
    })
    return () => {
      active = false
    }
  }, [])

  return models
}

// modelLabel resolves a stored model key to its display label, falling back to
// the raw key. The fallback carries legacy ids from the Anansi/Charlotte
// archives, which name models that are no longer in the list.
export function modelLabel(models: LLMModel[], key: string | undefined): string {
  if (!key) return 'no model'
  return models.find((m) => m.id === key)?.label ?? key
}
