import { useCallback, useEffect, useMemo, useState } from 'react'
import { messageFromError } from '../../../shared/api/client'
import { listDueInstallments } from '../api/billing'
import type { DueInstallmentFilters, DueInstallmentPage } from '../types'

export type UseDueInstallments = {
  page: DueInstallmentPage
  isLoading: boolean
  error: string | null
  refresh: () => void
}

const emptyPage: DueInstallmentPage = {
  cuotas: [],
  resumen: { cuotasVencidas: 0, cuotasProximas: 0 },
  pagina: 1,
  porPagina: 25,
  total: 0,
  paginas: 0,
}
const SEARCH_DEBOUNCE_MS = 300

export function useDueInstallments(token: string, filters: DueInstallmentFilters): UseDueInstallments {
  const [page, setPage] = useState<DueInstallmentPage>(emptyPage)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshKey, setRefreshKey] = useState(0)
  const search = filters.q?.trim() ?? ''
  const [debouncedSearch, setDebouncedSearch] = useState(search)

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [search])

  const queryEnabled = token !== ''
  const filterKey = JSON.stringify({ ...filters, q: debouncedSearch || undefined })
  const requestFilters = useMemo(() => JSON.parse(filterKey) as DueInstallmentFilters, [filterKey])
  const requestKey = JSON.stringify([token, filterKey, refreshKey])
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
    listDueInstallments(token, requestFilters, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setPage(loaded)
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setPage(emptyPage)
          setError(messageFromError(loadError))
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setIsLoading(false)
      })
    return () => controller.abort()
  }, [queryEnabled, requestFilters, refreshKey, token])

  const refresh = useCallback(() => setRefreshKey((value) => value + 1), [])

  return {
    page: queryEnabled ? page : emptyPage,
    isLoading: queryEnabled && isLoading,
    error: queryEnabled ? error : null,
    refresh,
  }
}
