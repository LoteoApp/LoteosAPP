import { useCallback, useEffect, useState } from 'react'
import { messageFromError } from '../../../shared/api/client'
import { getDebtStatement } from '../api/billing'
import type { DebtStatement } from '../types'

export type UseDebtStatement = {
  statement: DebtStatement | null
  isLoading: boolean
  error: string | null
  refresh: () => void
}

export function useDebtStatement(token: string, saleId: string): UseDebtStatement {
  const [statement, setStatement] = useState<DebtStatement | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshKey, setRefreshKey] = useState(0)
  const queryEnabled = token !== '' && saleId !== ''
  const requestKey = JSON.stringify([token, saleId, refreshKey])
  const [loadedKey, setLoadedKey] = useState(requestKey)
  if (requestKey !== loadedKey) {
    setLoadedKey(requestKey)
    setIsLoading(queryEnabled)
    setError(null)
  }

  useEffect(() => {
    if (!queryEnabled) {
      return
    }
    const controller = new AbortController()
    getDebtStatement(token, saleId, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setStatement(loaded)
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setStatement(null)
          setError(messageFromError(loadError))
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setIsLoading(false)
      })
    return () => controller.abort()
  }, [queryEnabled, refreshKey, saleId, token])

  const refresh = useCallback(() => setRefreshKey((value) => value + 1), [])

  return {
    statement: queryEnabled ? statement : null,
    isLoading: queryEnabled && isLoading,
    error: queryEnabled ? error : null,
    refresh,
  }
}
