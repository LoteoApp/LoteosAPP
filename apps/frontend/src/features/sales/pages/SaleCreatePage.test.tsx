import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import SaleCreatePage, { type SaleCreatePageProps } from './SaleCreatePage'
import type { ClienteOption, SaleCreateDevelopment, SellerOption } from '../types'

const loteo: SaleCreateDevelopment = {
  id: 'loteo-1',
  nombre: 'Las Acacias',
  ubicacion: 'Córdoba',
  descripcion: 'A metros de la ruta.',
  manzanas: [{ id: 'block-1', numero: '2', tieneAgua: true, tieneCloaca: false, tieneLuz: true, tieneGas: false }],
  lotes: [
    { id: 'lot-1', manzanaId: 'block-1', numero: '7', estado: 'disponible', precio: 120000, moneda: 'USD', superficie: 300, caracteristicas: 'Frente norte' },
    { id: 'lot-2', manzanaId: 'block-1', numero: '8', estado: 'vendido', precio: 120000, moneda: 'USD', superficie: 300, caracteristicas: '' },
  ],
}

const cliente: ClienteOption = { id: 'cl-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }

const agencySeller: SellerOption = {
  id: 'us-1',
  nombre: 'Marta',
  apellido: 'Suárez',
  rol: 'inmobiliaria',
  inmobiliariaId: 'ag-1',
  inmobiliariaRazonSocial: 'Inmobiliaria Sur',
}

const directSeller: SellerOption = { id: 'us-3', nombre: 'Sofía', apellido: 'Luna', rol: 'administrativo' }

function renderPage(overrides: Partial<SaleCreatePageProps> = {}) {
  const props: SaleCreatePageProps = {
    loteoId: loteo.id,
    loteId: 'lot-1',
    loteo,
    loteoStatus: 'loaded',
    loadClientes: vi.fn().mockResolvedValue([cliente]),
    createCliente: vi.fn(),
    loadSellers: vi.fn().mockResolvedValue([agencySeller, directSeller]),
    renderPlan: <p>Plano del loteo</p>,
    ...overrides,
  }
  render(
    <MemoryRouter>
      <SaleCreatePage {...props} />
    </MemoryRouter>,
  )
  return props
}

describe('SaleCreatePage', () => {
  it('preloads the lote from the viewer and asks its loteo for the sellers', async () => {
    const props = renderPage()

    expect(screen.getByRole('heading', { name: 'Nueva venta' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Volver al loteo' })).toHaveAttribute('href', '/lotes/loteo-1')
    expect(screen.getByText('Plano del loteo')).toBeInTheDocument()
    expect(screen.getByText('Las Acacias')).toBeInTheDocument()
    expect(screen.getByText('Córdoba · Manzana 2 · Lote 7')).toBeInTheDocument()
    expect(screen.getByText('Frente norte')).toBeInTheDocument()
    expect(screen.getByText('Agua · Luz')).toBeInTheDocument()
    expect(screen.queryByLabelText('Lote')).not.toBeInTheDocument()

    await waitFor(() => expect(props.loadSellers).toHaveBeenCalledWith('loteo-1', expect.anything()))
    await waitFor(() => expect(screen.getByLabelText('Inmobiliaria')).toBeEnabled())
  })

  it('sells the lote to a cliente through the chosen inmobiliaria and vendedor', async () => {
    const user = userEvent.setup()
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})
    renderPage()

    await waitFor(() => expect(screen.getByLabelText('Inmobiliaria')).toBeEnabled())
    await user.click(screen.getByLabelText('Cliente'))
    await user.click(await screen.findByRole('option', { name: /Pérez, Ana/ }))
    await user.click(screen.getByLabelText('Inmobiliaria'))
    await user.click(await screen.findByRole('option', { name: 'Inmobiliaria Sur' }))
    await user.click(screen.getByLabelText('Vendedor'))
    await user.click(await screen.findByRole('option', { name: 'Suárez, Marta' }))

    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('Recibo de venta')
    expect(dialog).toHaveTextContent('Las Acacias · Mz 2 · Lote 7')
    expect(dialog).toHaveTextContent('Pérez, Ana')
    expect(dialog).toHaveTextContent('Suárez, Marta')
    expect(dialog).toHaveTextContent('Inmobiliaria Sur')

    await user.click(within(dialog).getByRole('button', { name: 'Imprimir recibo' }))
    expect(print).toHaveBeenCalledTimes(1)
    print.mockRestore()
  })

  it('shows the loading state before the loteo arrives', () => {
    renderPage({ loteo: null, loteoStatus: 'loading' })

    expect(screen.getByRole('status', { name: 'Cargando los datos para registrar la venta…' })).toBeInTheDocument()
    expect(screen.queryByLabelText('Cliente')).not.toBeInTheDocument()
  })

  it('explains when the loteo cannot be loaded', () => {
    renderPage({ loteo: null, loteoStatus: 'not-found' })
    expect(screen.getByText('No se puede iniciar la venta')).toBeInTheDocument()
    expect(screen.getByText('No encontramos este loteo.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Volver al loteo' })).toHaveAttribute('href', '/lotes/loteo-1')
  })

  it('shows the backend error when the loteo fails to load', () => {
    renderPage({ loteo: null, loteoStatus: 'error', loteoError: 'Servicio no disponible' })
    expect(screen.getByText('Servicio no disponible')).toBeInTheDocument()
  })

  it('blocks the sale of a lote that is no longer available', async () => {
    const props = renderPage({ loteId: 'lot-2' })

    expect(screen.getByText('Lote no disponible')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeDisabled()
    expect(screen.getByLabelText('Cliente')).toBeDisabled()
    await waitFor(() => expect(props.loadClientes).toHaveBeenCalled())
    expect(props.loadSellers).not.toHaveBeenCalled()
  })

  it('warns when the lote does not belong to the loteo', () => {
    renderPage({ loteId: 'lot-999' })

    expect(screen.getByText('El lote seleccionado no existe.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeDisabled()
  })
})
