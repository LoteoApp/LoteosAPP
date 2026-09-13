import { useEffect, useState } from 'react'
import { listAgencyOptions } from '../api/agencies'
import type { AgencyOption } from '../types'

export type UseAgencyOptions = {
  agencies: AgencyOption[]
  isLoading: boolean
  error: string | null
}

export function useAgencyOptions(token: string): UseAgencyOptions {
  const [agencies, setAgencies] = useState<AgencyOption[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    listAgencyOptions(token, controller.signal)
      .then((loaded) => {
        if (controller.signal.aborted) {
          return
        }
        setAgencies(loaded)
        setError(null)
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setError(loadError instanceof Error ? loadError.message : 'Ocurrió un error inesperado.')
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setIsLoading(false)
        }
      })

    return () => {
      controller.abort()
    }
  }, [token])

  return { agencies, isLoading, error }
}
