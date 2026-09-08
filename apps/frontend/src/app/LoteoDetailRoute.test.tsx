import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoteoDetailRoute from './LoteoDetailRoute'
import type { LoteoLote } from '../features/lots/types'
import type { Reservation } from '../features/reservations/types'

const useAuthMock = vi.hoisted(() => vi.fn())
const useReservationsMock = vi.hoisted(() => vi.fn())
const useReservationMutationsMock = vi.hoisted(() => vi.fn())

vi.mock('../features/auth/hooks/use-auth', () => ({ useAuth: useAuthMock }))
vi.mock('../features/reservations/hooks/use-reservations', () => ({ useReservations: useReservationsMock }))
vi.mock('../features/reservations/hooks/use-reservation-mutations', () => ({ useReservationMutations: useReservationMutationsMock }))
vi.mock('../features/reservations/components/ReserveLotDialog', () => ({
  default: ({ onCreated }: { onCreated: () => void }) => (
    <button type="button" onClick={onCreated}>Reservar lote</button>
  ),
}))
vi.mock('../features/lots/pages/LoteoDetailPage', () => ({
  default: ({ renderReservationAction, renderReservationCancelAction }: {
    renderReservationAction?: (lote: LoteoLote, onCreated: () => void) => ReactNode
    renderReservationCancelAction?: (lote: LoteoLote, onCanceled: () => void) => ReactNode
  }) => {
    const availableLot: LoteoLote = {
      id: 'lot-available', manzanaId: 'block-1', numero: '8', estado: 'disponible', precio: null,
      moneda: 'USD', superficie: null, caracteristicas: '', poligono: [],
    }
    const reservedLot: LoteoLote = {
      id: 'lot-1', manzanaId: 'block-1', numero: '7', estado: 'reservado', precio: null,
      moneda: 'USD', superficie: null, caracteristicas: '', poligono: [],
    }
    return <div>
      {renderReservationAction?.(availableLot, vi.fn())}
      {renderReservationCancelAction?.(reservedLot, vi.fn())}
    </div>
  },
}))

const reservation: Reservation = {
  id: 'reservation-1', loteoId: 'loteo-1', loteoNombre: 'Las Acacias', loteId: 'lot-1', loteNumero: '7',
  cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
  vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
  usuarioAlta: { id: 'actor-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
  estado: 'activa', fechaVencimiento: '2026-09-21T12:00:00Z', fechaCreacion: '2026-09-06T12:00:00Z', fechaModificacion: '2026-09-06T12:00:00Z', historial: [],
}

function renderRoute() {
  return render(
    <MemoryRouter initialEntries={['/lotes/loteo-1']}>
      <Routes>
        <Route path="/lotes/:loteoId" element={<LoteoDetailRoute />} />
      </Routes>
    </MemoryRouter>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  useReservationsMock.mockReturnValue({
    page: { reservas: [reservation], pagina: 1, porPagina: 100, total: 1, paginas: 1 },
    isLoading: false,
    error: null,
    refresh: vi.fn(),
    prepend: vi.fn(),
  })
  useReservationMutationsMock.mockReturnValue({ cancel: vi.fn(), isSubmitting: false, error: null, reset: vi.fn() })
})

describe('LoteoDetailRoute', () => {
  it.each(['administrador', 'administrativo', 'inmobiliaria'])('shows cancellation for %s', (role) => {
    useAuthMock.mockReturnValue({ session: { access_token: 'token' }, user: { app_metadata: { role } } })

    renderRoute()

    expect(screen.getByRole('button', { name: 'Cancelar reserva' })).toBeInTheDocument()
    expect(useReservationsMock).toHaveBeenCalledWith(
      'token',
      { loteoId: 'loteo-1', loteId: 'lot-1', estado: 'activa', porPagina: 1 },
      { enabled: true },
    )
  })

  it('creates a reservation from the visor', async () => {
    useAuthMock.mockReturnValue({ session: { access_token: 'token' }, user: { app_metadata: { role: 'administrador' } } })

    renderRoute()

    await screen.getByRole('button', { name: 'Reservar lote' }).click()

    expect(screen.getByRole('button', { name: 'Reservar lote' })).toBeInTheDocument()
  })

  it.each(['agrimensor', 'escribano'])('hides cancellation for %s', (role) => {
    useAuthMock.mockReturnValue({ session: { access_token: 'token' }, user: { app_metadata: { role } } })

    renderRoute()

    expect(screen.queryByRole('button', { name: 'Cancelar reserva' })).not.toBeInTheDocument()
    expect(useReservationsMock).not.toHaveBeenCalled()
  })
})
