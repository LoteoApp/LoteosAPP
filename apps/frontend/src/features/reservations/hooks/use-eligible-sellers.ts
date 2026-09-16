import { useEffect, useState } from 'react'
import { listEligibleSellers } from '../api/reservations'
import type { SellerOption } from '../types'

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

export type UseEligibleSellers = {
  sellers: SellerOption[]
  isLoading: boolean
  error: string | null
}

export function useEligibleSellers(token: string, loteoId: string): UseEligibleSellers {
  const [sellers, setSellers] = useState<SellerOption[]>([])
  const [isLoading, setIsLoading] = useState(Boolean(loteoId))
  const [error, setError] = useState<string | null>(null)
  const requestKey = JSON.stringify([token, loteoId])
  const [loadedKey, setLoadedKey] = useState(requestKey)
  if (requestKey !== loadedKey) {
    setLoadedKey(requestKey)
    setSellers([])
    setIsLoading(Boolean(loteoId))
    setError(null)
  }

  useEffect(() => {
    if (!loteoId) {
      return
    }
    const controller = new AbortController()
    listEligibleSellers(token, loteoId, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setSellers(loaded)
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setSellers([])
          setError(messageOf(loadError))
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setIsLoading(false)
      })
    return () => controller.abort()
  }, [loteoId, token])

  return { sellers, isLoading, error }
}
