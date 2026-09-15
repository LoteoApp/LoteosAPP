import { useCallback, useEffect, useMemo, useState } from 'react'
import { listSales } from '../api/sales'
import type { SaleListFilters, SalePage } from '../types'

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

export type UseSales = {
  page: SalePage
  isLoading: boolean
  error: string | null
  refresh: () => void
}

const emptyPage: SalePage = { ventas: [], pagina: 1, porPagina: 25, total: 0, paginas: 0 }
const SEARCH_DEBOUNCE_MS = 300

export function useSales(token: string, filters: SaleListFilters): UseSales {
  const [page, setPage] = useState<SalePage>(emptyPage)
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
  const requestFilters = useMemo(() => JSON.parse(filterKey) as SaleListFilters, [filterKey])
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
    listSales(token, requestFilters, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setPage(loaded)
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setPage(emptyPage)
          setError(messageOf(loadError))
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
