import { apiFetch } from '../../../shared/api/client'
import { isSaleState, PAYMENT_METHODS } from '../types'
import type { CreateSaleValues, Sale, SaleActor, SaleListFilters, SalePage } from '../types'

const SALES_PATH = '/api/v1/ventas'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object'
}

function isActor(value: unknown): value is SaleActor {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.nombre === 'string' &&
    typeof value.apellido === 'string' &&
    typeof value.rol === 'string'
  )
}

function isSale(value: unknown): value is Sale {
  if (!isRecord(value) || !isSaleState(value.estado)) return false
  const client = value.cliente
  return (
    typeof value.id === 'string' &&
    typeof value.loteoId === 'string' &&
    typeof value.loteoNombre === 'string' &&
    typeof value.loteId === 'string' &&
    typeof value.loteNumero === 'string' &&
    typeof value.manzanaNumero === 'string' &&
    (value.loteSuperficie === null || typeof value.loteSuperficie === 'number') &&
    isRecord(client) &&
    typeof client.id === 'string' &&
    typeof client.nombre === 'string' &&
    typeof client.apellido === 'string' &&
    typeof client.dni === 'string' &&
    isActor(value.vendedor) &&
    isActor(value.usuarioAlta) &&
    (value.inmobiliaria === undefined ||
      (isRecord(value.inmobiliaria) &&
        typeof value.inmobiliaria.id === 'string' &&
        typeof value.inmobiliaria.razonSocial === 'string')) &&
    typeof value.modalidadPago === 'string' &&
    (PAYMENT_METHODS as readonly string[]).includes(value.modalidadPago) &&
    typeof value.monto === 'number' &&
    typeof value.moneda === 'string' &&
    typeof value.fechaCreacion === 'string' &&
    typeof value.fechaModificacion === 'string'
  )
}

function isPage(value: unknown): value is SalePage {
  return (
    isRecord(value) &&
    Array.isArray(value.ventas) &&
    value.ventas.every(isSale) &&
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

function queryString(filters: SaleListFilters): string {
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

export async function listSales(token: string, filters: SaleListFilters = {}, signal?: AbortSignal): Promise<SalePage> {
  const body = await apiFetch<unknown>(`${SALES_PATH}${queryString(filters)}`, { token, signal })
  return readBody(body, isPage)
}

export async function getSale(token: string, id: string, signal?: AbortSignal): Promise<Sale> {
  const body = await apiFetch<unknown>(`${SALES_PATH}/${encodeURIComponent(id)}`, { token, signal })
  return readBody(body, isSale)
}

export async function createSale(
  token: string,
  values: CreateSaleValues,
  idempotencyKey: string,
): Promise<Sale> {
  const body = await apiFetch<unknown>(
    `/api/v1/loteos/${encodeURIComponent(values.loteoId)}/lotes/${encodeURIComponent(values.loteId)}/ventas`,
    {
      method: 'POST',
      token,
      body: { clienteId: values.clienteId, vendedorId: values.vendedorId, modalidadPago: values.modalidadPago },
      headers: { 'Idempotency-Key': idempotencyKey },
    },
  )
  return readBody(body, isSale)
}
