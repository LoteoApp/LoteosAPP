import { useEffect, useState } from 'react'
import { messageFromError } from '../../../shared/api/client'
import type { SellerOption } from '../types'

export type UseSaleSellers = {
  sellers: SellerOption[]
  isLoading: boolean
  error: string | null
}

// Who can sell depends on the loteo, so the list is reloaded whenever the
// loteo changes and cleared while the new one is on its way.
export function useSaleSellers(
  developmentId: string,
  loadSellers: (developmentId: string, signal?: AbortSignal) => Promise<SellerOption[]>,
): UseSaleSellers {
  const [sellers, setSellers] = useState<SellerOption[]>([])
  const [isLoading, setIsLoading] = useState(developmentId !== '')
  const [error, setError] = useState<string | null>(null)
  const [loadedDevelopmentId, setLoadedDevelopmentId] = useState(developmentId)

  if (loadedDevelopmentId !== developmentId) {
    setLoadedDevelopmentId(developmentId)
    setSellers([])
    setIsLoading(developmentId !== '')
    setError(null)
  }

  useEffect(() => {
    if (developmentId === '') {
      return
    }

    const controller = new AbortController()

    loadSellers(developmentId, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) {
          setSellers(loaded)
        }
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setError(messageFromError(loadError))
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setIsLoading(false)
        }
      })

    return () => controller.abort()
  }, [developmentId, loadSellers])

  return { sellers, isLoading, error }
}
