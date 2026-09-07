import { useCallback, useEffect, useMemo, useState } from 'react'
import { listReservations } from '../api/reservations'
import type { Reservation, ReservationListFilters, ReservationPage } from '../types'

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

export type UseReservations = {
  page: ReservationPage
  isLoading: boolean
  error: string | null
  refresh: () => void
  prepend: (reservation: Reservation) => void
}

export type UseReservationsOptions = {
  enabled?: boolean
}

const emptyPage: ReservationPage = { reservas: [], pagina: 1, porPagina: 25, total: 0, paginas: 0 }

function matchesFilters(reservation: Reservation, filters: ReservationListFilters): boolean {
  if (filters.pagina && filters.pagina !== 1) return false
  if (filters.estado && filters.estado !== reservation.estado) return false
  if (filters.loteoId && filters.loteoId !== reservation.loteoId) return false
  if (filters.loteId && filters.loteId !== reservation.loteId) return false
  const search = filters.q?.trim().toLocaleLowerCase()
  if (!search) return true
  return [
    reservation.loteoNombre,
    reservation.loteNumero,
    reservation.cliente.nombre,
    reservation.cliente.apellido,
    reservation.cliente.dni,
    reservation.vendedor.nombre,
    reservation.vendedor.apellido,
  ].some((value) => value.toLocaleLowerCase().includes(search))
}

export function useReservations(
  token: string,
  filters: ReservationListFilters,
  { enabled = true }: UseReservationsOptions = {},
): UseReservations {
  const [page, setPage] = useState<ReservationPage>(emptyPage)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshKey, setRefreshKey] = useState(0)
  const queryEnabled = enabled && token !== ''
  const filterKey = JSON.stringify(filters)
  const requestFilters = useMemo(() => JSON.parse(filterKey) as ReservationListFilters, [filterKey])
  const requestKey = JSON.stringify([token, filterKey, refreshKey, enabled])
  const [loadedKey, setLoadedKey] = useState(requestKey)
  if (requestKey !== loadedKey) {
    setLoadedKey(requestKey)
    setPage(emptyPage)
    setIsLoading(queryEnabled)
    setError(null)
  }

  useEffect(() => {
    if (!queryEnabled) {
      return
    }

    const controller = new AbortController()
    listReservations(token, requestFilters, controller.signal)
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
  const prepend = useCallback((reservation: Reservation) => {
    if (!matchesFilters(reservation, requestFilters)) return
    setPage((current) => {
      if (current.reservas.some((item) => item.id === reservation.id)) return current
      const reservas = [reservation, ...current.reservas].slice(0, current.porPagina)
      const total = current.total + 1
      return { ...current, reservas, total, paginas: Math.ceil(total / current.porPagina) }
    })
  }, [requestFilters])
  return {
    page: queryEnabled ? page : emptyPage,
    isLoading: queryEnabled && isLoading,
    error: queryEnabled ? error : null,
    refresh,
    prepend,
  }
}
