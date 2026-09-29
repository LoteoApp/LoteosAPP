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

// The cargos adicionales a cobro may carry besides the cuota: taxes, fees,
// services. Each one has its own currency, so a cuota in USD can be
// collected together with services in ARS; nothing is ever converted.
export const CHARGE_TYPES = [
  'impuesto_municipal',
  'impuesto_provincial',
  'gasto_administrativo',
  'honorarios',
  'servicios',
  'cargo_inmobiliaria',
  'otros',
] as const

export type ChargeType = (typeof CHARGE_TYPES)[number]

export const CHARGE_TYPE_LABELS: Record<ChargeType, string> = {
  impuesto_municipal: 'Impuesto municipal',
  impuesto_provincial: 'Impuesto provincial',
  gasto_administrativo: 'Gasto administrativo',
  honorarios: 'Honorarios',
  servicios: 'Servicios',
  cargo_inmobiliaria: 'Cargo de inmobiliaria',
  otros: 'Otros',
}

export function isChargeType(value: unknown): value is ChargeType {
  return typeof value === 'string' && (CHARGE_TYPES as readonly string[]).includes(value)
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

export type PaymentCharge = {
  id: string
  tipo: ChargeType
  monto: number
  moneda: string
  detalle?: string
}

// What a cobro adds up to in one currency. A cobro that mixes a cuota in USD
// with services in ARS has one of these per currency.
export type CurrencyTotal = {
  moneda: string
  monto: number
}

// `monto`/`moneda` are only what went to the plan, in the sale's currency;
// `totales` is what the client actually handed over, per currency.
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
  cargos: PaymentCharge[]
  totales: CurrencyTotal[]
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
  // What the cobros charged beyond the plan, per currency. It isn't debt of
  // the plan and may be in a currency the plan never uses.
  cargosCobrados: CurrencyTotal[]
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

// DevelopmentOption is a loteo as the vencimientos filter lists it.
export type DevelopmentOption = {
  id: string
  nombre: string
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

// A cargo as the API receives it. An empty `moneda` means the sale's.
export type ChargeInput = {
  tipo: ChargeType
  monto: number
  moneda: string
  detalle: string
}

// A charge row as the form holds it while the user types: strings, so a
// half-typed number never snaps to something else under their cursor.
export type ChargeRow = {
  // Stable key for React; it never leaves the form.
  key: string
  tipo: ChargeType
  monto: string
  moneda: string
  detalle: string
}

export type RegisterPaymentValues = PaymentTerms & {
  cuotaIds: string[]
  incluirEntrega: boolean
  cargos: ChargeInput[]
}

export type SettleSaleValues = PaymentTerms & {
  montoEsperado: number
  cargos: ChargeInput[]
}

export const MAX_OBSERVATION_LENGTH = 500
export const MAX_PAYMENT_CHARGES = 20
export const MAX_CHARGE_DETAIL_LENGTH = 200
export const MAX_CURRENCY_CODE_LENGTH = 10
const MONEY_DECIMALS = 2

let chargeRowSequence = 0

export function newChargeRow(currency: string): ChargeRow {
  chargeRowSequence += 1
  return { key: `cargo-${chargeRowSequence}`, tipo: 'servicios', monto: '', moneda: currency, detalle: '' }
}

// Accepts "1234.5", "1234,5" and "1.234,50": with a comma present the dots
// are thousands separators, as people type prices here. More decimals than
// `maxDecimals` is a typo, not something to round.
function parseDecimal(value: string, maxDecimals: number): number | null {
  const trimmed = value.trim()
  const normalized = trimmed.includes(',') ? trimmed.replace(/\./g, '').replace(',', '.') : trimmed
  const match = /^\d+(?:\.(\d+))?$/.exec(normalized)
  if (match === null || (match[1]?.length ?? 0) > maxDecimals) {
    return null
  }
  return Number(normalized)
}

export function normalizeCurrency(currency: string, fallback: string): string {
  const normalized = currency.trim().toUpperCase()
  return normalized === '' ? fallback.trim().toUpperCase() : normalized
}

// The currencies a charge may be in. `lotes.moneda` is free text, so the
// sale's own currency leads the list even when it isn't one of these.
const COMMON_CURRENCIES = ['ARS', 'USD'] as const

export function currencyOptions(saleCurrency: string): string[] {
  const sale = normalizeCurrency(saleCurrency, '')
  const options = COMMON_CURRENCIES.filter((currency) => currency !== sale)
  return sale === '' ? [...options] : [sale, ...options]
}

export type ChargesResult = { ok: true; charges: ChargeInput[] } | { ok: false; error: string }

// Validates the typed charges with the same rules the backend applies, so
// the confirm button explains what is missing before the request is sent.
export function parseCharges(rows: readonly ChargeRow[], fallbackCurrency: string): ChargesResult {
  if (rows.length > MAX_PAYMENT_CHARGES) {
    return { ok: false, error: `No se pueden cargar más de ${MAX_PAYMENT_CHARGES} cargos adicionales.` }
  }
  const charges: ChargeInput[] = []
  for (const row of rows) {
    if (!isChargeType(row.tipo)) {
      return { ok: false, error: 'Elegí el tipo de cada cargo adicional.' }
    }
    const amount = parseDecimal(row.monto, MONEY_DECIMALS)
    if (amount === null || amount <= 0) {
      return {
        ok: false,
        error: `Ingresá el monto de "${CHARGE_TYPE_LABELS[row.tipo]}", mayor a cero y con hasta 2 decimales.`,
      }
    }
    const currency = normalizeCurrency(row.moneda, fallbackCurrency)
    if (currency === '' || currency.length > MAX_CURRENCY_CODE_LENGTH) {
      return { ok: false, error: `Ingresá la moneda de "${CHARGE_TYPE_LABELS[row.tipo]}".` }
    }
    if (row.detalle.trim().length > MAX_CHARGE_DETAIL_LENGTH) {
      return { ok: false, error: `El detalle del cargo no puede superar los ${MAX_CHARGE_DETAIL_LENGTH} caracteres.` }
    }
    charges.push({ tipo: row.tipo, monto: amount, moneda: currency, detalle: row.detalle.trim() })
  }
  return { ok: true, charges }
}

// Mirrors domain.PaymentTotals on the backend: what a cobro adds up to per
// currency, the plan's currency first and the rest alphabetically. Amounts
// in different currencies are never added together.
export function paymentTotals(
  planAmount: number,
  planCurrency: string,
  charges: readonly { monto: number; moneda: string }[],
): CurrencyTotal[] {
  const plan = normalizeCurrency(planCurrency, '')
  const amounts = new Map<string, number>()
  if (planAmount > 0) {
    amounts.set(plan, planAmount)
  }
  for (const charge of charges) {
    const currency = normalizeCurrency(charge.moneda, plan)
    amounts.set(currency, (amounts.get(currency) ?? 0) + charge.monto)
  }
  const others = [...amounts.keys()].filter((currency) => currency !== plan).sort()
  const ordered = amounts.has(plan) ? [plan, ...others] : others
  return ordered.map((moneda) => ({ moneda, monto: roundMoney(amounts.get(moneda) ?? 0) }))
}

export function chargeTotals(charges: readonly { monto: number; moneda: string }[]): CurrencyTotal[] {
  return paymentTotals(0, '', charges)
}

// The instant the API receives for the chosen day. Today is sent as nothing
// at all, so the backend stamps its own "now": anchoring today at noon would
// be a future instant all morning and the backend rejects a payment dated in
// the future. A past day keeps its noon anchor.
export function paymentInstant(day: string, today = todayInputValue()): string {
  return day.trim() === today ? '' : paymentDateToISO(day)
}

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

export function chargeLabel(charge: Pick<PaymentCharge, 'tipo' | 'detalle'>): string {
  const detail = charge.detalle?.trim() ?? ''
  return detail === '' ? CHARGE_TYPE_LABELS[charge.tipo] : `${CHARGE_TYPE_LABELS[charge.tipo]} · ${detail}`
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
