import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import ReservationCreateRoute from './ReservationCreateRoute'
import type { LoteoDetail } from '../features/lots/types'
import type { Reservation } from '../features/reservations/types'

const authState = vi.hoisted(() => ({ role: 'administrativo' }))
const useLoteoMock = vi.hoisted(() => vi.fn())
const useClientsMock = vi.hoisted(() => vi.fn())
const useEligibleSellersMock = vi.hoisted(() => vi.fn())
const createReservationMock = vi.hoisted(() => vi.fn())
const createClientMock = vi.hoisted(() => vi.fn())
const downloadReceiptMock = vi.hoisted(() => vi.fn())

vi.mock('../features/auth/hooks/use-auth', () => ({
  useAuth: () => ({ session: { access_token: 'token' }, user: { app_metadata: { role: authState.role } } }),
}))
vi.mock('../features/lots/hooks/use-loteo', () => ({ useLoteo: useLoteoMock }))
vi.mock('../features/clients/hooks/use-clients', () => ({ useClients: useClientsMock }))
vi.mock('../features/clients/api/clients', () => ({ createClient: createClientMock }))
vi.mock('../features/reservations/hooks/use-eligible-sellers', () => ({ useEligibleSellers: useEligibleSellersMock }))
vi.mock('../features/reservations/hooks/use-reservation-mutations', () => ({
  useReservationMutations: () => ({ create: createReservationMock, isSubmitting: false, error: null }),
}))
vi.mock('../features/reservations/api/reservations', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../features/reservations/api/reservations')>()),
  downloadReservationReceipt: downloadReceiptMock,
}))

const triangle = [{ x: 0, y: 0 }, { x: 10, y: 0 }, { x: 10, y: 10 }]
const loteo: LoteoDetail = {
  id: 'loteo-1', nombre: 'Las Acacias', ubicacion: 'Córdoba', descripcion: 'A metros de la ruta.',
  contorno: triangle,
  manzanas: [{ id: 'block-1', numero: '2', tieneAgua: true, tieneCloaca: false, tieneLuz: true, tieneGas: false, calleIds: [], poligono: triangle }],
  lotes: [
    { id: 'lot-1', manzanaId: 'block-1', numero: '7', estado: 'disponible', precio: 120000, moneda: 'USD', superficie: 300, caracteristicas: 'Frente norte', poligono: triangle },
    { id: 'lot-2', manzanaId: 'block-1', numero: '8', estado: 'reservado', precio: 130000, moneda: 'USD', superficie: 310, caracteristicas: '', poligono: triangle },
  ],
  calles: [], fechaCreacion: '2026-08-20T12:00:00Z',
}
const client = { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222', celular: '', email: '' }
const seller = { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', email: 'beto@example.test', rol: 'administrador' }
const reservation: Reservation = {
  id: 'reservation-1', loteoId: loteo.id, loteoNombre: loteo.nombre, loteId: 'lot-1', loteNumero: '7',
  cliente: { id: client.id, nombre: client.nombre, apellido: client.apellido, dni: client.dni },
  vendedor: seller, usuarioAlta: seller, estado: 'activa', fechaVencimiento: '2026-09-21T12:00:00Z',
  fechaCreacion: '2026-09-06T12:00:00Z', fechaModificacion: '2026-09-06T12:00:00Z',
}

function renderRoute(role = 'administrativo') {
  authState.role = role
  return render(
    <MemoryRouter initialEntries={['/reservas/nueva/loteo-1/lot-1']}>
      <Routes>
        <Route path="/reservas/nueva/:loteoId/:loteId" element={<ReservationCreateRoute />} />
      </Routes>
    </MemoryRouter>,
  )
}

afterEach(() => vi.clearAllMocks())

describe('ReservationCreateRoute', () => {
  it('composes loteo and client data, adds a new client to the reservation form, and requires the admin seller', async () => {
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo })
    useClientsMock.mockReturnValue({ clientes: [], isLoading: false, error: null })
    useEligibleSellersMock.mockReturnValue({ sellers: [seller], isLoading: false, error: null })
    createClientMock.mockResolvedValue(client)
    createReservationMock.mockResolvedValue(reservation)
    downloadReceiptMock.mockRejectedValue(new Error('PDF no disponible'))

    const user = userEvent.setup()
    renderRoute()

    const plan = within(screen.getByRole('img', { name: 'Plano del loteo' }))
    expect(plan.getByLabelText('Lote 7')).toHaveAttribute('aria-description', 'Disponible')
    expect(plan.getByText('7')).toBeInTheDocument()
    expect(plan.getByLabelText('Lote 8')).not.toHaveAttribute('aria-description')
    expect(plan.queryByText('8')).not.toBeInTheDocument()
    expect(screen.getByText('Las Acacias')).toBeInTheDocument()
    expect(screen.getByText('Frente norte')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await user.type(screen.getByLabelText('Nombre'), client.nombre)
    await user.type(screen.getByLabelText('Apellido'), client.apellido)
    await user.type(screen.getByLabelText('DNI'), client.dni)
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))
    await waitFor(() => expect(createClientMock).toHaveBeenCalled())

    await user.click(screen.getByRole('combobox', { name: 'Cliente' }))
    await user.click(await screen.findByRole('option', { name: /Pérez, Ana/ }))
    await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))
    expect(screen.getByText('Seleccioná un vendedor.')).toBeInTheDocument()

    await user.click(screen.getByRole('combobox', { name: 'Vendedor' }))
    await user.click(await screen.findByRole('option', { name: /Gómez, Beto/ }))
    await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))
    await waitFor(() => expect(createReservationMock).toHaveBeenCalledWith(
      { loteoId: loteo.id, loteId: 'lot-1', clienteId: client.id, vendedorId: seller.id },
      expect.any(String),
    ))
    expect(await screen.findByText('Reserva creada')).toBeInTheDocument()
  })

  it('uses the seller returned for an agency user without showing a selector', () => {
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo })
    useClientsMock.mockReturnValue({ clientes: [client], isLoading: false, error: null })
    useEligibleSellersMock.mockReturnValue({ sellers: [seller], isLoading: false, error: null })
    createReservationMock.mockResolvedValue(reservation)
    downloadReceiptMock.mockRejectedValue(new Error('PDF no disponible'))

    renderRoute('inmobiliaria')

    expect(screen.getByText('Gómez, Beto')).toBeInTheDocument()
    expect(screen.queryByRole('combobox', { name: 'Vendedor' })).not.toBeInTheDocument()
  })
})
