import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ComponentProps } from 'react'
import { MemoryRouter } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import ReservationCreatePage from './ReservationCreatePage'
import type { Reservation, ReservationClient, ReservationCreateDevelopment } from '../types'

const useEligibleSellersMock = vi.hoisted(() => vi.fn())
const useReservationMutationsMock = vi.hoisted(() => vi.fn())
const createReservationMock = vi.hoisted(() => vi.fn())
const downloadReceiptMock = vi.hoisted(() => vi.fn())
let restoreURLMocks: (() => void) | null = null

vi.mock('../hooks/use-eligible-sellers', () => ({ useEligibleSellers: useEligibleSellersMock }))
vi.mock('../hooks/use-reservation-mutations', () => ({ useReservationMutations: useReservationMutationsMock }))
vi.mock('../api/reservations', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../api/reservations')>()),
  downloadReservationReceipt: downloadReceiptMock,
}))

const loteo: ReservationCreateDevelopment = {
  id: 'loteo-1',
  nombre: 'Las Acacias',
  ubicacion: 'Córdoba',
  descripcion: 'A metros de la ruta.',
  manzanas: [{ id: 'block-1', numero: '2', tieneAgua: true, tieneCloaca: false, tieneLuz: true, tieneGas: false }],
  lotes: [{ id: 'lot-1', manzanaId: 'block-1', numero: '7', estado: 'disponible', precio: 120000, moneda: 'USD', superficie: 300, caracteristicas: 'Frente norte' }],
}
const client: ReservationClient = { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222', celular: '', email: '' }
const seller = { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrador' }
const reservation: Reservation = {
  id: 'reservation-1', loteoId: loteo.id, loteoNombre: loteo.nombre, loteId: 'lot-1', loteNumero: '7',
  cliente: { id: client.id, nombre: client.nombre, apellido: client.apellido, dni: client.dni },
  vendedor: seller, usuarioAlta: seller, estado: 'activa', fechaVencimiento: '2026-09-21T12:00:00Z',
  fechaCreacion: '2026-09-06T12:00:00Z', fechaModificacion: '2026-09-06T12:00:00Z',
}

function renderPage(overrides: Partial<ComponentProps<typeof ReservationCreatePage>> = {}) {
  const props: ComponentProps<typeof ReservationCreatePage> = {
    accessToken: 'token',
    loteoId: loteo.id,
    loteId: 'lot-1',
    loteo,
    loteoStatus: 'loaded',
    clients: [client],
    clientsLoading: false,
    clientsError: null,
    isAgencyUser: false,
    renderPlan: <p>Plano completo del loteo</p>,
    ...overrides,
  }
  return render(<MemoryRouter><ReservationCreatePage {...props} /></MemoryRouter>)
}

function configureMocks() {
  useEligibleSellersMock.mockReturnValue({ sellers: [seller], isLoading: false, error: null })
  createReservationMock.mockResolvedValue(reservation)
  useReservationMutationsMock.mockReturnValue({
    create: createReservationMock,
    isSubmitting: false,
    error: null,
  })
  downloadReceiptMock.mockRejectedValue(new Error('No se pudo descargar el comprobante.'))
}

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
  restoreURLMocks?.()
  restoreURLMocks = null
})

describe('ReservationCreatePage', () => {
  it('requires an administrative seller and keeps the reservation after receipt failure', async () => {
    configureMocks()
    const user = userEvent.setup()
    renderPage()

    expect(screen.getByText('Plano completo del loteo')).toBeInTheDocument()
    expect(screen.getByText('A metros de la ruta.')).toBeInTheDocument()
    expect(screen.getByText('Frente norte')).toBeInTheDocument()

    await user.click(screen.getByRole('combobox', { name: 'Cliente' }))
    await user.click(await screen.findByRole('option', { name: /Pérez, Ana/ }))
    await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))
    expect(screen.getByText('Seleccioná un vendedor.')).toBeInTheDocument()

    await user.click(screen.getByRole('combobox', { name: 'Vendedor' }))
    await user.click(await screen.findByRole('option', { name: /Gómez, Beto/ }))
    await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))
    expect(await screen.findByText('Reserva creada')).toBeInTheDocument()
    expect(await screen.findByText('No se pudo descargar el comprobante.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Descargar comprobante PDF' }))
    await waitFor(() => expect(downloadReceiptMock).toHaveBeenCalledTimes(2))
    expect(screen.getByRole('link', { name: 'Ver detalle' })).toHaveAttribute('href', '/reservas/reservation-1')
  })

  it('fixes the seller to the current agency user', async () => {
    configureMocks()
    const user = userEvent.setup()
    renderPage({ isAgencyUser: true })

    expect(screen.getByText('Gómez, Beto')).toBeInTheDocument()
    expect(screen.queryByRole('combobox', { name: 'Vendedor' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('combobox', { name: 'Cliente' }))
    await user.click(await screen.findByRole('option', { name: /Pérez, Ana/ }))
    await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))
    await waitFor(() => expect(createReservationMock).toHaveBeenCalledWith(
      { loteoId: loteo.id, loteId: 'lot-1', clienteId: client.id, vendedorId: seller.id },
      expect.any(String),
    ))
  })

  it('shows loading and unavailable loteo states without opening the reservation form', () => {
    configureMocks()
    renderPage({ loteoStatus: 'loading' })
    const placeholder = screen.getByRole('status', { name: 'Cargando los datos para crear la reserva…' })
    expect(placeholder).not.toHaveAttribute('aria-busy')
    expect(placeholder).toHaveAttribute('aria-live', 'polite')
    expect(screen.getByRole('heading', { name: 'Nueva reserva' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Volver al loteo' })).toHaveAttribute('href', '/lotes/loteo-1')
    expect(screen.queryByRole('button', { name: 'Confirmar reserva' })).not.toBeInTheDocument()

    cleanup()
    renderPage({ loteoStatus: 'not-found', loteo: null })
    expect(screen.getByText('No encontramos este loteo.')).toBeInTheDocument()

    cleanup()
    renderPage({ loteoStatus: 'error', loteo: null, loteoError: 'Loteo sin acceso' })
    expect(screen.getByText('Loteo sin acceso')).toBeInTheDocument()

    cleanup()
    renderPage({ loteId: 'missing-lot' })
    expect(screen.getByText('El lote seleccionado no existe.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirmar reserva' })).toBeDisabled()
  })

  it('downloads the PDF automatically after the reservation is saved', async () => {
    configureMocks()
    downloadReceiptMock.mockResolvedValue(new Blob(['%PDF-1.4'], { type: 'application/pdf' }))
    const createObjectURL = vi.fn().mockReturnValue('blob:reservation-receipt')
    const revokeObjectURL = vi.fn()
    const createDescriptor = Object.getOwnPropertyDescriptor(URL, 'createObjectURL')
    const revokeDescriptor = Object.getOwnPropertyDescriptor(URL, 'revokeObjectURL')
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: createObjectURL })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: revokeObjectURL })
    restoreURLMocks = () => {
      if (createDescriptor) Object.defineProperty(URL, 'createObjectURL', createDescriptor)
      else Reflect.deleteProperty(URL, 'createObjectURL')
      if (revokeDescriptor) Object.defineProperty(URL, 'revokeObjectURL', revokeDescriptor)
      else Reflect.deleteProperty(URL, 'revokeObjectURL')
    }
    const anchorClick = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    const user = userEvent.setup()
    renderPage()

    await user.click(screen.getByRole('combobox', { name: 'Cliente' }))
    await user.click(await screen.findByRole('option', { name: /Pérez, Ana/ }))
    await user.click(screen.getByRole('combobox', { name: 'Vendedor' }))
    await user.click(await screen.findByRole('option', { name: /Gómez, Beto/ }))
    await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))

    expect(await screen.findByText('Reserva creada')).toBeInTheDocument()
    await waitFor(() => expect(anchorClick).toHaveBeenCalled())
    expect(downloadReceiptMock).toHaveBeenCalledWith('token', 'reservation-1')
    expect(createObjectURL).toHaveBeenCalledWith(expect.any(Blob))
  })
})
