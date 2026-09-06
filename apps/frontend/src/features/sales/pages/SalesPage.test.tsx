import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import SalesPage, { type SalesPageProps } from './SalesPage'
import type { AgencyOption, ClienteOption, LoteOption } from '../types'

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

function agency(overrides: Partial<AgencyOption> = {}): AgencyOption {
  return { id: 'ag-1', razonSocial: 'Inmobiliaria Sur', ...overrides }
}

function renderPage(overrides: Partial<SalesPageProps> = {}) {
  const props: SalesPageProps = {
    loadLotes: vi.fn().mockResolvedValue([lote()]),
    loadClientes: vi.fn().mockResolvedValue([cliente()]),
    createCliente: vi.fn(),
    loadAgencies: vi.fn().mockResolvedValue([agency()]),
    ...overrides,
  }

  render(<SalesPage {...props} />)
  return props
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
  })

  it('hides the inmobiliaria when the caller has no agency to pick', async () => {
    renderPage({ loadAgencies: undefined })

    await waitFor(() => {
      expect(screen.getByLabelText('Lote')).toBeEnabled()
    })
    expect(screen.queryByLabelText('Inmobiliaria')).not.toBeInTheDocument()
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

  it('reports a failure while loading the form data', async () => {
    renderPage({ loadLotes: vi.fn().mockRejectedValue(new Error('boom')) })

    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })
})
