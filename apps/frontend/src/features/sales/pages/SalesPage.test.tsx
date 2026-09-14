import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import SalesPage, { type SalesPageProps } from './SalesPage'
import type { ClienteOption, LoteOption, SellerOption } from '../types'

function lote(overrides: Partial<LoteOption> = {}): LoteOption {
  return {
    id: 'lt-1',
    numero: '7',
    manzanaId: 'mz-1',
    manzanaNumero: '1',
    loteoId: 'loteo-1',
    loteoNombre: 'Norte',
    estado: 'disponible',
    precio: 150000,
    moneda: 'USD',
    superficie: 300,
    ...overrides,
  }
}

function cliente(overrides: Partial<ClienteOption> = {}): ClienteOption {
  return { id: 'cl-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222', ...overrides }
}

const agencySeller: SellerOption = {
  id: 'us-1',
  nombre: 'Marta',
  apellido: 'Suárez',
  rol: 'inmobiliaria',
  inmobiliariaId: 'ag-1',
  inmobiliariaRazonSocial: 'Inmobiliaria Sur',
}

const otherAgencySeller: SellerOption = {
  id: 'us-2',
  nombre: 'Diego',
  apellido: 'Ramos',
  rol: 'inmobiliaria',
  inmobiliariaId: 'ag-2',
  inmobiliariaRazonSocial: 'Norte Propiedades',
}

const directSeller: SellerOption = {
  id: 'us-3',
  nombre: 'Sofía',
  apellido: 'Luna',
  rol: 'administrativo',
}

function renderPage(overrides: Partial<SalesPageProps> = {}) {
  const props: SalesPageProps = {
    loadLotes: vi.fn().mockResolvedValue([lote()]),
    loadClientes: vi.fn().mockResolvedValue([cliente()]),
    createCliente: vi.fn(),
    loadSellers: vi.fn().mockResolvedValue([agencySeller, otherAgencySeller, directSeller]),
    ...overrides,
  }

  render(<SalesPage {...props} />)
  return props
}

async function selectLote(user: ReturnType<typeof userEvent.setup>) {
  const input = screen.getByLabelText('Lote')
  await waitFor(() => expect(input).toBeEnabled())
  await user.click(input)
  await user.click(await screen.findByRole('option', { name: /Norte · Mz 1 · Lote 7/ }))
}

async function selectCliente(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByLabelText('Cliente'))
  await user.click(await screen.findByRole('option', { name: /Pérez, Ana/ }))
}

async function selectSeller(
  user: ReturnType<typeof userEvent.setup>,
  agencyName: string,
  sellerName: string,
) {
  await waitFor(() => expect(screen.getByLabelText('Inmobiliaria')).toBeEnabled())
  await user.click(screen.getByLabelText('Inmobiliaria'))
  await user.click(await screen.findByRole('option', { name: agencyName }))
  await user.click(screen.getByLabelText('Vendedor'))
  await user.click(await screen.findByRole('option', { name: sellerName }))
}

describe('SalesPage', () => {
  it('offers the lote, the cliente and the inmobiliaria', async () => {
    renderPage()

    expect(screen.getByRole('heading', { name: 'Nueva venta' })).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByLabelText('Lote')).toBeEnabled()
    })
    expect(screen.getByLabelText('Cliente')).toBeInTheDocument()
    expect(screen.getByLabelText('Inmobiliaria')).toBeInTheDocument()
    expect(screen.getByLabelText('Vendedor')).toBeInTheDocument()
  })

  it('waits for the lote before offering inmobiliarias and vendedores', async () => {
    const loadSellers = vi.fn().mockResolvedValue([agencySeller])
    renderPage({ loadSellers })

    await waitFor(() => {
      expect(screen.getByLabelText('Lote')).toBeEnabled()
    })
    expect(screen.getByLabelText('Inmobiliaria')).toBeDisabled()
    expect(screen.getByLabelText('Vendedor')).toBeDisabled()
    expect(loadSellers).not.toHaveBeenCalled()
  })

  it('asks the sellers eligible for the loteo of the chosen lote', async () => {
    const user = userEvent.setup()
    const loadSellers = vi.fn().mockResolvedValue([agencySeller, otherAgencySeller])
    renderPage({ loadSellers })

    await selectLote(user)

    await waitFor(() => {
      expect(loadSellers).toHaveBeenCalledWith('loteo-1', expect.anything())
    })
  })

  it('preselects the logged-in user as a direct sale', async () => {
    const user = userEvent.setup()
    renderPage({
      loadSellers: vi
        .fn()
        .mockResolvedValue([agencySeller, { ...directSeller, esActor: true }, otherAgencySeller]),
    })

    await selectLote(user)

    await waitFor(() => {
      expect(screen.getByLabelText('Vendedor')).toHaveValue('Luna, Sofía')
    })
    expect(screen.getByLabelText('Inmobiliaria')).toHaveValue('Venta directa')
  })

  it('lets the logged-in user hand the sale to another vendedor', async () => {
    const user = userEvent.setup()
    renderPage({
      loadSellers: vi
        .fn()
        .mockResolvedValue([agencySeller, { ...directSeller, esActor: true }, otherAgencySeller]),
    })

    await selectLote(user)
    await waitFor(() => expect(screen.getByLabelText('Vendedor')).toHaveValue('Luna, Sofía'))

    await selectSeller(user, 'Inmobiliaria Sur', 'Suárez, Marta')

    expect(screen.getByLabelText('Vendedor')).toHaveValue('Suárez, Marta')
  })

  it('preselects the seller when only one is eligible', async () => {
    const user = userEvent.setup()
    renderPage({ loadSellers: vi.fn().mockResolvedValue([agencySeller]) })

    await selectLote(user)

    await waitFor(() => {
      expect(screen.getByLabelText('Vendedor')).toHaveValue('Suárez, Marta')
    })
    expect(screen.getByLabelText('Inmobiliaria')).toHaveValue('Inmobiliaria Sur')
  })

  it('narrows the vendedores to the chosen inmobiliaria', async () => {
    const user = userEvent.setup()
    renderPage()

    await selectLote(user)
    await waitFor(() => expect(screen.getByLabelText('Inmobiliaria')).toBeEnabled())

    await user.click(screen.getByLabelText('Inmobiliaria'))
    expect(await screen.findByRole('option', { name: 'Venta directa' })).toBeInTheDocument()
    await user.click(await screen.findByRole('option', { name: 'Inmobiliaria Sur' }))

    await user.click(screen.getByLabelText('Vendedor'))

    expect(await screen.findByRole('option', { name: 'Suárez, Marta' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Ramos, Diego' })).not.toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Luna, Sofía' })).not.toBeInTheDocument()
  })

  it('offers the internal users under venta directa', async () => {
    const user = userEvent.setup()
    renderPage()

    await selectLote(user)
    await waitFor(() => expect(screen.getByLabelText('Inmobiliaria')).toBeEnabled())

    await user.click(screen.getByLabelText('Inmobiliaria'))
    await user.click(await screen.findByRole('option', { name: 'Venta directa' }))

    await user.click(screen.getByLabelText('Vendedor'))

    expect(await screen.findByRole('option', { name: 'Luna, Sofía' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Suárez, Marta' })).not.toBeInTheDocument()
  })

  it('clears the vendedor when the inmobiliaria changes', async () => {
    const user = userEvent.setup()
    renderPage()

    await selectLote(user)
    await selectSeller(user, 'Inmobiliaria Sur', 'Suárez, Marta')

    await user.click(screen.getByLabelText('Inmobiliaria'))
    await user.click(await screen.findByRole('option', { name: 'Norte Propiedades' }))

    expect(screen.getByLabelText('Vendedor')).toHaveValue('')
  })

  it('reports a failure while loading the sellers', async () => {
    const user = userEvent.setup()
    renderPage({ loadSellers: vi.fn().mockRejectedValue(new Error('boom')) })

    await selectLote(user)

    expect(await screen.findByRole('alert')).toBeInTheDocument()
    expect(screen.getByLabelText('Inmobiliaria')).toHaveValue('')
  })

  it('selects a lote from the search results', async () => {
    const user = userEvent.setup()
    renderPage()

    const input = screen.getByLabelText('Lote')
    await waitFor(() => expect(input).toBeEnabled())

    await user.click(input)
    await user.type(input, 'Norte')

    const option = await screen.findByRole('option', { name: /Norte · Mz 1 · Lote 7/ })
    await user.click(option)

    expect(input).toHaveValue('Norte · Mz 1 · Lote 7')
  })

  it('selects a cliente from the search results', async () => {
    const user = userEvent.setup()
    renderPage()

    const input = screen.getByLabelText('Cliente')
    await waitFor(() => expect(input).toBeEnabled())

    await user.click(input)
    await user.type(input, 'Pérez')

    await user.click(await screen.findByRole('option', { name: /Pérez, Ana/ }))

    expect(input).toHaveValue('Pérez, Ana · DNI 30111222')
  })

  it('registers a cliente from the modal and selects it', async () => {
    const user = userEvent.setup()
    const created = cliente({ id: 'cl-2', nombre: 'Luis', apellido: 'Gómez', dni: '28999111' })
    const createCliente = vi.fn().mockResolvedValue(created)
    renderPage({ createCliente })

    await waitFor(() => expect(screen.getByLabelText('Cliente')).toBeEnabled())
    await user.click(screen.getByRole('button', { name: 'Registrar cliente' }))

    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText('Nombre'), 'Luis')
    await user.type(within(dialog).getByLabelText('Apellido'), 'Gómez')
    await user.type(within(dialog).getByLabelText('DNI'), '28999111')
    await user.click(within(dialog).getByRole('button', { name: 'Guardar cliente' }))

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
    expect(createCliente).toHaveBeenCalledWith({
      nombre: 'Luis',
      apellido: 'Gómez',
      dni: '28999111',
      celular: '',
      email: '',
    })
    expect(screen.getByLabelText('Cliente')).toHaveValue('Gómez, Luis · DNI 28999111')
  })

  it('asks for the mandatory fields before creating a cliente', async () => {
    const user = userEvent.setup()
    const createCliente = vi.fn()
    renderPage({ createCliente })

    await waitFor(() => expect(screen.getByLabelText('Cliente')).toBeEnabled())
    await user.click(screen.getByRole('button', { name: 'Registrar cliente' }))

    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText('Nombre'), 'Luis')
    await user.click(within(dialog).getByRole('button', { name: 'Guardar cliente' }))

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      'Completá nombre, apellido y DNI.',
    )
    expect(createCliente).not.toHaveBeenCalled()
  })

  it('confirms the sale and offers the receipt to print', async () => {
    const user = userEvent.setup()
    const print = vi.spyOn(window, 'print').mockImplementation(() => {})
    renderPage()

    await selectLote(user)
    await selectCliente(user)
    await selectSeller(user, 'Inmobiliaria Sur', 'Suárez, Marta')

    await user.click(screen.getByRole('button', { name: 'Confirmar venta' }))

    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('Recibo de venta')
    expect(dialog).toHaveTextContent('Norte · Mz 1 · Lote 7')
    expect(dialog).toHaveTextContent('Pérez, Ana')
    expect(dialog).toHaveTextContent('Suárez, Marta')
    expect(dialog).toHaveTextContent('Inmobiliaria Sur')

    await user.click(within(dialog).getByRole('button', { name: 'Imprimir recibo' }))

    expect(print).toHaveBeenCalledTimes(1)
    print.mockRestore()
  })

  it('keeps the confirmation disabled and says what is missing', async () => {
    const user = userEvent.setup()
    renderPage()

    const confirm = screen.getByRole('button', { name: 'Confirmar venta' })
    await waitFor(() => expect(screen.getByLabelText('Lote')).toBeEnabled())

    expect(confirm).toBeDisabled()
    expect(screen.getByText('Elegí el lote que se vende.')).toBeInTheDocument()

    await selectLote(user)

    expect(confirm).toBeDisabled()
    expect(screen.getByText('Elegí el cliente comprador.')).toBeInTheDocument()

    await selectCliente(user)

    expect(confirm).toBeDisabled()
    expect(screen.getByText('Elegí el vendedor que realizó la venta.')).toBeInTheDocument()

    await selectSeller(user, 'Inmobiliaria Sur', 'Suárez, Marta')

    expect(confirm).toBeEnabled()
  })

  it('does not confirm a lote without a price', async () => {
    const user = userEvent.setup()
    renderPage({ loadLotes: vi.fn().mockResolvedValue([lote({ precio: null })]) })

    await selectLote(user)
    await selectCliente(user)
    await selectSeller(user, 'Inmobiliaria Sur', 'Suárez, Marta')

    expect(screen.getByRole('button', { name: 'Confirmar venta' })).toBeDisabled()
    expect(screen.getByRole('alert')).toHaveTextContent('El lote no tiene precio cargado.')
  })

  it('reports a failure while loading the form data', async () => {
    renderPage({ loadLotes: vi.fn().mockRejectedValue(new Error('boom')) })

    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })

  it('drops the lotes and their failure when the screen is unmounted while loading', async () => {
    let resolveLotes: (lotes: LoteOption[]) => void = () => {}
    let rejectLotes: (error: Error) => void = () => {}
    const loadLotes = vi
      .fn()
      .mockImplementationOnce(() => new Promise<LoteOption[]>((resolve) => { resolveLotes = resolve }))
      .mockImplementationOnce(() => new Promise<LoteOption[]>((_, reject) => { rejectLotes = reject }))

    const first = render(<SalesPage loadLotes={loadLotes} loadClientes={vi.fn().mockResolvedValue([])} createCliente={vi.fn()} loadSellers={vi.fn()} />)
    first.unmount()
    resolveLotes([lote()])
    await Promise.resolve()

    const second = render(<SalesPage loadLotes={loadLotes} loadClientes={vi.fn().mockResolvedValue([])} createCliente={vi.fn()} loadSellers={vi.fn()} />)
    second.unmount()
    rejectLotes(new Error('boom'))
    await Promise.resolve()

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Lote')).not.toBeInTheDocument()
  })
})
