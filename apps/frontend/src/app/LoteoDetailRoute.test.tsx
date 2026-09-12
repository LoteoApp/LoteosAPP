import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoteoDetailRoute from './LoteoDetailRoute'
import type { LoteoDetail, LoteoLote } from '../features/lots/types'
import type { Reservation } from '../features/reservations/types'

const useAuthMock = vi.hoisted(() => vi.fn())
const useReservationsMock = vi.hoisted(() => vi.fn())
const useReservationMutationsMock = vi.hoisted(() => vi.fn())
const freeLotMock = vi.hoisted(() => vi.fn())

vi.mock('../features/auth/hooks/use-auth', () => ({ useAuth: useAuthMock }))
vi.mock('../features/reservations/hooks/use-reservations', () => ({ useReservations: useReservationsMock }))
vi.mock('../features/reservations/hooks/use-reservation-mutations', () => ({ useReservationMutations: useReservationMutationsMock }))
vi.mock('../features/reservations/components/ReserveLotDialog', () => ({
  default: ({ onCreated }: { onCreated: () => void }) => (
    <button type="button" onClick={onCreated}>Reservar lote</button>
  ),
}))
vi.mock('../features/lots/pages/LoteoDetailPage', () => ({
  default: ({ renderReservationAction, renderReservationSummary, renderReservations }: {
    renderReservationAction?: (lote: LoteoLote, onCreated: () => void) => ReactNode
    renderReservationSummary?: (lote: LoteoLote, onCanceled: () => void) => ReactNode
    renderReservations?: (loteo: LoteoDetail, onCanceled: (loteId: string) => void) => ReactNode
  }) => {
    const availableLot: LoteoLote = {
      id: 'lot-available', manzanaId: 'block-1', numero: '8', estado: 'disponible', precio: null,
      moneda: 'USD', superficie: null, caracteristicas: '', poligono: [],
    }
    const reservedLot: LoteoLote = {
      id: 'lot-1', manzanaId: 'block-1', numero: '7', estado: 'reservado', precio: null,
      moneda: 'USD', superficie: null, caracteristicas: '', poligono: [],
    }
    const loteo: LoteoDetail = {
      id: 'loteo-1', nombre: 'Las Acacias', ubicacion: '', descripcion: '', contorno: [],
      manzanas: [], lotes: [availableLot, reservedLot], calles: [], fechaCreacion: '2026-08-20T12:00:00Z',
    }
    return <div>
      {renderReservationAction?.(availableLot, vi.fn())}
      {renderReservationSummary?.(reservedLot, vi.fn())}
      {renderReservations?.(loteo, freeLotMock)}
    </div>
  },
}))

const reservation: Reservation = {
  id: 'reservation-1', loteoId: 'loteo-1', loteoNombre: 'Las Acacias', loteId: 'lot-1', loteNumero: '7',
  cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
  vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
  usuarioAlta: { id: 'actor-1', nombre: 'Beto', apellido: 'Gómez', rol: 'inmobiliaria' },
  estado: 'activa', fechaVencimiento: '2026-09-21T12:00:00Z', fechaCreacion: '2026-09-06T12:00:00Z', fechaModificacion: '2026-09-06T12:00:00Z', historial: [], puedeCancelar: true,
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
    page: { reservas: [reservation], pagina: 1, porPagina: 25, total: 1, paginas: 1 },
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

  it('hides cancellation when the server denies permission', () => {
    useAuthMock.mockReturnValue({ session: { access_token: 'token' }, user: { app_metadata: { role: 'inmobiliaria' } } })
    useReservationsMock.mockReturnValue({
      page: { reservas: [{ ...reservation, puedeCancelar: false }], pagina: 1, porPagina: 25, total: 1, paginas: 1 },
      isLoading: false,
      error: null,
      refresh: vi.fn(),
      prepend: vi.fn(),
    })

    renderRoute()

    expect(screen.queryByRole('button', { name: 'Cancelar reserva' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Ver reserva' })).toBeInTheDocument()
  })

  it('creates a reservation from the visor', async () => {
    useAuthMock.mockReturnValue({ session: { access_token: 'token' }, user: { app_metadata: { role: 'administrador' } } })

    renderRoute()

    await screen.getByRole('button', { name: 'Reservar lote' }).click()

    expect(screen.getByRole('button', { name: 'Reservar lote' })).toBeInTheDocument()
  })

  it('lists the reservations of the loteo and cancels one from the list', async () => {
    const user = userEvent.setup()
    const cancel = vi.fn().mockResolvedValue(true)
    useReservationMutationsMock.mockReturnValue({ cancel, isSubmitting: false, error: null, reset: vi.fn() })
    useAuthMock.mockReturnValue({ session: { access_token: 'token' }, user: { app_metadata: { role: 'administrador' } } })

    renderRoute()

    expect(useReservationsMock).toHaveBeenCalledWith(
      'token',
      { loteoId: 'loteo-1', estado: 'activa', pagina: 1 },
      { enabled: true },
    )
    expect(screen.getByRole('link', { name: 'Las Acacias' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    const dialog = within(screen.getByRole('dialog'))
    await user.type(
      dialog.getByLabelText('Justificación de la cancelación'),
      'El cliente se arrepintió',
    )
    await user.click(dialog.getByRole('button', { name: 'Cancelar reserva' }))

    expect(cancel).toHaveBeenCalledWith('reservation-1', 'El cliente se arrepintió')
    expect(freeLotMock).toHaveBeenCalledWith('lot-1')
  })

  it.each(['agrimensor', 'escribano'])('does not request reservation data for %s', (role) => {
    useAuthMock.mockReturnValue({ session: { access_token: 'token' }, user: { app_metadata: { role } } })

    renderRoute()

    expect(screen.queryByText(/Reservado por Ana Pérez/)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Cancelar reserva' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Reservar lote' })).not.toBeInTheDocument()
    expect(useReservationsMock).not.toHaveBeenCalled()
  })

  it('paginates active reservations without truncating the loteo history', async () => {
    const user = userEvent.setup()
    useAuthMock.mockReturnValue({ session: { access_token: 'token' }, user: { app_metadata: { role: 'administrador' } } })
    useReservationsMock.mockImplementation((_token, filters) => ({
      page: filters.loteId
        ? { reservas: [reservation], pagina: 1, porPagina: 1, total: 1, paginas: 1 }
        : { reservas: [reservation], pagina: filters.pagina ?? 1, porPagina: 25, total: 51, paginas: 3 },
      isLoading: false,
      error: null,
      refresh: vi.fn(),
      prepend: vi.fn(),
    }))

    renderRoute()

    expect(screen.getByText('Página 1 de 3 · 51 reservas')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Siguiente' }))

    expect(useReservationsMock).toHaveBeenCalledWith(
      'token',
      { loteoId: 'loteo-1', estado: 'activa', pagina: 2 },
      { enabled: true },
    )
  })
})
