import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import BillingPage from './BillingPage'
import type { DueInstallment, DueInstallmentPage } from '../types'

const listDueInstallmentsMock = vi.hoisted(() => vi.fn())
const listDevelopmentOptionsMock = vi.hoisted(() => vi.fn())

vi.mock('../api/billing', () => ({
  listDueInstallments: listDueInstallmentsMock,
  listDevelopmentOptions: listDevelopmentOptionsMock,
}))

const cuota: DueInstallment = {
  id: 'c-1',
  ventaId: 'sale-1',
  numero: 1,
  cantidadCuotas: 3,
  monto: 20000,
  moneda: 'USD',
  estado: 'vencida',
  fechaVencimiento: '2026-04-15T12:00:00Z',
  loteoId: 'loteo-1',
  loteoNombre: 'Las Acacias',
  loteId: 'lot-1',
  loteNumero: '7',
  manzanaNumero: '2',
  cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
  vendedor: { id: 'seller-1', nombre: 'Marta', apellido: 'Suárez', rol: 'inmobiliaria' },
  inmobiliaria: { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
}

const page: DueInstallmentPage = {
  cuotas: [cuota, { ...cuota, id: 'c-2', numero: 2, estado: 'pagada', fechaPago: '2026-07-10T12:00:00Z', inmobiliaria: undefined, vendedor: { id: 'u-2', nombre: 'Carla', apellido: 'López', rol: 'administrativo' } }],
  resumen: { cuotasVencidas: 4, cuotasProximas: 2 },
  pagina: 1,
  porPagina: 25,
  total: 2,
  paginas: 1,
}

function renderPage(token = 'token') {
  return render(
    <MemoryRouter>
      <BillingPage accessToken={token} />
    </MemoryRouter>,
  )
}

beforeEach(() => {
  listDevelopmentOptionsMock.mockResolvedValue([
    { id: 'loteo-1', nombre: 'Las Acacias' },
    { id: 'loteo-2', nombre: 'Los Pinos' },
  ])
})

afterEach(() => vi.clearAllMocks())

describe('BillingPage', () => {
  it('lists the vencimientos with the scope summary', async () => {
    listDueInstallmentsMock.mockResolvedValue(page)

    renderPage()

    expect(screen.getByRole('heading', { name: 'Cobranzas' })).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Cargando cuotas…')
    expect(await screen.findByText('4')).toBeInTheDocument()
    expect(screen.getByText('Cuotas vencidas')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(listDueInstallmentsMock).toHaveBeenCalledWith('token', { estado: 'pendientes', pagina: 1 }, expect.anything())

    expect(screen.getAllByRole('link', { name: 'Pérez, Ana' })).toHaveLength(2)
    expect(screen.getAllByText('Las Acacias · Mz 2 · Lote 7')).toHaveLength(2)
    expect(screen.getByText(/Cuota 1 de 3 · US\$\s?20\.000,00 · Vence 15\/04\/2026/)).toBeInTheDocument()
    expect(screen.getByText(/Cuota 2 de 3 .* Pagada el 10\/07\/2026/)).toBeInTheDocument()
    expect(screen.getByText('Vendedor: Suárez, Marta (Inmobiliaria Sur)')).toBeInTheDocument()
    expect(screen.getByText('Vendedor: López, Carla (Venta directa)')).toBeInTheDocument()
    expect(screen.getByText('Vencida')).toBeInTheDocument()
    expect(screen.getByText('Pagada')).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Ver estado de deuda de Pérez, Ana por Las Acacias · Mz 2 · Lote 7, cuota 1' }),
    ).toHaveAttribute('href', '/cobranzas/sale-1')
  })

  it('applies the filters and resets the page', async () => {
    const user = userEvent.setup()
    listDueInstallmentsMock.mockResolvedValue({ ...page, paginas: 3, total: 60 })

    renderPage()
    await screen.findByText('Vencida')

    await user.click(screen.getByRole('button', { name: 'Siguiente' }))
    await waitFor(() =>
      expect(listDueInstallmentsMock).toHaveBeenLastCalledWith('token', { estado: 'pendientes', pagina: 2 }, expect.anything()),
    )

    await user.click(screen.getByRole('combobox', { name: 'Estado' }))
    await user.click(await screen.findByRole('option', { name: 'Vencidas' }))
    await waitFor(() =>
      expect(listDueInstallmentsMock).toHaveBeenLastCalledWith('token', { estado: 'vencida', pagina: 1 }, expect.anything()),
    )

    await user.type(screen.getByLabelText('Buscar'), 'Ana')
    await waitFor(() =>
      expect(listDueInstallmentsMock).toHaveBeenLastCalledWith('token', { q: 'Ana', estado: 'vencida', pagina: 1 }, expect.anything()),
    )

    await user.type(screen.getByLabelText('Vence desde'), '2026-04-01')
    await user.type(screen.getByLabelText('Vence hasta'), '2026-04-30')
    await waitFor(() =>
      expect(listDueInstallmentsMock).toHaveBeenLastCalledWith(
        'token',
        { q: 'Ana', estado: 'vencida', desde: '2026-04-01', hasta: '2026-04-30', pagina: 1 },
        expect.anything(),
      ),
    )
  }, 15000)

  it('filters the vencimientos by loteo', async () => {
    const user = userEvent.setup()
    listDueInstallmentsMock.mockResolvedValue({ ...page, paginas: 3, total: 60 })

    renderPage()
    await screen.findByText('Vencida')
    expect(listDevelopmentOptionsMock).toHaveBeenCalledWith('token', expect.anything())
    await user.click(screen.getByRole('button', { name: 'Siguiente' }))

    const loteo = screen.getByRole('combobox', { name: 'Loteo' })
    expect(loteo).toHaveTextContent('Todos los loteos')
    await user.click(loteo)
    await user.click(await screen.findByRole('option', { name: 'Los Pinos' }))
    await waitFor(() =>
      expect(listDueInstallmentsMock).toHaveBeenLastCalledWith(
        'token',
        { estado: 'pendientes', loteoId: 'loteo-2', pagina: 1 },
        expect.anything(),
      ),
    )
    expect(screen.getByRole('combobox', { name: 'Loteo' })).toHaveTextContent('Los Pinos')

    await user.click(screen.getByRole('combobox', { name: 'Loteo' }))
    await user.click(await screen.findByRole('option', { name: 'Todos los loteos' }))
    await waitFor(() =>
      expect(listDueInstallmentsMock).toHaveBeenLastCalledWith('token', { estado: 'pendientes', pagina: 1 }, expect.anything()),
    )
  }, 15000)

  it('still lists the vencimientos when the loteos cannot be loaded', async () => {
    listDevelopmentOptionsMock.mockRejectedValue(new ApiError('Sin permiso', 'forbidden', 403))
    listDueInstallmentsMock.mockResolvedValue(page)
    const user = userEvent.setup()

    renderPage()
    expect(await screen.findByText('Vencida')).toBeInTheDocument()
    await user.click(screen.getByRole('combobox', { name: 'Loteo' }))
    expect(await screen.findByRole('option', { name: 'Todos los loteos' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Los Pinos' })).not.toBeInTheDocument()
  })

  it('shows the empty state and the API error', async () => {
    listDueInstallmentsMock.mockResolvedValueOnce({ ...page, cuotas: [], total: 0, paginas: 0 })

    const { unmount } = renderPage()
    expect(await screen.findByText('No hay cuotas que coincidan con los filtros.')).toBeInTheDocument()
    unmount()

    listDueInstallmentsMock.mockRejectedValueOnce(new ApiError('Sin permiso', 'forbidden', 403))
    renderPage()
    expect(await screen.findByText('Sin permiso')).toBeInTheDocument()
  })

  it('waits for a token before asking the API', () => {
    renderPage('')

    expect(listDueInstallmentsMock).not.toHaveBeenCalled()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(screen.getByText('No hay cuotas que coincidan con los filtros.')).toBeInTheDocument()
  })
})
