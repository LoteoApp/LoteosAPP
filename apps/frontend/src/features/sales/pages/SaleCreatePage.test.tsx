import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import SaleCreatePage, { type SaleCreatePageProps } from './SaleCreatePage'
import type { ClientOption, Sale, SaleCreateDevelopment, SellerOption } from '../types'

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

const cliente: ClientOption = { id: 'cl-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }

const agencySeller: SellerOption = {
  id: 'us-1',
  nombre: 'Marta',
  apellido: 'Suárez',
  rol: 'inmobiliaria',
  inmobiliariaId: 'ag-1',
  inmobiliariaRazonSocial: 'Inmobiliaria Sur',
}

const directSeller: SellerOption = { id: 'us-3', nombre: 'Sofía', apellido: 'Luna', rol: 'administrativo' }

const sale: Sale = {
  id: 'sale-1',
  loteoId: 'loteo-1',
  loteoNombre: 'Las Acacias',
  loteId: 'lot-1',
  loteNumero: '7',
  manzanaNumero: '2',
  loteSuperficie: 300,
  cliente,
  vendedor: { id: 'us-1', nombre: 'Marta', apellido: 'Suárez', rol: 'inmobiliaria' },
  usuarioAlta: { id: 'us-9', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
  inmobiliaria: { id: 'ag-1', razonSocial: 'Inmobiliaria Sur' },
  modalidadPago: 'contado',
  monto: 120000,
  moneda: 'USD',
  estado: 'activa',
  fechaCreacion: '2026-09-14T15:00:00Z',
  fechaModificacion: '2026-09-14T15:00:00Z',
}

function baseProps(): SaleCreatePageProps {
  return {
    loteoId: loteo.id,
    loteId: 'lot-1',
    loteo,
    loteoStatus: 'loaded',
    clients: [cliente],
    loadSellers: vi.fn().mockResolvedValue([agencySeller, directSeller]),
    createSale: vi.fn().mockResolvedValue(sale),
    renderPlan: <p>Plano del loteo</p>,
  }
}

function renderPage(overrides: Partial<SaleCreatePageProps> = {}) {
  const props: SaleCreatePageProps = { ...baseProps(), ...overrides }
  render(
    <MemoryRouter>
      <SaleCreatePage {...props} />
    </MemoryRouter>,
  )
  return props
}

async function fillSale(user: ReturnType<typeof userEvent.setup>) {
  await waitFor(() => expect(screen.getByLabelText('Inmobiliaria')).toBeEnabled())
  await user.click(screen.getByLabelText('Cliente'))
  await user.click(await screen.findByRole('option', { name: /Pérez, Ana/ }))
  await user.click(screen.getByLabelText('Inmobiliaria'))
  await user.click(await screen.findByRole('option', { name: 'Inmobiliaria Sur' }))
  await user.click(screen.getByLabelText('Vendedor'))
  await user.click(await screen.findByRole('option', { name: 'Suárez, Marta' }))
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

    await waitFor(() => expect(props.loadSellers).toHaveBeenCalledWith('loteo-1', expect.anything()))
    await waitFor(() => expect(screen.getByLabelText('Inmobiliaria')).toBeEnabled())
  })

  it('registers the sale and offers the receipt of the persisted venta', async () => {
    const user = userEvent.setup()
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})
    const props = renderPage()

    await fillSale(user)
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    await waitFor(() =>
      expect(props.createSale).toHaveBeenCalledWith(
        {
          loteoId: 'loteo-1',
          loteId: 'lot-1',
          clienteId: 'cl-1',
          vendedorId: 'us-1',
          modalidadPago: 'contado',
        },
        expect.any(String),
      ),
    )
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('Recibo de venta')
    expect(dialog).toHaveTextContent('Las Acacias · Mz 2 · Lote 7')
    expect(dialog).toHaveTextContent('Pérez, Ana')
    expect(dialog).toHaveTextContent('Suárez, Marta')
    expect(dialog).toHaveTextContent('Inmobiliaria Sur')
    expect(dialog).not.toHaveTextContent('provisorio')

    await user.click(within(dialog).getByRole('button', { name: 'Imprimir recibo' }))
    expect(print).toHaveBeenCalledTimes(1)
    print.mockRestore()

    await user.click(within(dialog).getByRole('button', { name: 'Cerrar' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())

    expect(screen.getByText('Venta registrada')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Confirmar venta' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Ver detalle' })).toHaveAttribute('href', '/ventas/sale-1')
    expect(screen.getByRole('link', { name: 'Ir al listado' })).toHaveAttribute('href', '/ventas')

    await user.click(screen.getByRole('button', { name: 'Imprimir recibo' }))
    expect(await screen.findByRole('dialog')).toHaveTextContent('Recibo de venta')
  })

  it('keeps the form and shows the backend error when the sale is rejected', async () => {
    const user = userEvent.setup()
    renderPage({
      createSale: vi.fn().mockRejectedValue(new ApiError('El lote no está disponible para vender', 'sale_lot_unavailable', 409)),
    })

    await fillSale(user)
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    expect(await screen.findByText('El lote no está disponible para vender')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeEnabled()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('retries a failed sale with the same idempotency key and rotates it after success', async () => {
    const user = userEvent.setup()
    const createSale = vi
      .fn()
      .mockRejectedValueOnce(new ApiError('Servicio no disponible', 'database_unavailable', 503))
      .mockResolvedValue(sale)
    renderPage({ createSale })

    await fillSale(user)
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))
    await screen.findByText('Servicio no disponible')
    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))
    await screen.findByRole('dialog')

    expect(createSale).toHaveBeenCalledTimes(2)
    const [, firstKey] = createSale.mock.calls[0] as [unknown, string]
    const [, secondKey] = createSale.mock.calls[1] as [unknown, string]
    expect(firstKey).not.toBe('')
    expect(secondKey).toBe(firstKey)
  })

  it('selects the cliente registered from the dialog app renders', async () => {
    const user = userEvent.setup()
    const onRegisterClient = vi.fn()
    const created: ClientOption = { id: 'cl-2', nombre: 'Bruno', apellido: 'Gómez', dni: '28999888' }
    const { rerender } = render(
      <MemoryRouter>
        <SaleCreatePage
          {...baseProps()}
          onRegisterClient={onRegisterClient}
          renderClientDialog={<p>Diálogo de cliente</p>}
        />
      </MemoryRouter>,
    )

    expect(screen.getByText('Diálogo de cliente')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Registrar cliente' }))
    expect(onRegisterClient).toHaveBeenCalledTimes(1)

    rerender(
      <MemoryRouter>
        <SaleCreatePage
          {...baseProps()}
          clients={[created, cliente]}
          createdClient={created}
          onRegisterClient={onRegisterClient}
          renderClientDialog={<p>Diálogo de cliente</p>}
        />
      </MemoryRouter>,
    )

    expect(screen.getByLabelText('Cliente')).toHaveValue('Gómez, Bruno · DNI 28999888')
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
    expect(props.loadSellers).not.toHaveBeenCalled()
  })

  it('warns when the lote does not belong to the loteo', () => {
    renderPage({ loteId: 'lot-999' })

    expect(screen.getByText('El lote seleccionado no existe.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeDisabled()
  })
})
