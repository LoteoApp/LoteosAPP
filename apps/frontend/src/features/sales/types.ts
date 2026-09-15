// Property names of the option and wire types below are the JSON contract of
// the API, so they keep the Spanish it publishes; the symbols are in English.
export const LOT_STATES = ['disponible', 'reservado', 'vendido', 'finalizado'] as const

export type LotState = (typeof LOT_STATES)[number]

export type LotOption = {
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
export function lotOptionLabel(lot: LotOption): string {
  const block = lot.manzanaNumero || '—'
  const number = lot.numero || '—'
  return `${lot.loteoNombre} · Mz ${block} · Lote ${number}`
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

// The three modalidades of the ventas.modalidad_pago contract. Only `contado`
// is implemented; the other two are listed so the selector is already in
// place for the feature that adds them.
export const PAYMENT_METHODS = ['contado', 'financiado', 'entrega_financiada'] as const

export type PaymentMethod = (typeof PAYMENT_METHODS)[number]

export const PAYMENT_METHOD_LABELS: Record<PaymentMethod, string> = {
  contado: 'Contado',
  financiado: 'Financiado',
  entrega_financiada: 'Entrega + financiación',
}

const AVAILABLE_PAYMENT_METHODS: readonly PaymentMethod[] = ['contado']

export function isPaymentMethodAvailable(method: PaymentMethod): boolean {
  return AVAILABLE_PAYMENT_METHODS.includes(method)
}

export type SaleableLot = { numero: string; precio: number | null; moneda: string }

// Why a lote can't be sold yet, or null when it can. The backend rejects a
// sale of a lote without number, without a price above zero or without a
// currency (`sale_lot_incomplete`), so the viewer link and the form apply the
// same rule and never offer a sale that can't persist.
export function saleDisabledReason(lot: SaleableLot): string | null {
  const missing: string[] = []
  if (!lot.numero.trim()) {
    missing.push('el número')
  }
  if (lot.precio === null || lot.precio <= 0) {
    missing.push('el precio')
  } else if (!lot.moneda.trim()) {
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
}

export type SaleReceipt = {
  issuedAt: string
  lot: LotOption
  client: ClientOption
  seller: SellerOption
  method: PaymentMethod
  amount: number
  currency: string
}

export type SaleReceiptResult =
  | { ok: true; receipt: SaleReceipt }
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

  if (!isPaymentMethodAvailable(method)) {
    return { ok: false, error: 'Por ahora solo se puede registrar una venta al contado.' }
  }

  const disabledReason = saleDisabledReason(lot)
  if (disabledReason !== null) {
    return { ok: false, error: disabledReason }
  }

  return {
    ok: true,
    receipt: {
      issuedAt,
      lot,
      client,
      seller,
      method,
      amount: lot.precio ?? 0,
      currency: lot.moneda,
    },
  }
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
    numero: lot.numero,
    manzanaId: lot.manzanaId,
    manzanaNumero: block?.numero ?? '',
    loteoId: development.id,
    loteoNombre: development.nombre,
    estado: lot.estado,
    precio: lot.precio,
    moneda: lot.moneda,
    superficie: lot.superficie,
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
    issuedAt: sale.fechaCreacion,
    lot: {
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
