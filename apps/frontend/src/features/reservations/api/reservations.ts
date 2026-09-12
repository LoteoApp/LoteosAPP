import { ApiError, apiFetch, apiFetchBlob } from '../../../shared/api/client'
import type { Reservation, ReservationListFilters, ReservationPage, SellerOption } from '../types'

const RESERVATIONS_PATH = '/api/v1/reservas'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object'
}

function isState(value: unknown): value is Reservation['estado'] {
  return value === 'activa' || value === 'vencida' || value === 'cancelada' || value === 'convertida'
}

function isActor(value: unknown): value is SellerOption {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.nombre === 'string' &&
    typeof value.apellido === 'string' &&
    typeof value.rol === 'string'
  )
}

function isAgency(value: unknown): value is NonNullable<Reservation['inmobiliaria']> {
  return isRecord(value) && typeof value.id === 'string' && typeof value.razonSocial === 'string'
}

function isReservation(value: unknown): value is Reservation {
  if (!isRecord(value) || !isState(value.estado)) return false
  const client = value.cliente
  return (
    typeof value.id === 'string' &&
    typeof value.loteoId === 'string' &&
    typeof value.loteoNombre === 'string' &&
    typeof value.loteId === 'string' &&
    typeof value.loteNumero === 'string' &&
    typeof value.fechaVencimiento === 'string' &&
    typeof value.fechaCreacion === 'string' &&
    isRecord(client) &&
    typeof client.id === 'string' &&
    typeof client.nombre === 'string' &&
    typeof client.apellido === 'string' &&
    typeof client.dni === 'string' &&
    isActor(value.vendedor) &&
    isActor(value.usuarioAlta) &&
    (value.inmobiliaria === undefined || isAgency(value.inmobiliaria)) &&
    (value.puedeCancelar === undefined || typeof value.puedeCancelar === 'boolean')
  )
}

function isPage(value: unknown): value is ReservationPage {
  return (
    isRecord(value) &&
    Array.isArray(value.reservas) &&
    value.reservas.every(isReservation) &&
    typeof value.pagina === 'number' &&
    typeof value.porPagina === 'number' &&
    typeof value.total === 'number' &&
    typeof value.paginas === 'number'
  )
}

function readBody<T>(body: unknown, valid: (value: unknown) => value is T): T {
  if (!valid(body)) throw new Error(GENERIC_ERROR)
  return body
}

function queryString(filters: ReservationListFilters): string {
  const params = new URLSearchParams()
  if (filters.estado) params.set('estado', filters.estado)
  if (filters.loteoId) params.set('loteoId', filters.loteoId)
  if (filters.loteId) params.set('loteId', filters.loteId)
  if (filters.q) params.set('q', filters.q)
  if (filters.pagina) params.set('pagina', String(filters.pagina))
  if (filters.porPagina) params.set('porPagina', String(filters.porPagina))
  const encoded = params.toString()
  return encoded ? `?${encoded}` : ''
}

export async function listReservations(
  token: string,
  filters: ReservationListFilters = {},
  signal?: AbortSignal,
): Promise<ReservationPage> {
  const body = await apiFetch<unknown>(`${RESERVATIONS_PATH}${queryString(filters)}`, { token, signal })
  return readBody(body, isPage)
}

export async function getReservation(token: string, id: string, signal?: AbortSignal): Promise<Reservation> {
  const body = await apiFetch<unknown>(`${RESERVATIONS_PATH}/${encodeURIComponent(id)}`, { token, signal })
  return readBody(body, isReservation)
}

export async function listEligibleSellers(token: string, loteoId: string, signal?: AbortSignal): Promise<SellerOption[]> {
  const body = await apiFetch<unknown>(`/api/v1/loteos/${encodeURIComponent(loteoId)}/vendedores`, { token, signal })
  if (!isRecord(body) || !Array.isArray(body.vendedores) || !body.vendedores.every(isActor)) {
    throw new Error(GENERIC_ERROR)
  }
  return body.vendedores
}

export async function createReservation(
  token: string,
  values: { loteoId: string; loteId: string; clienteId: string; vendedorId?: string },
  idempotencyKey: string,
): Promise<Reservation> {
  const body = await apiFetch<unknown>(
    `/api/v1/loteos/${encodeURIComponent(values.loteoId)}/lotes/${encodeURIComponent(values.loteId)}/reservas`,
    {
      method: 'POST',
      token,
      body: { clienteId: values.clienteId, vendedorId: values.vendedorId },
      headers: { 'Idempotency-Key': idempotencyKey },
    },
  )
  return readBody(body, isReservation)
}

export async function cancelReservation(token: string, id: string, razon: string): Promise<Reservation> {
  const body = await apiFetch<unknown>(`${RESERVATIONS_PATH}/${encodeURIComponent(id)}/cancelar`, {
    method: 'POST',
    token,
    body: { razon },
  })
  return readBody(body, isReservation)
}

export async function downloadReservationReceipt(token: string, id: string): Promise<Blob> {
  return apiFetchBlob(`${RESERVATIONS_PATH}/${encodeURIComponent(id)}/comprobante`, { token })
}

export function isReservationConflict(error: unknown): boolean {
  return error instanceof ApiError && (
    error.code === 'reservation_already_active' ||
    error.code === 'reservation_lot_unavailable' ||
    error.code === 'lot_state_conflict'
  )
}
