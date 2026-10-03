import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router'
import { ApiError } from '../shared/api/client'
import { useAuth } from '../features/auth/hooks/use-auth'
import { useLoteo } from '../features/lots/hooks/use-loteo'
import type { LoteoDetail } from '../features/lots/types'
import { getReservation } from '../features/reservations/api/reservations'
import type { Reservation } from '../features/reservations/types'
import { convertReservationToSale } from '../features/sales/api/sales'
import ReservationSalePage from '../features/sales/pages/ReservationSalePage'
import type { ReservationSaleContext, SalePaymentTerms } from '../features/sales/types'
import ReservationLoteoPlan from './ReservationLoteoPlan'

type ReservationLoad =
  | { status: 'loading' }
  | { status: 'loaded'; reservation: Reservation; refreshError?: string }
  | { status: 'not-found' }
  | { status: 'error'; message: string }

export default function ReservationSaleRoute() {
  const { session, user } = useAuth()
  const token = session?.access_token ?? ''
  const userId = user?.id ?? ''
  const { id = '' } = useParams()
  const [refreshCount, setRefreshCount] = useState(0)
  const reservationState = useReservationForSale(token, userId, id, refreshCount)
  const reservation = reservationState.status === 'loaded' ? reservationState.reservation : null
  const developmentState = useLoteo(reservation?.loteoId ?? '', token)
  const development = useKeptDevelopment(
    developmentState.status === 'loaded' ? developmentState.loteo : null,
    userId,
    reservation?.loteoId ?? '',
  )

  const context = useMemo(
    () => (reservation && development ? saleContextFrom(reservation, development) : null),
    [reservation, development],
  )

  const convert = useCallback(
    (terms: SalePaymentTerms, idempotencyKey: string) => convertReservationToSale(token, id, terms, idempotencyKey),
    [token, id],
  )

  const developmentFailed = developmentState.status === 'not-found' || developmentState.status === 'error'
  let status: 'loading' | 'loaded' | 'not-found' | 'error' = 'loading'
  let error: string | undefined
  let refreshError: string | undefined
  if (reservationState.status === 'not-found') {
    status = 'not-found'
  } else if (reservationState.status === 'error') {
    status = 'error'
    error = reservationState.message
  } else if (context !== null) {
    // Data already shown stays while a later reload fails, so the form and an
    // uncertain attempt aren't torn down.
    status = 'loaded'
    refreshError = reservationState.status === 'loaded' && reservationState.refreshError
      ? reservationState.refreshError
      : developmentFailed ? 'No se pudo actualizar el loteo de la reserva.' : undefined
  } else if (developmentFailed) {
    status = 'error'
    error = 'No se pudo cargar el loteo de la reserva.'
  } else if (reservation && development) {
    status = 'error'
    error = 'El lote de la reserva ya no figura en el loteo.'
  }

  // Another user or reserva starts a new attempt: remounting drops the draft,
  // the idempotency key and any answer still on its way. The token is left
  // out because Supabase refreshes it within the same session.
  const attemptIdentity = JSON.stringify([userId, id])

  return (
    <ReservationSalePage
      key={attemptIdentity}
      reservationId={id}
      context={context}
      status={status}
      error={error}
      refreshError={refreshError}
      convert={convert}
      onRefresh={() => setRefreshCount((count) => count + 1)}
      renderPlan={development && reservation && (
        <ReservationLoteoPlan loteo={development} selectedLoteId={reservation.loteId} variant="reference" />
      )}
    />
  )
}

// A refresh (asked for, or a renewed token) keeps the shown reserva until the
// new one arrives, so the typed terms below stay mounted; only another
// reserva or user starts over.
function useReservationForSale(token: string, userId: string, id: string, refreshCount: number): ReservationLoad {
  const [state, setState] = useState<ReservationLoad>({ status: 'loading' })
  const identity = JSON.stringify([userId, id])
  const [loadedIdentity, setLoadedIdentity] = useState(identity)
  if (identity !== loadedIdentity) {
    setLoadedIdentity(identity)
    setState({ status: 'loading' })
  }

  useEffect(() => {
    if (token === '' || id === '') return
    const controller = new AbortController()
    getReservation(token, id, controller.signal)
      .then((reservation) => {
        if (!controller.signal.aborted) setState({ status: 'loaded', reservation })
      })
      .catch((loadError: unknown) => {
        if (controller.signal.aborted) return
        if (loadError instanceof ApiError && loadError.status === 404) {
          setState({ status: 'not-found' })
          return
        }
        const message = loadError instanceof Error ? loadError.message : 'No se pudo cargar la reserva.'
        setState((current) => current.status === 'loaded'
          ? { status: 'loaded', reservation: current.reservation, refreshError: message }
          : { status: 'error', message })
      })
    return () => controller.abort()
  }, [token, id, refreshCount])

  return state
}

// useLoteo starts over on a renewed token; the loteo of the same user and
// reserva stays shown meanwhile so the form isn't torn down.
function useKeptDevelopment(loaded: LoteoDetail | null, userId: string, loteoId: string): LoteoDetail | null {
  const [kept, setKept] = useState<{ userId: string; loteo: LoteoDetail } | null>(null)
  if (loaded !== null && (kept?.loteo !== loaded || kept.userId !== userId)) {
    setKept({ userId, loteo: loaded })
  }
  if (loaded !== null) return loaded
  return kept !== null && kept.userId === userId && kept.loteo.id === loteoId ? kept.loteo : null
}

function saleContextFrom(reservation: Reservation, development: LoteoDetail): ReservationSaleContext | null {
  const lot = development.lotes.find((candidate) => candidate.id === reservation.loteId)
  if (lot === undefined) return null
  const block = development.manzanas.find((candidate) => candidate.id === lot.manzanaId)
  return {
    reservationId: reservation.id,
    reservationState: reservation.estado,
    canConvert: reservation.puedeConvertir === true,
    saleId: reservation.ventaId,
    dueAt: reservation.fechaVencimiento,
    lot: {
      id: lot.id,
      number: lot.numero,
      blockId: lot.manzanaId,
      blockNumber: block?.numero ?? '',
      developmentId: development.id,
      developmentName: development.nombre,
      state: lot.estado,
      price: lot.precio,
      currency: lot.moneda,
      area: lot.superficie,
    },
    client: {
      id: reservation.cliente.id,
      nombre: reservation.cliente.nombre,
      apellido: reservation.cliente.apellido,
      dni: reservation.cliente.dni,
    },
    seller: {
      id: reservation.vendedor.id,
      nombre: reservation.vendedor.nombre,
      apellido: reservation.vendedor.apellido,
      rol: reservation.vendedor.rol,
      inmobiliariaId: reservation.inmobiliaria?.id,
      inmobiliariaRazonSocial: reservation.inmobiliaria?.razonSocial,
    },
  }
}
