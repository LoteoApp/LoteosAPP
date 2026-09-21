import { apiFetch } from '../../../shared/api/client'
import { isInstallmentState, isPaymentMedium, isPaymentType, paymentDateToISO } from '../types'
import type {
  Actor,
  Client,
  DebtStatement,
  DownPayment,
  DueInstallment,
  DueInstallmentFilters,
  DueInstallmentPage,
  Installment,
  Payment,
  RegisterPaymentValues,
  SettleSaleValues,
  StatementSale,
} from '../types'

const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object'
}

function isOptionalString(value: unknown): value is string | undefined {
  return value === undefined || typeof value === 'string'
}

function isActor(value: unknown): value is Actor {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.nombre === 'string' &&
    typeof value.apellido === 'string' &&
    typeof value.rol === 'string'
  )
}

function isClient(value: unknown): value is Client {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.nombre === 'string' &&
    typeof value.apellido === 'string' &&
    typeof value.dni === 'string'
  )
}

function isOptionalAgency(value: unknown): boolean {
  return (
    value === undefined ||
    (isRecord(value) && typeof value.id === 'string' && typeof value.razonSocial === 'string')
  )
}

function isInstallment(value: unknown): value is Installment {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.numero === 'number' &&
    typeof value.monto === 'number' &&
    isInstallmentState(value.estado) &&
    typeof value.fechaVencimiento === 'string' &&
    isOptionalString(value.fechaPago) &&
    isOptionalString(value.cobroId)
  )
}

function isDownPayment(value: unknown): value is DownPayment {
  return (
    isRecord(value) &&
    typeof value.monto === 'number' &&
    isInstallmentState(value.estado) &&
    isOptionalString(value.fechaPago) &&
    isOptionalString(value.cobroId)
  )
}

function isStatementSale(value: unknown): value is StatementSale {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.loteoId === 'string' &&
    typeof value.loteoNombre === 'string' &&
    typeof value.loteId === 'string' &&
    typeof value.loteNumero === 'string' &&
    typeof value.manzanaNumero === 'string' &&
    (value.loteSuperficie === null || typeof value.loteSuperficie === 'number') &&
    isClient(value.cliente) &&
    isActor(value.vendedor) &&
    isOptionalAgency(value.inmobiliaria) &&
    typeof value.modalidadPago === 'string' &&
    typeof value.monto === 'number' &&
    typeof value.moneda === 'string' &&
    typeof value.estado === 'string' &&
    typeof value.fechaCreacion === 'string' &&
    (value.planPago === undefined ||
      (isRecord(value.planPago) &&
        typeof value.planPago.id === 'string' &&
        typeof value.planPago.montoEntrega === 'number' &&
        typeof value.planPago.cantidadCuotas === 'number' &&
        typeof value.planPago.tasaInteres === 'number' &&
        typeof value.planPago.periodicidad === 'string' &&
        typeof value.planPago.moneda === 'string' &&
        typeof value.planPago.montoFinanciado === 'number' &&
        typeof value.planPago.montoCuota === 'number' &&
        typeof value.planPago.montoTotal === 'number'))
  )
}

function isPayment(value: unknown): value is Payment {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.ventaId === 'string' &&
    isPaymentType(value.tipo) &&
    typeof value.monto === 'number' &&
    typeof value.moneda === 'string' &&
    isPaymentMedium(value.medioPago) &&
    isOptionalString(value.observacion) &&
    typeof value.incluyeEntrega === 'boolean' &&
    typeof value.montoEntrega === 'number' &&
    Array.isArray(value.cuotas) &&
    value.cuotas.every(isInstallment) &&
    isActor(value.usuarioAlta) &&
    typeof value.fechaPago === 'string' &&
    typeof value.fechaCreacion === 'string'
  )
}

function isStatement(value: unknown): value is DebtStatement {
  if (!isRecord(value) || !isRecord(value.resumen)) return false
  const resumen = value.resumen
  return (
    isStatementSale(value.venta) &&
    (value.entrega === undefined || isDownPayment(value.entrega)) &&
    Array.isArray(value.cuotas) &&
    value.cuotas.every(isInstallment) &&
    typeof resumen.montoTotal === 'number' &&
    typeof resumen.montoPagado === 'number' &&
    typeof resumen.montoPendiente === 'number' &&
    typeof resumen.montoVencido === 'number' &&
    typeof resumen.cuotasPagadas === 'number' &&
    typeof resumen.cuotasPendientes === 'number' &&
    typeof resumen.cuotasVencidas === 'number' &&
    isOptionalString(resumen.proximoVencimiento) &&
    Array.isArray(value.cobros) &&
    value.cobros.every(isPayment) &&
    typeof value.emitidoEl === 'string'
  )
}

function isDueInstallment(value: unknown): value is DueInstallment {
  return (
    isRecord(value) &&
    typeof value.id === 'string' &&
    typeof value.ventaId === 'string' &&
    typeof value.numero === 'number' &&
    typeof value.cantidadCuotas === 'number' &&
    typeof value.monto === 'number' &&
    typeof value.moneda === 'string' &&
    isInstallmentState(value.estado) &&
    typeof value.fechaVencimiento === 'string' &&
    isOptionalString(value.fechaPago) &&
    typeof value.loteoId === 'string' &&
    typeof value.loteoNombre === 'string' &&
    typeof value.loteId === 'string' &&
    typeof value.loteNumero === 'string' &&
    typeof value.manzanaNumero === 'string' &&
    isClient(value.cliente) &&
    isActor(value.vendedor) &&
    isOptionalAgency(value.inmobiliaria)
  )
}

function isDuePage(value: unknown): value is DueInstallmentPage {
  return (
    isRecord(value) &&
    Array.isArray(value.cuotas) &&
    value.cuotas.every(isDueInstallment) &&
    isRecord(value.resumen) &&
    typeof value.resumen.cuotasVencidas === 'number' &&
    typeof value.resumen.cuotasProximas === 'number' &&
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

function queryString(filters: DueInstallmentFilters): string {
  const params = new URLSearchParams()
  if (filters.estado === 'vencida' || filters.estado === 'pendiente' || filters.estado === 'pagada') {
    params.set('estado', filters.estado)
  }
  if (filters.loteoId) params.set('loteoId', filters.loteoId)
  if (filters.q) params.set('q', filters.q)
  if (filters.desde) params.set('desde', filters.desde)
  if (filters.hasta) params.set('hasta', filters.hasta)
  if (filters.pagina) params.set('pagina', String(filters.pagina))
  if (filters.porPagina) params.set('porPagina', String(filters.porPagina))
  const encoded = params.toString()
  return encoded ? `?${encoded}` : ''
}

export async function listDueInstallments(
  token: string,
  filters: DueInstallmentFilters = {},
  signal?: AbortSignal,
): Promise<DueInstallmentPage> {
  const body = await apiFetch<unknown>(`/api/v1/cobranzas/vencimientos${queryString(filters)}`, { token, signal })
  return readBody(body, isDuePage)
}

export async function getDebtStatement(token: string, saleId: string, signal?: AbortSignal): Promise<DebtStatement> {
  const body = await apiFetch<unknown>(`/api/v1/ventas/${encodeURIComponent(saleId)}/estado-deuda`, { token, signal })
  return readBody(body, isStatement)
}

export async function registerPayment(token: string, saleId: string, values: RegisterPaymentValues): Promise<Payment> {
  const fechaPago = paymentDateToISO(values.fechaPago)
  const body = await apiFetch<unknown>(`/api/v1/ventas/${encodeURIComponent(saleId)}/cobros`, {
    method: 'POST',
    token,
    body: {
      cuotaIds: values.cuotaIds,
      incluirEntrega: values.incluirEntrega,
      medioPago: values.medioPago,
      ...(fechaPago === '' ? {} : { fechaPago }),
      ...(values.observacion.trim() === '' ? {} : { observacion: values.observacion.trim() }),
    },
  })
  return readBody(body, isPayment)
}

export async function settleSale(token: string, saleId: string, values: SettleSaleValues): Promise<Payment> {
  const fechaPago = paymentDateToISO(values.fechaPago)
  const body = await apiFetch<unknown>(`/api/v1/ventas/${encodeURIComponent(saleId)}/cancelacion-total`, {
    method: 'POST',
    token,
    body: {
      montoEsperado: values.montoEsperado,
      medioPago: values.medioPago,
      ...(fechaPago === '' ? {} : { fechaPago }),
      ...(values.observacion.trim() === '' ? {} : { observacion: values.observacion.trim() }),
    },
  })
  return readBody(body, isPayment)
}
