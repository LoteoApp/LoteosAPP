import { useEffect, useState } from 'react'
import { listAgencies, type AgencyOption } from '../api/agencies'

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

export type UseAgencyOptions = {
  agencies: AgencyOption[]
  isLoading: boolean
  error: string | null
}

// Fetches the agencies catalog for the inmobiliaria field on the user
// creation form. enabled gates the request so it only fires while rol is
// inmobiliaria, not on every form open.
export function useAgencyOptions(token: string, enabled: boolean): UseAgencyOptions {
  const [agencies, setAgencies] = useState<AgencyOption[]>([])
  const [isLoading, setIsLoading] = useState(enabled)
  const [error, setError] = useState<string | null>(null)
  const requestKey = JSON.stringify([token, enabled])
  const [loadedKey, setLoadedKey] = useState(requestKey)
  if (requestKey !== loadedKey) {
    setLoadedKey(requestKey)
    setIsLoading(enabled)
    setError(null)
  }

  useEffect(() => {
    if (!enabled) {
      return
    }
    const controller = new AbortController()
    listAgencies(token, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) {
          setAgencies(loaded)
          setError(null)
        }
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setError(messageOf(loadError))
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setIsLoading(false)
        }
      })
    return () => controller.abort()
  }, [enabled, token])

  return { agencies, isLoading, error }
}
