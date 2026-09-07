export const RESERVATION_STATES = ['activa', 'vencida', 'cancelada', 'convertida'] as const

export type ReservationState = (typeof RESERVATION_STATES)[number]

export type ReservationActor = {
  id: string
  nombre: string
  apellido: string
  email?: string
  rol: string
}

export type ReservationClient = {
  id: string
  nombre: string
  apellido: string
  dni: string
  celular?: string
  email?: string
}

export type ReservationLoteoOption = {
  id: string
  nombre: string
}

export type ReservationLot = {
  id: string
  numero: string
  estado: 'disponible' | 'reservado' | 'vendido' | 'finalizado'
  precio: number | null
}

export type ReservationDraft = {
  clienteId: string
  vendedorId: string
}

export type ReservationHistoryEntry = {
  id: string
  estado: ReservationState
  razon?: string
  usuario?: ReservationActor
  fecha: string
}

export type Reservation = {
  id: string
  loteoId: string
  loteoNombre: string
  loteId: string
  loteNumero: string
  cliente: ReservationClient
  vendedor: ReservationActor
  usuarioAlta: ReservationActor
  estado: ReservationState
  fechaVencimiento: string
  fechaCreacion: string
  fechaModificacion: string
  historial?: ReservationHistoryEntry[]
}

export type SellerOption = ReservationActor

export type ReservationPage = {
  reservas: Reservation[]
  pagina: number
  porPagina: number
  total: number
  paginas: number
}

export type ReservationListFilters = {
  estado?: ReservationState | ''
  loteoId?: string
  loteId?: string
  q?: string
  pagina?: number
  porPagina?: number
}
