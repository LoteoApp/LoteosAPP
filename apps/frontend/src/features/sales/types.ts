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
// from a cliente and an inmobiliaria. The objects `app` injects satisfy these
// structurally.
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
