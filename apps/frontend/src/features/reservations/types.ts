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

export type ReservationCreateLot = ReservationLot & {
  manzanaId: string
  moneda: string
  superficie: number | null
  caracteristicas: string
}

export type ReservationCreateBlock = {
  id: string
  numero: string
  tieneAgua: boolean
  tieneCloaca: boolean
  tieneLuz: boolean
  tieneGas: boolean
}

export type ReservationCreateDevelopment = {
  id: string
  nombre: string
  ubicacion: string
  descripcion: string
  manzanas: ReservationCreateBlock[]
  lotes: ReservationCreateLot[]
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
  inmobiliaria?: {
    id: string
    razonSocial: string
  }
  puedeCancelar?: boolean
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
