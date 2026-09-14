import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import SaleCreateRoute from './SaleCreateRoute'
import type { LoteoDetail } from '../features/lots/types'

const useLoteoMock = vi.hoisted(() => vi.fn())
const listClientsMock = vi.hoisted(() => vi.fn())
const createClientMock = vi.hoisted(() => vi.fn())
const listEligibleSellersMock = vi.hoisted(() => vi.fn())

vi.mock('../features/auth/hooks/use-auth', () => ({
  useAuth: () => ({ session: { access_token: 'token' }, user: { app_metadata: { role: 'administrativo' } } }),
}))
vi.mock('../features/lots/hooks/use-loteo', () => ({ useLoteo: useLoteoMock }))
vi.mock('../features/clients/api/clients', () => ({ listClients: listClientsMock, createClient: createClientMock }))
vi.mock('../features/reservations/api/reservations', () => ({ listEligibleSellers: listEligibleSellersMock }))

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
const seller = { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrativo', esActor: true }

function renderRoute() {
  return render(
    <MemoryRouter initialEntries={['/ventas/nueva/loteo-1/lot-1']}>
      <Routes>
        <Route path="/ventas/nueva/:loteoId/:loteId" element={<SaleCreateRoute />} />
      </Routes>
    </MemoryRouter>,
  )
}

afterEach(() => vi.clearAllMocks())

describe('SaleCreateRoute', () => {
  it('composes the loteo, the clients and the sale-scoped sellers into the sale page', async () => {
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo })
    listClientsMock.mockResolvedValue([client])
    listEligibleSellersMock.mockResolvedValue([seller])
    const user = userEvent.setup()
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})

    renderRoute()

    const plan = within(screen.getByRole('img', { name: 'Plano del loteo' }))
    expect(plan.getByLabelText('Lote 7')).toHaveAttribute('aria-description', 'Disponible')
    expect(screen.getByText('Las Acacias')).toBeInTheDocument()
    expect(screen.getByText('Frente norte')).toBeInTheDocument()

    await waitFor(() => expect(listEligibleSellersMock).toHaveBeenCalledWith('token', 'loteo-1', expect.anything(), 'agencia'))
    await waitFor(() => expect(screen.getByLabelText('Vendedor')).toHaveValue('Gómez, Beto'))
    expect(screen.getByLabelText('Inmobiliaria')).toHaveValue('Venta directa')

    await user.click(screen.getByLabelText('Cliente'))
    await user.click(await screen.findByRole('option', { name: /Pérez, Ana/ }))
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('Las Acacias · Mz 2 · Lote 7')
    expect(dialog).toHaveTextContent('Gómez, Beto')
    print.mockRestore()
  })

  it('registers a new client through the clients API and selects it', async () => {
    useLoteoMock.mockReturnValue({ status: 'loaded', loteo })
    listClientsMock.mockResolvedValue([])
    listEligibleSellersMock.mockResolvedValue([seller])
    createClientMock.mockResolvedValue(client)
    const user = userEvent.setup()

    renderRoute()

    await user.click(screen.getByRole('button', { name: 'Registrar cliente' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText('Nombre'), client.nombre)
    await user.type(within(dialog).getByLabelText('Apellido'), client.apellido)
    await user.type(within(dialog).getByLabelText('DNI'), client.dni)
    await user.click(within(dialog).getByRole('button', { name: 'Guardar cliente' }))

    await waitFor(() => expect(createClientMock).toHaveBeenCalledWith('token', expect.objectContaining({ dni: client.dni })))
    await waitFor(() => expect(screen.getByLabelText('Cliente')).toHaveValue('Pérez, Ana · DNI 30111222'))
  })

  it('shows the loteo loading state', () => {
    useLoteoMock.mockReturnValue({ status: 'loading' })
    listClientsMock.mockResolvedValue([])

    renderRoute()

    expect(screen.getByRole('status', { name: 'Cargando los datos para registrar la venta…' })).toBeInTheDocument()
  })
})
