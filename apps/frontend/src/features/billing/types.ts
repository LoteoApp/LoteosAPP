// Types that mirror an API body keep the Spanish property names of the JSON
// contract; everything the feature builds for itself is in English.

export const INSTALLMENT_STATES = ['pendiente', 'pagada', 'vencida'] as const

export type InstallmentState = (typeof INSTALLMENT_STATES)[number]

export const INSTALLMENT_STATE_LABELS: Record<InstallmentState, string> = {
  pendiente: 'Pendiente',
  pagada: 'Pagada',
  vencida: 'Vencida',
}

export function isInstallmentState(value: unknown): value is InstallmentState {
  return typeof value === 'string' && (INSTALLMENT_STATES as readonly string[]).includes(value)
}

export const PAYMENT_MEDIUMS = ['efectivo', 'transferencia', 'cheque', 'otro'] as const

export type PaymentMedium = (typeof PAYMENT_MEDIUMS)[number]

export const PAYMENT_MEDIUM_LABELS: Record<PaymentMedium, string> = {
  efectivo: 'Efectivo',
  transferencia: 'Transferencia',
  cheque: 'Cheque',
  otro: 'Otro',
}

export function isPaymentMedium(value: unknown): value is PaymentMedium {
  return typeof value === 'string' && (PAYMENT_MEDIUMS as readonly string[]).includes(value)
}

export const PAYMENT_TYPES = ['pago', 'cancelacion_total'] as const

export type PaymentType = (typeof PAYMENT_TYPES)[number]

export const PAYMENT_TYPE_LABELS: Record<PaymentType, string> = {
  pago: 'Pago de cuotas',
  cancelacion_total: 'Cancelación total',
}

export function isPaymentType(value: unknown): value is PaymentType {
  return typeof value === 'string' && (PAYMENT_TYPES as readonly string[]).includes(value)
}

export const PAYMENT_METHOD_LABELS: Record<string, string> = {
  contado: 'Contado',
  financiado: 'Financiado',
  entrega_financiada: 'Entrega + financiación',
}

export const PAYMENT_PERIOD_LABELS: Record<string, string> = {
  mensual: 'mensual',
  bimestral: 'bimestral',
  trimestral: 'trimestral',
  semestral: 'semestral',
}

export const SALE_STATE_LABELS: Record<string, string> = {
  activa: 'Activa',
  completada: 'Completada',
  cancelada: 'Cancelada',
}

export type Actor = {
  id: string
  nombre: string
  apellido: string
  rol: string
}

export type Client = {
  id: string
  nombre: string
  apellido: string
  dni: string
}

export type Agency = {
  id: string
  razonSocial: string
}

export type Installment = {
  id: string
  numero: number
  monto: number
  estado: InstallmentState
  fechaVencimiento: string
  fechaPago?: string
  cobroId?: string
}

export type DownPayment = {
  monto: number
  estado: InstallmentState
  fechaPago?: string
  cobroId?: string
}

// The venta as the statement embeds it: the plan summary without cuotas,
// which the statement lists on its own with their state as of today.
export type StatementSale = {
  id: string
  loteoId: string
  loteoNombre: string
  loteId: string
  loteNumero: string
  manzanaNumero: string
  loteSuperficie: number | null
  cliente: Client
  vendedor: Actor
  inmobiliaria?: Agency
  modalidadPago: string
  monto: number
  moneda: string
  estado: string
  fechaCreacion: string
  planPago?: {
    id: string
    montoEntrega: number
    cantidadCuotas: number
    tasaInteres: number
    periodicidad: string
    moneda: string
    montoFinanciado: number
    montoCuota: number
    montoTotal: number
  }
}

export type Payment = {
  id: string
  ventaId: string
  tipo: PaymentType
  monto: number
  moneda: string
  medioPago: PaymentMedium
  observacion?: string
  incluyeEntrega: boolean
  montoEntrega: number
  cuotas: Installment[]
  usuarioAlta: Actor
  fechaPago: string
  fechaCreacion: string
}

export type DebtSummary = {
  montoTotal: number
  montoPagado: number
  montoPendiente: number
  montoVencido: number
  cuotasPagadas: number
  cuotasPendientes: number
  cuotasVencidas: number
  proximoVencimiento?: string
}

export type DebtStatement = {
  venta: StatementSale
  entrega?: DownPayment
  cuotas: Installment[]
  resumen: DebtSummary
  cobros: Payment[]
  emitidoEl: string
}

export type DueInstallment = {
  id: string
  ventaId: string
  numero: number
  cantidadCuotas: number
  monto: number
  moneda: string
  estado: InstallmentState
  fechaVencimiento: string
  fechaPago?: string
  loteoId: string
  loteoNombre: string
  loteId: string
  loteNumero: string
  manzanaNumero: string
  cliente: Client
  vendedor: Actor
  inmobiliaria?: Agency
}

export type DueSummary = {
  cuotasVencidas: number
  cuotasProximas: number
}

export type DueInstallmentPage = {
  cuotas: DueInstallment[]
  resumen: DueSummary
  pagina: number
  porPagina: number
  total: number
  paginas: number
}

// The list's estado filter: `pendientes` is the API default (pendiente and
// vencida together), the others narrow to one state.
export const DUE_FILTERS = ['pendientes', 'vencida', 'pendiente', 'pagada'] as const

export type DueFilter = (typeof DUE_FILTERS)[number]

export const DUE_FILTER_LABELS: Record<DueFilter, string> = {
  pendientes: 'A cobrar',
  vencida: 'Vencidas',
  pendiente: 'Al día',
  pagada: 'Pagadas',
}

export function isDueFilter(value: unknown): value is DueFilter {
  return typeof value === 'string' && (DUE_FILTERS as readonly string[]).includes(value)
}

export type DueInstallmentFilters = {
  estado?: DueFilter
  loteoId?: string
  q?: string
  desde?: string
  hasta?: string
  pagina?: number
  porPagina?: number
}

export type PaymentTerms = {
  medioPago: PaymentMedium
  // Calendar day (YYYY-MM-DD) as the date input holds it; empty means today.
  fechaPago: string
  observacion: string
}

export const EMPTY_PAYMENT_TERMS: PaymentTerms = {
  medioPago: 'efectivo',
  fechaPago: '',
  observacion: '',
}

export type RegisterPaymentValues = PaymentTerms & {
  cuotaIds: string[]
  incluirEntrega: boolean
}

export type SettleSaleValues = PaymentTerms & {
  montoEsperado: number
}

export const MAX_OBSERVATION_LENGTH = 500

// Turns the day typed in the date input into the instant sent to the API:
// noon in the browser's zone, so the day survives the round trip to UTC
// whatever the offset. Empty stays empty (the API takes "now").
export function paymentDateToISO(day: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(day.trim())
  if (match === null) {
    return ''
  }
  const date = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]), 12)
  return Number.isNaN(date.getTime()) ? '' : date.toISOString()
}

// Today as the date input expects it, in the browser's zone.
export function todayInputValue(now = new Date()): string {
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${now.getFullYear()}-${month}-${day}`
}

export function validatePaymentTerms(terms: PaymentTerms, today = todayInputValue()): string | null {
  if (!isPaymentMedium(terms.medioPago)) {
    return 'Elegí el medio de pago.'
  }
  if (terms.fechaPago !== '' && paymentDateToISO(terms.fechaPago) === '') {
    return 'Ingresá una fecha de pago válida.'
  }
  if (terms.fechaPago !== '' && terms.fechaPago > today) {
    return 'La fecha de pago no puede ser futura.'
  }
  if (terms.observacion.trim().length > MAX_OBSERVATION_LENGTH) {
    return `La observación no puede superar los ${MAX_OBSERVATION_LENGTH} caracteres.`
  }
  return null
}

export function roundMoney(value: number): number {
  return Math.round(value * 100) / 100
}

// The cuotas still owed, in the order they have to be collected.
export function pendingInstallments(cuotas: readonly Installment[]): Installment[] {
  return cuotas.filter((cuota) => cuota.estado !== 'pagada')
}

// Cuotas are collected in order, so toggling one selects (or unselects)
// every pending cuota up to it: clicking cuota 3 with nothing selected takes
// 1, 2 and 3; clicking 2 afterwards leaves only 1.
export function toggleInstallment(
  cuotas: readonly Installment[],
  selected: readonly string[],
  id: string,
): string[] {
  const pending = pendingInstallments(cuotas)
  const position = pending.findIndex((cuota) => cuota.id === id)
  if (position === -1) {
    return [...selected]
  }
  const selectedCount = pending.filter((cuota) => selected.includes(cuota.id)).length
  const nextCount = position + 1 <= selectedCount ? position : position + 1
  return pending.slice(0, nextCount).map((cuota) => cuota.id)
}

export function selectionAmount(
  statement: Pick<DebtStatement, 'cuotas' | 'entrega'>,
  selected: readonly string[],
  includeDownPayment: boolean,
): number {
  let total = 0
  if (includeDownPayment && statement.entrega !== undefined && statement.entrega.estado !== 'pagada') {
    total += statement.entrega.monto
  }
  for (const cuota of statement.cuotas) {
    if (selected.includes(cuota.id)) {
      total += cuota.monto
    }
  }
  return roundMoney(total)
}

export function lotLabel(item: Pick<DueInstallment, 'loteoNombre' | 'manzanaNumero' | 'loteNumero'>): string {
  const block = item.manzanaNumero || '—'
  const number = item.loteNumero || '—'
  return `${item.loteoNombre} · Mz ${block} · Lote ${number}`
}

export function clientLabel(client: Pick<Client, 'nombre' | 'apellido'>): string {
  return `${client.apellido}, ${client.nombre}`
}

export function paymentItemsLabel(payment: Pick<Payment, 'incluyeEntrega' | 'cuotas'>): string {
  const parts: string[] = []
  if (payment.incluyeEntrega) {
    parts.push('Entrega')
  }
  const numbers = payment.cuotas.map((cuota) => cuota.numero)
  if (numbers.length === 1) {
    parts.push(`Cuota ${numbers[0]}`)
  } else if (numbers.length > 1) {
    parts.push(`Cuotas ${numbers[0]} a ${numbers[numbers.length - 1]}`)
  }
  return parts.join(' + ') || '—'
}
