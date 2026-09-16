// Types that mirror an API body (`Sale`, `SaleCreateDevelopment`, the
// option types `app` fills with API objects) keep the Spanish property names
// of the JSON contract; everything the feature builds for itself is in English.
export const LOT_STATES = ['disponible', 'reservado', 'vendido', 'finalizado'] as const

export type LotState = (typeof LOT_STATES)[number]

export type LotOption = {
  id: string
  number: string
  blockId: string
  blockNumber: string
  developmentId: string
  developmentName: string
  state: LotState
  price: number | null
  currency: string
  area: number | null
}

export function isLotState(value: unknown): value is LotState {
  return typeof value === 'string' && (LOT_STATES as readonly string[]).includes(value)
}

// A lote is identified by its loteo, manzana and number together: the number
// alone repeats across manzanas and loteos.
export function lotOptionLabel(lot: LotOption): string {
  const block = lot.blockNumber || '—'
  const number = lot.number || '—'
  return `${lot.developmentName} · Mz ${block} · Lote ${number}`
}

// Sales never imports another feature's files, so it states the shape it needs
// from a cliente, an inmobiliaria and a vendedor. The objects `app` injects
// satisfy these structurally.
export type ClientOption = {
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

export function clientOptionLabel(client: ClientOption): string {
  return `${client.apellido}, ${client.nombre} · DNI ${client.dni}`
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
// The backend persists the rate as NUMERIC(8,4) and amounts as NUMERIC(14,2)
// and rejects anything finer, so the form does too.
const INTEREST_RATE_DECIMALS = 4
const MONEY_DECIMALS = 2

// The plan as POST .../ventas receives it (the `planPago` body): keys are the
// JSON contract. tasaInteres is a percentage and montoEntrega is 0 unless the
// modalidad is entrega_financiada.
export type PaymentPlanInput = {
  cantidadCuotas: number
  tasaInteres: number
  periodicidad: PaymentPeriod
  montoEntrega: number
}

// What the form holds while the user types: strings, so a half-typed number
// never snaps to something else under their cursor.
export type PaymentPlanValues = {
  installments: string
  interestRate: string
  period: PaymentPeriod
  downPayment: string
}

export const EMPTY_PAYMENT_PLAN: PaymentPlanValues = {
  installments: '',
  interestRate: '',
  period: 'mensual',
  downPayment: '',
}

export type PaymentSchedule = {
  financedAmount: number
  totalAmount: number
  installmentAmount: number
  installments: number[]
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
//   financed    = round2(amount - downPayment)
//   total       = round2(financed * (1 + interestRate / 100))
//   installment = round2(total / installments)
//   last        = round2(total - installment * (installments - 1))
//
// The last installment absorbs the rounding remainder so they add up to
// total exactly. Due dates are derived by the backend from the sale date.
export function buildPaymentSchedule(amount: number, plan: PaymentPlanInput): PaymentSchedule {
  const financedAmount = roundMoney(amount - plan.montoEntrega)
  const totalAmount = roundMoney(financedAmount * (1 + plan.tasaInteres / 100))
  const installmentAmount = roundMoney(totalAmount / plan.cantidadCuotas)
  const last = roundMoney(totalAmount - installmentAmount * (plan.cantidadCuotas - 1))
  const installments = Array.from({ length: plan.cantidadCuotas }, (_, index) =>
    index === plan.cantidadCuotas - 1 ? last : installmentAmount,
  )
  return { financedAmount, totalAmount, installmentAmount, installments }
}

export function lastInstallmentAmount(schedule: PaymentSchedule): number {
  return schedule.installments[schedule.installments.length - 1]
}

// Accepts "1234.5", "1234,5" and "1.234,50": with a comma present the dots
// are thousands separators, as people type prices here. More decimals than
// `maxDecimals` is a typo, not something to round.
function parseDecimal(value: string, maxDecimals: number): number | null {
  const trimmed = value.trim()
  const normalized = trimmed.includes(',')
    ? trimmed.replace(/\./g, '').replace(',', '.')
    : trimmed
  const match = /^\d+(?:\.(\d+))?$/.exec(normalized)
  if (match === null || (match[1]?.length ?? 0) > maxDecimals) {
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
  amount: number,
): PaymentPlanResult {
  if (!isFinancedMethod(method)) {
    return { ok: true, plan: undefined }
  }

  const installments = Number(values.installments.trim())
  if (
    values.installments.trim() === '' ||
    !Number.isInteger(installments) ||
    installments < 1 ||
    installments > MAX_SALE_INSTALLMENTS
  ) {
    return {
      ok: false,
      error: `Ingresá una cantidad de cuotas entre 1 y ${MAX_SALE_INSTALLMENTS}.`,
    }
  }

  const interestRate =
    values.interestRate.trim() === '' ? 0 : parseDecimal(values.interestRate, INTEREST_RATE_DECIMALS)
  if (interestRate === null || interestRate < 0 || interestRate > MAX_SALE_INTEREST_RATE) {
    return {
      ok: false,
      error: `Ingresá una tasa de interés entre 0 y ${MAX_SALE_INTEREST_RATE} %, con hasta ${INTEREST_RATE_DECIMALS} decimales.`,
    }
  }

  let downPayment = 0
  if (method === 'entrega_financiada') {
    const parsed = parseDecimal(values.downPayment, MONEY_DECIMALS)
    if (parsed === null || parsed <= 0) {
      return { ok: false, error: 'Ingresá el monto de la entrega, con hasta 2 decimales.' }
    }
    if (parsed >= amount) {
      return { ok: false, error: 'La entrega tiene que ser menor al precio del lote.' }
    }
    downPayment = parsed
  }

  const plan: PaymentPlanInput = {
    cantidadCuotas: installments,
    tasaInteres: interestRate,
    periodicidad: values.period,
    montoEntrega: downPayment,
  }
  const schedule = buildPaymentSchedule(amount, plan)
  if (schedule.installmentAmount < 0.01 || lastInstallmentAmount(schedule) < 0.01) {
    return { ok: false, error: 'El monto financiado no alcanza para esa cantidad de cuotas.' }
  }
  return { ok: true, plan }
}

export type SaleableLot = { number: string; price: number | null; currency: string }

// Why a lote can't be sold yet, or null when it can. The backend rejects a
// sale of a lote without number, without a price above zero or without a
// currency (`sale_lot_incomplete`), so the viewer link and the form apply the
// same rule and never offer a sale that can't persist.
export function saleDisabledReason(lot: SaleableLot): string | null {
  const missing: string[] = []
  if (!lot.number.trim()) {
    missing.push('el número')
  }
  if (lot.price === null || lot.price <= 0) {
    missing.push('el precio')
  } else if (!lot.currency.trim()) {
    missing.push('la moneda')
  }
  if (missing.length === 0) {
    return null
  }
  return `Completá ${missing.join(' y ')} del lote para habilitar la venta.`
}

export type SaleDraft = {
  lot: LotOption | null
  client: ClientOption | null
  seller: SellerOption | null
  method: PaymentMethod
  plan: PaymentPlanValues
}

// The plan as the receipt prints it: the typed terms plus the derived
// amounts, all of which the backend also publishes in Sale.planPago. The
// last installment is listed apart because it absorbs the rounding
// remainder and may differ from the regular one.
export type SaleReceiptPlan = {
  downPayment: number
  installments: number
  interestRate: number
  period: PaymentPeriod
  financedAmount: number
  installmentAmount: number
  lastInstallmentAmount: number
  totalAmount: number
}

function saleReceiptPlan(plan: PaymentPlanInput, schedule: PaymentSchedule): SaleReceiptPlan {
  return {
    downPayment: plan.montoEntrega,
    installments: plan.cantidadCuotas,
    interestRate: plan.tasaInteres,
    period: plan.periodicidad,
    financedAmount: schedule.financedAmount,
    installmentAmount: schedule.installmentAmount,
    lastInstallmentAmount: lastInstallmentAmount(schedule),
    totalAmount: schedule.totalAmount,
  }
}

export type SaleReceipt = {
  issuedAt: string
  lot: LotOption
  client: ClientOption
  seller: SellerOption
  method: PaymentMethod
  amount: number
  currency: string
  plan?: SaleReceiptPlan
}

export type SaleReceiptResult =
  | { ok: true; receipt: SaleReceipt; plan: PaymentPlanInput | undefined }
  | { ok: false; error: string }

export function buildSaleReceipt(draft: SaleDraft, issuedAt: string): SaleReceiptResult {
  const { lot, client, seller, method } = draft

  if (lot === null) {
    return { ok: false, error: 'Elegí el lote que se vende.' }
  }

  if (client === null) {
    return { ok: false, error: 'Elegí el cliente comprador.' }
  }

  if (seller === null) {
    return { ok: false, error: 'Elegí el vendedor que realizó la venta.' }
  }

  const disabledReason = saleDisabledReason(lot)
  if (disabledReason !== null) {
    return { ok: false, error: disabledReason }
  }
  const price = lot.price ?? 0

  const parsed = parsePaymentPlan(method, draft.plan, price)
  if (!parsed.ok) {
    return parsed
  }

  const receipt: SaleReceipt = {
    issuedAt,
    lot,
    client,
    seller,
    method,
    amount: price,
    currency: lot.currency,
  }
  if (parsed.plan !== undefined) {
    receipt.plan = saleReceiptPlan(parsed.plan, buildPaymentSchedule(price, parsed.plan))
  }

  return { ok: true, receipt, plan: parsed.plan }
}

// What the sale started from the loteo viewer needs of a loteo: enough to
// describe the lote being sold and to build its LotOption. `app` maps the
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

export function lotOptionFromDevelopment(
  development: SaleCreateDevelopment,
  lotId: string,
): LotOption | null {
  const lot = development.lotes.find((candidate) => candidate.id === lotId)
  if (lot === undefined) {
    return null
  }
  const block = development.manzanas.find((candidate) => candidate.id === lot.manzanaId)
  return {
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
  }
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
  cliente: ClientOption
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

export function saleLotLabel(sale: Pick<Sale, 'loteoNombre' | 'manzanaNumero' | 'loteNumero'>): string {
  const block = sale.manzanaNumero || '—'
  const number = sale.loteNumero || '—'
  return `${sale.loteoNombre} · Mz ${block} · Lote ${number}`
}
// The receipt of a persisted sale: what the dialog prints once the backend
// has registered the venta.
export function saleReceiptFromSale(sale: Sale): SaleReceipt {
  return {
    plan: sale.planPago === undefined ? undefined : persistedReceiptPlan(sale.planPago),
    issuedAt: sale.fechaCreacion,
    lot: {
      id: sale.loteId,
      number: sale.loteNumero,
      blockId: '',
      blockNumber: sale.manzanaNumero,
      developmentId: sale.loteoId,
      developmentName: sale.loteoNombre,
      state: 'vendido',
      price: sale.monto,
      currency: sale.moneda,
      area: sale.loteSuperficie,
    },
    client: sale.cliente,
    seller: {
      id: sale.vendedor.id,
      nombre: sale.vendedor.nombre,
      apellido: sale.vendedor.apellido,
      rol: sale.vendedor.rol,
      inmobiliariaId: sale.inmobiliaria?.id,
      inmobiliariaRazonSocial: sale.inmobiliaria?.razonSocial,
    },
    method: sale.modalidadPago,
    amount: sale.monto,
    currency: sale.moneda,
  }
}

// The last installment comes from the persisted cuotas when the response
// carries them; otherwise it is derived with the same formula the backend
// used, so a list summary prints the same remainder as the detail.
function persistedReceiptPlan(plan: PaymentPlan): SaleReceiptPlan {
  const persistedLast = plan.cuotas?.[plan.cuotas.length - 1]?.monto
  return {
    downPayment: plan.montoEntrega,
    installments: plan.cantidadCuotas,
    interestRate: plan.tasaInteres,
    period: plan.periodicidad,
    financedAmount: plan.montoFinanciado,
    installmentAmount: plan.montoCuota,
    lastInstallmentAmount:
      persistedLast ?? roundMoney(plan.montoTotal - plan.montoCuota * (plan.cantidadCuotas - 1)),
    totalAmount: plan.montoTotal,
  }
}
