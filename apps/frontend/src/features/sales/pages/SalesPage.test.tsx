import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SalesPage from './SalesPage'
import type { Sale, SalePage } from '../types'

const useSalesMock = vi.hoisted(() => vi.fn())

vi.mock('../hooks/use-sales', () => ({ useSales: useSalesMock }))

function sale(overrides: Partial<Sale> = {}): Sale {
  return {
    id: 'sale-1',
    loteoId: 'loteo-1',
    loteoNombre: 'Las Acacias',
    loteId: 'lot-1',
    loteNumero: '7',
    manzanaNumero: '2',
    loteSuperficie: 300,
    cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
    vendedor: { id: 'seller-1', nombre: 'Marta', apellido: 'Suárez', rol: 'inmobiliaria' },
    usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
    inmobiliaria: { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
    modalidadPago: 'contado',
    monto: 120000,
    moneda: 'USD',
    estado: 'activa',
    fechaCreacion: '2026-09-14T15:00:00Z',
    fechaModificacion: '2026-09-14T15:00:00Z',
    ...overrides,
  }
}

const emptyPage: SalePage = { ventas: [], pagina: 1, porPagina: 25, total: 0, paginas: 0 }

function mockSales(page: SalePage = emptyPage, state: { isLoading?: boolean; error?: string | null } = {}) {
  const refresh = vi.fn()
  useSalesMock.mockReturnValue({ page, isLoading: state.isLoading ?? false, error: state.error ?? null, refresh })
  return { refresh }
}

function renderPage() {
  return render(
    <MemoryRouter>
      <SalesPage accessToken="token" />
    </MemoryRouter>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('SalesPage', () => {
  it('directs sale creation to the lot visor and lists nothing yet', () => {
    mockSales()
    renderPage()

    expect(screen.getByRole('heading', { name: 'Ventas' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Abrir visor de lotes' })).toHaveAttribute('href', '/lotes')
    expect(screen.getByText('No hay ventas que coincidan con los filtros.')).toBeInTheDocument()
    expect(screen.queryByLabelText('Lote')).not.toBeInTheDocument()
  })

  it('lists the sales with their lote, comprador, monto and vendedor', () => {
    mockSales({
      ventas: [sale(), sale({ id: 'sale-2', loteNumero: '8', inmobiliaria: undefined, vendedor: { id: 'us-3', nombre: 'Sofía', apellido: 'Luna', rol: 'administrativo' }, cliente: { id: 'client-2', nombre: 'Luis', apellido: 'Gómez', dni: '28999111' } })],
      pagina: 1,
      porPagina: 25,
      total: 2,
      paginas: 1,
    })
    renderPage()

    expect(screen.getByText('Mz 2 · Lote 7 · Pérez, Ana')).toBeInTheDocument()
    expect(screen.getByText(/Suárez, Marta \(Inmobiliaria Sur\)/)).toBeInTheDocument()
    expect(screen.getByText(/Luna, Sofía \(Venta directa\)/)).toBeInTheDocument()
    expect(screen.getAllByText('Activa')).toHaveLength(2)
    expect(screen.getByRole('link', { name: 'Ver detalle de la venta del lote 7 en Las Acacias' })).toHaveAttribute('href', '/ventas/sale-1')
    expect(screen.getByRole('link', { name: 'Ver detalle de la venta del lote 8 en Las Acacias' })).toHaveAttribute('href', '/ventas/sale-2')
    expect(screen.getAllByRole('link', { name: 'Ver loteo Las Acacias' })[0]).toHaveAttribute('href', '/lotes/loteo-1')
  })

  it('passes the search and the state filter to the hook and resets the page', async () => {
    const user = userEvent.setup()
    mockSales({ ventas: [sale()], pagina: 1, porPagina: 1, total: 2, paginas: 2 })
    renderPage()

    await user.click(screen.getByRole('button', { name: 'Siguiente' }))
    expect(useSalesMock).toHaveBeenLastCalledWith('token', { q: undefined, estado: undefined, pagina: 2 })

    await user.type(screen.getByLabelText('Buscar'), 'Ana')
    expect(useSalesMock).toHaveBeenLastCalledWith('token', { q: 'Ana', estado: undefined, pagina: 1 })

    await user.click(screen.getByRole('combobox', { name: 'Estado' }))
    await user.click(await screen.findByRole('option', { name: 'Canceladas' }))
    expect(useSalesMock).toHaveBeenLastCalledWith('token', { q: 'Ana', estado: 'cancelada', pagina: 1 })
  })

  it('describes a cancelled venta directa of a lote without numbers', () => {
    mockSales({
      ventas: [sale({ estado: 'cancelada', loteNumero: '', manzanaNumero: '', inmobiliaria: undefined })],
      pagina: 1,
      porPagina: 25,
      total: 1,
      paginas: 1,
    })
    renderPage()

    expect(screen.getByText('Cancelada')).toBeInTheDocument()
    expect(screen.getByText('Lote sin número · Pérez, Ana')).toBeInTheDocument()
    expect(screen.getByText(/Venta directa/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Ver detalle de la venta del lote sin número en Las Acacias' })).toHaveAttribute('href', '/ventas/sale-1')
  })

  it('goes back a page', async () => {
    const user = userEvent.setup()
    mockSales({ ventas: [sale()], pagina: 2, porPagina: 1, total: 2, paginas: 2 })
    renderPage()

    expect(screen.getByText('Página 2 de 2 · 2 ventas')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Siguiente' })).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Anterior' }))
    expect(useSalesMock).toHaveBeenLastCalledWith('token', { q: undefined, estado: undefined, pagina: 1 })
  })

  it('shows the loading state and the backend error', () => {
    mockSales(emptyPage, { isLoading: true })
    const { unmount } = renderPage()
    expect(screen.getByRole('status', { name: 'Cargando ventas…' })).toBeInTheDocument()
    unmount()

    mockSales(emptyPage, { error: 'Servicio no disponible' })
    renderPage()
    expect(screen.getByText('Servicio no disponible')).toBeInTheDocument()
  })
})
