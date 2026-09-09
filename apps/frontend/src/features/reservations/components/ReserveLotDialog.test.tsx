import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ReservationClient, ReservationLot, SellerOption } from '../types'
import ReserveLotDialog from './ReserveLotDialog'

const useEligibleSellersMock = vi.hoisted(() => vi.fn())
const useReservationMutationsMock = vi.hoisted(() => vi.fn())

vi.mock('../hooks/use-eligible-sellers', () => ({ useEligibleSellers: useEligibleSellersMock }))
vi.mock('../hooks/use-reservation-mutations', () => ({ useReservationMutations: useReservationMutationsMock }))

const lot: ReservationLot = {
	id: 'lot-12345678', numero: '7', estado: 'disponible', precio: 100000,
}
const fallbackLot: ReservationLot = { ...lot, id: 'lot-fallback', numero: '', precio: null }
const client: ReservationClient = { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' }
const sellers: SellerOption[] = [
	{ id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrador' },
	{ id: 'seller-2', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
]

function configureMocks(overrides: {
	clientsError?: string | null
	sellers?: SellerOption[]
	sellersError?: string | null
	create?: ReturnType<typeof vi.fn>
} = {}) {
	useEligibleSellersMock.mockReturnValue({ sellers: overrides.sellers ?? sellers, isLoading: false, error: overrides.sellersError ?? null })
	useReservationMutationsMock.mockReturnValue({
		isSubmitting: false,
		error: null,
		create: overrides.create ?? vi.fn().mockResolvedValue({ id: 'reservation-1' }),
		cancel: vi.fn(),
		reset: vi.fn(),
	})
}

beforeEach(() => {
	vi.clearAllMocks()
})

describe('ReserveLotDialog', () => {
	it('submits the selected lot and closes after success', async () => {
		const user = userEvent.setup()
		const onCreated = vi.fn()
		const reset = vi.fn()
		const create = vi.fn().mockResolvedValue({ id: 'reservation-1' })
		configureMocks({ create })
		useReservationMutationsMock.mockReturnValue({ isSubmitting: false, error: null, create, cancel: vi.fn(), reset })

		render(<ReserveLotDialog accessToken="token" loteoId="loteo-1" lote={lot} clients={[client]} onCreated={onCreated} />)
		await user.click(screen.getByRole('button', { name: 'Reservar lote' }))
		expect(screen.getByRole('heading', { name: 'Nueva reserva · Lote 7' })).toBeInTheDocument()
		expect(screen.getByText('ID: lot-12345678')).toBeInTheDocument()

		await user.click(screen.getByRole('combobox', { name: 'Cliente' }))
		await user.click(screen.getByRole('option', { name: /Pérez, Ana/ }))
		await user.click(screen.getByRole('combobox', { name: 'Vendedor' }))
		await user.click(screen.getByRole('option', { name: /Gómez, Beto/ }))
		await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))

		expect(create).toHaveBeenCalledWith({ loteoId: 'loteo-1', loteId: lot.id, clienteId: 'client-1', vendedorId: 'seller-1' }, expect.any(String))
		expect(onCreated).toHaveBeenCalledTimes(1)
		expect(reset).toHaveBeenCalledTimes(1)
		expect(screen.queryByRole('heading', { name: /Nueva reserva/ })).not.toBeInTheDocument()
	})

	it('keeps the dialog open when creation fails and shows loading errors', async () => {
		const user = userEvent.setup()
		const onCreated = vi.fn()
		configureMocks({ clientsError: 'No se pudieron cargar clientes', sellersError: 'No se pudieron cargar vendedores' })
		render(<ReserveLotDialog accessToken="token" loteoId="loteo-1" lote={lot} clients={[client]} clientsError="No se pudieron cargar clientes" onCreated={onCreated} />)
		await user.click(screen.getByRole('button', { name: 'Reservar lote' }))

		expect(screen.getByRole('heading', { name: 'Nueva reserva · Lote 7' })).toBeInTheDocument()
		expect(screen.getByText('No se pudieron cargar clientes')).toBeInTheDocument()
		expect(screen.getByText('ID: lot-12345678')).toBeInTheDocument()
		await user.click(screen.getByRole('button', { name: 'Cancelar' }))
		expect(screen.queryByRole('heading', { name: /Nueva reserva/ })).not.toBeInTheDocument()
	})

	it('disables reservation and explains the missing lot data', () => {
		configureMocks()
		render(<ReserveLotDialog accessToken="token" loteoId="loteo-1" lote={fallbackLot} onCreated={vi.fn()} />)

		expect(screen.getByRole('button', { name: 'Reservar lote' })).toBeDisabled()
		expect(screen.getByRole('tooltip')).toHaveTextContent('Completá el número y el precio del lote')
		expect(screen.getByTitle('Completá el número y el precio del lote para habilitar la reserva.')).toBeInTheDocument()

		const missingPriceLot = { ...lot, precio: null }
		const { unmount } = render(
			<ReserveLotDialog accessToken="token" loteoId="loteo-1" lote={missingPriceLot} onCreated={vi.fn()} />,
		)

		expect(screen.getAllByRole('button', { name: 'Reservar lote' })[1]).toBeDisabled()
		expect(screen.getAllByRole('tooltip')[1]).toHaveTextContent('Completá el precio del lote')
		unmount()
	})

	it('does not close when the reservation API rejects the creation', async () => {
		const user = userEvent.setup()
		const onCreated = vi.fn()
		const create = vi.fn().mockResolvedValue(null)
		configureMocks({ create })
		render(<ReserveLotDialog accessToken="token" loteoId="loteo-1" lote={lot} clients={[client]} onCreated={onCreated} />)
		await user.click(screen.getByRole('button', { name: 'Reservar lote' }))
		await user.click(screen.getByRole('combobox', { name: 'Cliente' }))
		await user.click(screen.getByRole('option', { name: /Pérez, Ana/ }))
		await user.click(screen.getByRole('combobox', { name: 'Vendedor' }))
		await user.click(screen.getByRole('option', { name: /Gómez, Beto/ }))
		await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))

		expect(create).toHaveBeenCalled()
		expect(onCreated).not.toHaveBeenCalled()
		expect(screen.getByRole('heading', { name: 'Nueva reserva · Lote 7' })).toBeInTheDocument()
	})

	it('keeps the same idempotency key after an uncertain attempt is closed', async () => {
		const user = userEvent.setup()
		const create = vi.fn().mockResolvedValue(null)
		configureMocks({ create })
		render(<ReserveLotDialog accessToken="token" loteoId="loteo-1" lote={lot} clients={[client]} onCreated={vi.fn()} />)

		async function submit() {
			await user.click(screen.getByRole('button', { name: 'Reservar lote' }))
			await user.click(screen.getByRole('combobox', { name: 'Cliente' }))
			await user.click(screen.getByRole('option', { name: /Pérez, Ana/ }))
			await user.click(screen.getByRole('combobox', { name: 'Vendedor' }))
			await user.click(screen.getByRole('option', { name: /Gómez, Beto/ }))
			await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))
		}

		await submit()
		await user.click(screen.getByRole('button', { name: 'Cancelar' }))
		await user.click(screen.getByRole('button', { name: 'Reservar lote' }))
		await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))
		expect(create.mock.calls[0][1]).toBe(create.mock.calls[1][1])
		expect(create.mock.calls[1][0]).toEqual(create.mock.calls[0][0])
	})
})
