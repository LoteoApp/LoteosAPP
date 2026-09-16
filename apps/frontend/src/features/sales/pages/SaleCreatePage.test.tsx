import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import SaleCreatePage, { type SaleCreatePageProps } from './SaleCreatePage'
import type { ClientOption, Sale, SaleCreateDevelopment, SellerOption } from '../types'

const development: SaleCreateDevelopment = {
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

const client: ClientOption = { id: 'cl-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }

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
  cliente: client,
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
    developmentId: development.id,
    lotId: 'lot-1',
    development,
    developmentStatus: 'loaded',
    clients: [client],
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

  it('registers an entrega + financiación sale with its plan', async () => {
    const user = userEvent.setup()
    const props = renderPage({
      createSale: vi.fn().mockResolvedValue({
        ...sale,
        modalidadPago: 'entrega_financiada',
        planPago: {
          id: 'plan-1',
          montoEntrega: 20000,
          cantidadCuotas: 10,
          tasaInteres: 5,
          periodicidad: 'mensual',
          moneda: 'USD',
          montoFinanciado: 100000,
          montoCuota: 10500,
          montoTotal: 105000,
          cuotas: [],
        },
      }),
    })

    await fillSale(user)
    await user.click(screen.getByLabelText('Condiciones de pago'))
    await user.click(await screen.findByRole('option', { name: 'Entrega + financiación' }))

    expect(screen.getByText('Ingresá una cantidad de cuotas entre 1 y 360.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeDisabled()

    await user.type(screen.getByLabelText('Cantidad de cuotas'), '10')
    expect(screen.getByText('Ingresá el monto de la entrega, con hasta 2 decimales.')).toBeInTheDocument()
    await user.type(screen.getByLabelText('Monto de entrega'), '20000')
    await user.type(screen.getByLabelText('Tasa de interés (%)'), '5')

    const preview = screen.getByLabelText('Detalle del plan')
    expect(preview).toHaveTextContent('US$ 100.000,00')
    expect(preview).toHaveTextContent('10 de US$ 10.500,00')
    expect(preview).toHaveTextContent('US$ 105.000,00')
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeEnabled()

    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    await waitFor(() =>
      expect(props.createSale).toHaveBeenCalledWith(
        {
          loteoId: 'loteo-1',
          loteId: 'lot-1',
          clienteId: 'cl-1',
          vendedorId: 'us-1',
          modalidadPago: 'entrega_financiada',
          planPago: { cantidadCuotas: 10, tasaInteres: 5, periodicidad: 'mensual', montoEntrega: 20000 },
        },
        expect.any(String),
      ),
    )
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('Entrega + financiación')
    expect(within(dialog).getByLabelText('Plan de pago')).toHaveTextContent('10 de US$ 10.500,00 · mensual')
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
          clients={[created, client]}
          createdClient={created}
          onRegisterClient={onRegisterClient}
          renderClientDialog={<p>Diálogo de cliente</p>}
        />
      </MemoryRouter>,
    )

    expect(screen.getByLabelText('Cliente')).toHaveValue('Gómez, Bruno · DNI 28999888')
  })

  it('fixes the seller to the internal user on a venta directa and frees it for an agency', async () => {
    const user = userEvent.setup()
    const actor: SellerOption = { ...directSeller, esActor: true }
    renderPage({ loadSellers: vi.fn().mockResolvedValue([agencySeller, actor]) })

    await waitFor(() => expect(screen.getByLabelText('Inmobiliaria')).toHaveValue('Venta directa'))
    expect(screen.getByLabelText('Vendedor')).toHaveValue('Luna, Sofía')
    expect(screen.getByLabelText('Vendedor')).toBeDisabled()
    expect(screen.getByText(/Vendés a tu nombre/)).toBeInTheDocument()

    await user.click(screen.getByLabelText('Inmobiliaria'))
    await user.click(await screen.findByRole('option', { name: 'Inmobiliaria Sur' }))
    expect(screen.getByLabelText('Vendedor')).toBeEnabled()
    expect(screen.getByLabelText('Vendedor')).toHaveValue('')
    await user.click(screen.getByLabelText('Vendedor'))
    await user.click(await screen.findByRole('option', { name: 'Suárez, Marta' }))
    expect(screen.getByLabelText('Vendedor')).toHaveValue('Suárez, Marta')

    await user.click(screen.getByLabelText('Inmobiliaria'))
    await user.click(await screen.findByRole('option', { name: 'Venta directa' }))
    expect(screen.getByLabelText('Vendedor')).toHaveValue('Luna, Sofía')
    expect(screen.getByLabelText('Vendedor')).toBeDisabled()
  })

  it('shows the loading state before the loteo arrives', () => {
    renderPage({ development: null, developmentStatus: 'loading' })

    expect(screen.getByRole('status', { name: 'Cargando los datos para registrar la venta…' })).toBeInTheDocument()
    expect(screen.queryByLabelText('Cliente')).not.toBeInTheDocument()
  })

  it('explains when the loteo cannot be loaded', () => {
    renderPage({ development: null, developmentStatus: 'not-found' })
    expect(screen.getByText('No se puede iniciar la venta')).toBeInTheDocument()
    expect(screen.getByText('No encontramos este loteo.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Volver al loteo' })).toHaveAttribute('href', '/lotes/loteo-1')
  })

  it('shows the backend error when the loteo fails to load', () => {
    renderPage({ development: null, developmentStatus: 'error', developmentError: 'Servicio no disponible' })
    expect(screen.getByText('Servicio no disponible')).toBeInTheDocument()
  })

  it('blocks the sale of a lote that is no longer available', async () => {
    const props = renderPage({ lotId: 'lot-2' })

    expect(screen.getByText('Lote no disponible')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeDisabled()
    expect(screen.getByLabelText('Cliente')).toBeDisabled()
    expect(props.loadSellers).not.toHaveBeenCalled()
  })

  it('warns when the lote does not belong to the loteo', () => {
    renderPage({ lotId: 'lot-999' })

    expect(screen.getByText('El lote seleccionado no existe.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeDisabled()
  })
})
