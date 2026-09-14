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

export type SaleDraft = {
  lote: LoteOption | null
  cliente: ClienteOption | null
  seller: SellerOption | null
  method: PaymentMethod
}

export type SaleReceipt = {
  emitidoEl: string
  lote: LoteOption
  cliente: ClienteOption
  seller: SellerOption
  method: PaymentMethod
  monto: number
  moneda: string
}

export type SaleReceiptResult =
  | { ok: true; receipt: SaleReceipt }
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

  if (!isPaymentMethodAvailable(method)) {
    return { ok: false, error: 'Por ahora solo se puede registrar una venta al contado.' }
  }

  if (lote.precio === null) {
    return {
      ok: false,
      error: 'El lote no tiene precio cargado. Cargalo en el detalle del loteo antes de vender.',
    }
  }

  return {
    ok: true,
    receipt: {
      emitidoEl,
      lote,
      cliente,
      seller,
      method,
      monto: lote.precio,
      moneda: lote.moneda,
    },
  }
}
