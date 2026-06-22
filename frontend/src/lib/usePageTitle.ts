import { useEffect } from 'react'

const SUFFIX = 'Nisaba'

export function usePageTitle(title?: string | null) {
  useEffect(() => {
    document.title = title ? `${title} · ${SUFFIX}` : SUFFIX
  }, [title])
}
