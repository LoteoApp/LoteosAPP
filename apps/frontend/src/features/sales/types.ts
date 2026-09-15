// The property names are the wire contract of /api/v1/lotes, so they keep the
// Spanish the API publishes even though the symbols are in English.
export const LOT_STATES = ['disponible', 'reservado', 'vendido', 'finalizado'] as const

export type LotState = (typeof LOT_STATES)[number]

export type LoteOption = {
  id: string
  numero: string
  manzanaId: string
  manzanaNumero: string
  loteoId: string
  loteoNombre: string
  estado: LotState
  precio: number | null
  moneda: string
  superficie: number | null
}

export function isLotState(value: unknown): value is LotState {
  return typeof value === 'string' && (LOT_STATES as readonly string[]).includes(value)
}

// A lote is identified by its loteo, manzana and number together: the number
// alone repeats across manzanas and loteos.
export function loteOptionLabel(lote: LoteOption): string {
  const manzana = lote.manzanaNumero || '—'
  const numero = lote.numero || '—'
  return `${lote.loteoNombre} · Mz ${manzana} · Lote ${numero}`
}

// Sales never imports another feature's files, so it states the shape it needs
// from a cliente, an inmobiliaria and a vendedor. The objects `app` injects
// satisfy these structurally.
export type ClienteOption = {
  id: string
  nombre: string
  apellido: string
  dni: string
}

export type AgencyOption = {
  id: string
  razonSocial: string
}

// `ventas.vendedor_id` points at a usuario, and the agency is read from that
// user's profile, so the sale stores the seller and never the agency.
export type SellerOption = {
  id: string
  nombre: string
  apellido: string
  rol: string
  // Marks the logged-in user, whose usuario id the client does not know.
  esActor?: boolean
  inmobiliariaId?: string
  inmobiliariaRazonSocial?: string
}

// Administradores and administrativos sell without an agency, so they need a
// group of their own in the selector that narrows the sellers.
export const DIRECT_SALE: AgencyOption = { id: 'directa', razonSocial: 'Venta directa' }

export function sellerOptionLabel(seller: SellerOption): string {
  return `${seller.apellido}, ${seller.nombre}`
}

export function agencyOfSeller(seller: SellerOption): AgencyOption {
  const id = seller.inmobiliariaId ?? ''
  if (id === '') {
    return DIRECT_SALE
  }
  return { id, razonSocial: seller.inmobiliariaRazonSocial ?? '' }
}

export function sellerAgencyLabel(seller: SellerOption): string {
  return agencyOfSeller(seller).razonSocial
}

export function agencyOptionsFromSellers(sellers: readonly SellerOption[]): AgencyOption[] {
  const agencies = new Map<string, AgencyOption>()
  let hasDirectSale = false

  for (const seller of sellers) {
    const agency = agencyOfSeller(seller)
    if (agency.id === DIRECT_SALE.id) {
      hasDirectSale = true
      continue
    }
    agencies.set(agency.id, agency)
  }

  const options = [...agencies.values()].sort((left, right) =>
    left.razonSocial.localeCompare(right.razonSocial, 'es'),
  )

  return hasDirectSale ? [DIRECT_SALE, ...options] : options
}

// Whoever registers the sale is the seller unless they say otherwise, so the
// form opens already resolved for the usual case: an administrativo selling
// directly. Falling back to a lone eligible seller covers an agency user, who
// can only sell for their own agency.
export function defaultSeller(sellers: readonly SellerOption[]): SellerOption | null {
  const actor = sellers.find((seller) => seller.esActor === true)
  if (actor !== undefined) {
    return actor
  }
  return sellers.length === 1 ? sellers[0] : null
}

export function sellersOfAgency(
  sellers: readonly SellerOption[],
  agency: AgencyOption | null,
): SellerOption[] {
  if (agency === null) {
    return []
  }
  return sellers.filter((seller) => agencyOfSeller(seller).id === agency.id)
}

export type NewClientValues = {
  nombre: string
  apellido: string
  dni: string
  celular: string
  email: string
}

export function clienteOptionLabel(cliente: ClienteOption): string {
  return `${cliente.apellido}, ${cliente.nombre} · DNI ${cliente.dni}`
}

// The three modalidades of the ventas.modalidad_pago contract.
export const PAYMENT_METHODS = ['contado', 'financiado', 'entrega_financiada'] as const

export type PaymentMethod = (typeof PAYMENT_METHODS)[number]

export const PAYMENT_METHOD_LABELS: Record<PaymentMethod, string> = {
  contado: 'Contado',
  financiado: 'Financiado',
  entrega_financiada: 'Entrega + financiación',
}

export function isPaymentMethod(value: unknown): value is PaymentMethod {
  return typeof value === 'string' && (PAYMENT_METHODS as readonly string[]).includes(value)
}

export function isFinancedMethod(method: PaymentMethod): boolean {
  return method !== 'contado'
}

export const PAYMENT_PERIODS = ['mensual', 'bimestral', 'trimestral', 'semestral'] as const

export type PaymentPeriod = (typeof PAYMENT_PERIODS)[number]

export const PAYMENT_PERIOD_LABELS: Record<PaymentPeriod, string> = {
  mensual: 'Mensual',
  bimestral: 'Bimestral',
  trimestral: 'Trimestral',
  semestral: 'Semestral',
}

export function isPaymentPeriod(value: unknown): value is PaymentPeriod {
  return typeof value === 'string' && (PAYMENT_PERIODS as readonly string[]).includes(value)
}

export const MAX_SALE_INSTALLMENTS = 360
export const MAX_SALE_INTEREST_RATE = 1000

// The plan as POST .../ventas receives it. tasaInteres is a percentage and
// montoEntrega is 0 unless the modalidad is entrega_financiada.
export type PaymentPlanInput = {
  cantidadCuotas: number
  tasaInteres: number
  periodicidad: PaymentPeriod
  montoEntrega: number
}

// What the form holds while the user types: strings, so a half-typed number
// never snaps to something else under their cursor.
export type PaymentPlanValues = {
  cantidadCuotas: string
  tasaInteres: string
  periodicidad: PaymentPeriod
  montoEntrega: string
}

export const EMPTY_PAYMENT_PLAN: PaymentPlanValues = {
  cantidadCuotas: '',
  tasaInteres: '',
  periodicidad: 'mensual',
  montoEntrega: '',
}

export type PaymentSchedule = {
  montoFinanciado: number
  montoTotal: number
  montoCuota: number
  cuotas: number[]
}

// Mirrors domain.RoundMoney on the backend: Math.round(value * 100) / 100
// over the same IEEE 754 doubles, so both sides round every positive amount
// the same way.
export function roundMoney(value: number): number {
  return Math.round(value * 100) / 100
}

// Formula (simple interest on the financed amount; keep in sync with
// domain.BuildPaymentSchedule in apps/backend):
//
//   financiado = round2(monto - montoEntrega)
//   total      = round2(financiado * (1 + tasaInteres / 100))
//   cuota      = round2(total / cantidadCuotas)
//   última     = round2(total - cuota * (cantidadCuotas - 1))
//
// The last cuota absorbs the rounding remainder so the cuotas add up to
// total exactly. Due dates are derived by the backend from the sale date.
export function buildPaymentSchedule(monto: number, plan: PaymentPlanInput): PaymentSchedule {
  const montoFinanciado = roundMoney(monto - plan.montoEntrega)
  const montoTotal = roundMoney(montoFinanciado * (1 + plan.tasaInteres / 100))
  const montoCuota = roundMoney(montoTotal / plan.cantidadCuotas)
  const last = roundMoney(montoTotal - montoCuota * (plan.cantidadCuotas - 1))
  const cuotas = Array.from({ length: plan.cantidadCuotas }, (_, index) =>
    index === plan.cantidadCuotas - 1 ? last : montoCuota,
  )
  return { montoFinanciado, montoTotal, montoCuota, cuotas }
}

// Accepts "1234.5", "1234,5" and "1.234,50": with a comma present the dots
// are thousands separators, as people type prices here.
function parseDecimal(value: string): number | null {
  const trimmed = value.trim()
  const normalized = trimmed.includes(',')
    ? trimmed.replace(/\./g, '').replace(',', '.')
    : trimmed
  if (!/^\d+(\.\d+)?$/.test(normalized)) {
    return null
  }
  return Number(normalized)
}

export type PaymentPlanResult =
  | { ok: true; plan: PaymentPlanInput | undefined }
  | { ok: false; error: string }

// Validates the typed plan against the modalidad and the lote price. The
// messages mirror the backend rules so the confirm button explains what is
// missing before the request is sent.
export function parsePaymentPlan(
  method: PaymentMethod,
  values: PaymentPlanValues,
  monto: number,
): PaymentPlanResult {
  if (!isFinancedMethod(method)) {
    return { ok: true, plan: undefined }
  }

  const cantidadCuotas = Number(values.cantidadCuotas.trim())
  if (
    values.cantidadCuotas.trim() === '' ||
    !Number.isInteger(cantidadCuotas) ||
    cantidadCuotas < 1 ||
    cantidadCuotas > MAX_SALE_INSTALLMENTS
  ) {
    return {
      ok: false,
      error: `Ingresá una cantidad de cuotas entre 1 y ${MAX_SALE_INSTALLMENTS}.`,
    }
  }

  const tasaInteres = values.tasaInteres.trim() === '' ? 0 : parseDecimal(values.tasaInteres)
  if (tasaInteres === null || tasaInteres < 0 || tasaInteres > MAX_SALE_INTEREST_RATE) {
    return {
      ok: false,
      error: `Ingresá una tasa de interés entre 0 y ${MAX_SALE_INTEREST_RATE} %.`,
    }
  }

  let montoEntrega = 0
  if (method === 'entrega_financiada') {
    const parsed = parseDecimal(values.montoEntrega)
    if (parsed === null || parsed <= 0) {
      return { ok: false, error: 'Ingresá el monto de la entrega.' }
    }
    if (parsed >= monto) {
      return { ok: false, error: 'La entrega tiene que ser menor al precio del lote.' }
    }
    montoEntrega = parsed
  }

  const plan: PaymentPlanInput = {
    cantidadCuotas,
    tasaInteres,
    periodicidad: values.periodicidad,
    montoEntrega,
  }
  const schedule = buildPaymentSchedule(monto, plan)
  if (schedule.montoCuota < 0.01 || schedule.cuotas[schedule.cuotas.length - 1] < 0.01) {
    return { ok: false, error: 'El monto financiado no alcanza para esa cantidad de cuotas.' }
  }
  return { ok: true, plan }
}

export type SaleDraft = {
  lote: LoteOption | null
  cliente: ClienteOption | null
  seller: SellerOption | null
  method: PaymentMethod
  plan: PaymentPlanValues
}

// The plan as the receipt prints it: the typed terms plus the derived
// amounts, all of which the backend also publishes in Sale.planPago.
export type SaleReceiptPlan = {
  montoEntrega: number
  cantidadCuotas: number
  tasaInteres: number
  periodicidad: PaymentPeriod
  montoFinanciado: number
  montoCuota: number
  montoTotal: number
}

export type SaleReceipt = {
  emitidoEl: string
  lote: LoteOption
  cliente: ClienteOption
  seller: SellerOption
  method: PaymentMethod
  monto: number
  moneda: string
  plan?: SaleReceiptPlan
}

export type SaleReceiptResult =
  | { ok: true; receipt: SaleReceipt; plan: PaymentPlanInput | undefined }
  | { ok: false; error: string }

export function buildSaleReceipt(draft: SaleDraft, emitidoEl: string): SaleReceiptResult {
  const { lote, cliente, seller, method } = draft

  if (lote === null) {
    return { ok: false, error: 'Elegí el lote que se vende.' }
  }

  if (cliente === null) {
    return { ok: false, error: 'Elegí el cliente comprador.' }
  }

  if (seller === null) {
    return { ok: false, error: 'Elegí el vendedor que realizó la venta.' }
  }

  if (lote.precio === null) {
    return {
      ok: false,
      error: 'El lote no tiene precio cargado. Cargalo en el detalle del loteo antes de vender.',
    }
  }

  const parsed = parsePaymentPlan(method, draft.plan, lote.precio)
  if (!parsed.ok) {
    return parsed
  }

  const receipt: SaleReceipt = {
    emitidoEl,
    lote,
    cliente,
    seller,
    method,
    monto: lote.precio,
    moneda: lote.moneda,
  }
  if (parsed.plan !== undefined) {
    const schedule = buildPaymentSchedule(lote.precio, parsed.plan)
    receipt.plan = {
      montoEntrega: parsed.plan.montoEntrega,
      cantidadCuotas: parsed.plan.cantidadCuotas,
      tasaInteres: parsed.plan.tasaInteres,
      periodicidad: parsed.plan.periodicidad,
      montoFinanciado: schedule.montoFinanciado,
      montoCuota: schedule.montoCuota,
      montoTotal: schedule.montoTotal,
    }
  }

  return { ok: true, receipt, plan: parsed.plan }
}

// What the sale started from the loteo viewer needs of a loteo: enough to
// describe the lote being sold and to build its LoteOption. `app` maps the
// loteo detail into this shape; sales never imports the lots feature.
export type SaleCreateBlock = {
  id: string
  numero: string
  tieneAgua: boolean
  tieneCloaca: boolean
  tieneLuz: boolean
  tieneGas: boolean
}

export type SaleCreateLot = {
  id: string
  manzanaId: string
  numero: string
  estado: LotState
  precio: number | null
  moneda: string
  superficie: number | null
  caracteristicas: string
}

export type SaleCreateDevelopment = {
  id: string
  nombre: string
  ubicacion: string
  descripcion: string
  manzanas: SaleCreateBlock[]
  lotes: SaleCreateLot[]
}

export function loteOptionFromDevelopment(
  loteo: SaleCreateDevelopment,
  loteId: string,
): LoteOption | null {
  const lote = loteo.lotes.find((candidate) => candidate.id === loteId)
  if (lote === undefined) {
    return null
  }
  const manzana = loteo.manzanas.find((candidate) => candidate.id === lote.manzanaId)
  return {
    id: lote.id,
    numero: lote.numero,
    manzanaId: lote.manzanaId,
    manzanaNumero: manzana?.numero ?? '',
    loteoId: loteo.id,
    loteoNombre: loteo.nombre,
    estado: lote.estado,
    precio: lote.precio,
    moneda: lote.moneda,
    superficie: lote.superficie,
  }
}

// Why a lote can't be sold yet from the viewer, or null when it can. Mirrors
// the reserva rule: the receipt needs the number and the price.
export function saleDisabledReason(lote: { numero: string; precio: number | null }): string | null {
  const missing: string[] = []
  if (!lote.numero.trim()) {
    missing.push('el número')
  }
  if (lote.precio === null) {
    missing.push('el precio')
  }
  if (missing.length === 0) {
    return null
  }
  return `Completá ${missing.join(' y ')} del lote para habilitar la venta.`
}

// A venta as GET /api/v1/ventas publishes it. The agency comes from the
// seller's profile, so it is absent for a venta directa by an internal user.
export const SALE_STATES = ['activa', 'completada', 'cancelada'] as const

export type SaleState = (typeof SALE_STATES)[number]

export const SALE_STATE_LABELS: Record<SaleState, string> = {
  activa: 'Activa',
  completada: 'Completada',
  cancelada: 'Cancelada',
}

export function isSaleState(value: unknown): value is SaleState {
  return typeof value === 'string' && (SALE_STATES as readonly string[]).includes(value)
}

export type SaleActor = {
  id: string
  nombre: string
  apellido: string
  rol: string
}

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

export type Installment = {
  id: string
  numero: number
  monto: number
  estado: InstallmentState
  fechaVencimiento: string
  fechaPago?: string
}

// The plan as GET /api/v1/ventas publishes it. The list carries the summary
// and the detail also the cuotas.
export type PaymentPlan = {
  id: string
  montoEntrega: number
  cantidadCuotas: number
  tasaInteres: number
  periodicidad: PaymentPeriod
  moneda: string
  montoFinanciado: number
  montoCuota: number
  montoTotal: number
  cuotas?: Installment[]
}

export type Sale = {
  id: string
  loteoId: string
  loteoNombre: string
  loteId: string
  loteNumero: string
  manzanaNumero: string
  loteSuperficie: number | null
  cliente: ClienteOption
  vendedor: SaleActor
  usuarioAlta: SaleActor
  inmobiliaria?: AgencyOption
  modalidadPago: PaymentMethod
  monto: number
  moneda: string
  planPago?: PaymentPlan
  estado: SaleState
  fechaCreacion: string
  fechaModificacion: string
}

export type SalePage = {
  ventas: Sale[]
  pagina: number
  porPagina: number
  total: number
  paginas: number
}

export type SaleListFilters = {
  estado?: SaleState | ''
  loteoId?: string
  loteId?: string
  q?: string
  pagina?: number
  porPagina?: number
}

export type CreateSaleValues = {
  loteoId: string
  loteId: string
  clienteId: string
  vendedorId: string
  modalidadPago: PaymentMethod
  planPago?: PaymentPlanInput
}

export function saleLoteLabel(sale: Pick<Sale, 'loteoNombre' | 'manzanaNumero' | 'loteNumero'>): string {
  const manzana = sale.manzanaNumero || '—'
  const numero = sale.loteNumero || '—'
  return `${sale.loteoNombre} · Mz ${manzana} · Lote ${numero}`
}

// The receipt of a persisted sale: what the dialog prints once the backend
// has registered the venta.
export function saleReceiptFromSale(sale: Sale): SaleReceipt {
  const plan = sale.planPago
  return {
    plan:
      plan === undefined
        ? undefined
        : {
            montoEntrega: plan.montoEntrega,
            cantidadCuotas: plan.cantidadCuotas,
            tasaInteres: plan.tasaInteres,
            periodicidad: plan.periodicidad,
            montoFinanciado: plan.montoFinanciado,
            montoCuota: plan.montoCuota,
            montoTotal: plan.montoTotal,
          },
    emitidoEl: sale.fechaCreacion,
    lote: {
      id: sale.loteId,
      numero: sale.loteNumero,
      manzanaId: '',
      manzanaNumero: sale.manzanaNumero,
      loteoId: sale.loteoId,
      loteoNombre: sale.loteoNombre,
      estado: 'vendido',
      precio: sale.monto,
      moneda: sale.moneda,
      superficie: sale.loteSuperficie,
    },
    cliente: sale.cliente,
    seller: {
      id: sale.vendedor.id,
      nombre: sale.vendedor.nombre,
      apellido: sale.vendedor.apellido,
      rol: sale.vendedor.rol,
      inmobiliariaId: sale.inmobiliaria?.id,
      inmobiliariaRazonSocial: sale.inmobiliaria?.razonSocial,
    },
    method: sale.modalidadPago,
    monto: sale.monto,
    moneda: sale.moneda,
  }
}
