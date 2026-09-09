import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ReservationsPage from './ReservationsPage'
import type { Reservation } from '../types'

const useReservationMutationsMock = vi.hoisted(() => vi.fn())
const useReservationsMock = vi.hoisted(() => vi.fn())

vi.mock('../hooks/use-reservation-mutations', () => ({ useReservationMutations: useReservationMutationsMock }))
vi.mock('../hooks/use-reservations', () => ({ useReservations: useReservationsMock }))

function reservation(overrides: Partial<Reservation> = {}): Reservation {
	return {
		id: 'reservation-1', loteoId: 'loteo-1', loteoNombre: 'Las Acacias', loteId: 'lot-1', loteNumero: '7',
		cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
		vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrador' },
		usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' }, estado: 'activa',
		fechaVencimiento: '2026-09-21T12:00:00Z', fechaCreacion: '2026-09-06T12:00:00Z', fechaModificacion: '2026-09-06T12:00:00Z', historial: [],
		...overrides,
	}
}

function defaultMocks() {
	const refresh = vi.fn()
	useReservationMutationsMock.mockReturnValue({ isSubmitting: false, error: null, cancel: vi.fn(), reset: vi.fn() })
	useReservationsMock.mockReturnValue({ page: { reservas: [], pagina: 1, porPagina: 25, total: 0, paginas: 0 }, isLoading: false, error: null, refresh })
	return { refresh }
}

beforeEach(() => {
	vi.clearAllMocks()
})

describe('ReservationsPage', () => {
	it('directs reservation creation to the lot visor', () => {
		defaultMocks()
		render(<MemoryRouter><ReservationsPage /></MemoryRouter>)
		expect(screen.getByRole('heading', { name: 'Reservas' })).toBeInTheDocument()
		expect(screen.getByRole('link', { name: 'Abrir visor de lotes' })).toHaveAttribute('href', '/lotes')
		expect(screen.queryByRole('heading', { name: 'Nueva reserva' })).not.toBeInTheDocument()
		expect(screen.queryByRole('combobox', { name: 'Loteo' })).not.toBeInTheDocument()
	})

	it('cancels a reservation from the list', async () => {
		const user = userEvent.setup()
		const { refresh } = defaultMocks()
		const current = reservation()
		const cancel = vi.fn().mockResolvedValue({ ...current, estado: 'cancelada' })
		useReservationMutationsMock.mockReturnValue({ isSubmitting: false, error: null, cancel, reset: vi.fn() })
		useReservationsMock.mockReturnValue({ page: { reservas: [current], pagina: 1, porPagina: 25, total: 1, paginas: 1 }, isLoading: false, error: null, refresh })

		render(<MemoryRouter><ReservationsPage /></MemoryRouter>)
		await user.click(screen.getByRole('button', { name: 'Cancelar' }))
		await user.type(screen.getByLabelText('Justificación de la cancelación'), 'Cliente desistió')
		await user.click(screen.getByRole('button', { name: 'Cancelar reserva' }))
		expect(cancel).toHaveBeenCalledWith('reservation-1', 'Cliente desistió')
		expect(refresh).toHaveBeenCalled()
	})

	it('shows loading and server errors without hiding the reservation controls', () => {
		defaultMocks()
		useReservationsMock.mockReturnValue({ page: { reservas: [], pagina: 1, porPagina: 25, total: 0, paginas: 0 }, isLoading: true, error: 'No se pudo cargar reservas', refresh: vi.fn() })
		render(<MemoryRouter><ReservationsPage /></MemoryRouter>)
		expect(screen.getByText('No se pudo cargar reservas')).toBeInTheDocument()
	})

	it('changes pages using the server pagination controls', async () => {
		const user = userEvent.setup()
		defaultMocks()
		useReservationsMock.mockImplementation((_token, filters) => ({
			page: {
				reservas: [reservation()],
				pagina: filters.pagina ?? 1,
				porPagina: 25,
				total: 51,
				paginas: 3,
			},
			isLoading: false,
			error: null,
			refresh: vi.fn(),
		}))

		render(<MemoryRouter><ReservationsPage /></MemoryRouter>)
		expect(screen.getByText('Página 1 de 3 · 51 reservas')).toBeInTheDocument()
		await user.click(screen.getByRole('button', { name: 'Siguiente' }))
		expect(screen.getByText('Página 2 de 3 · 51 reservas')).toBeInTheDocument()
		await user.click(screen.getByRole('button', { name: 'Anterior' }))

		const calls = useReservationsMock.mock.calls
		expect(calls.some(([, filters]) => filters.pagina === 1)).toBe(true)
		expect(calls.some(([, filters]) => filters.pagina === 2)).toBe(true)
	})

	it('returns to the first page when filters change', async () => {
		const user = userEvent.setup()
		defaultMocks()
		useReservationsMock.mockImplementation((_token, filters) => ({
			page: {
				reservas: [reservation()],
				pagina: filters.pagina ?? 1,
				porPagina: 25,
				total: 51,
				paginas: 3,
			},
			isLoading: false,
			error: null,
			refresh: vi.fn(),
		}))

		render(<MemoryRouter><ReservationsPage /></MemoryRouter>)
		await user.click(screen.getByRole('button', { name: 'Siguiente' }))
		await user.type(screen.getByRole('searchbox', { name: 'Buscar' }), 'Ana')

		expect(useReservationsMock).toHaveBeenLastCalledWith('', {
			q: 'Ana',
			estado: undefined,
			pagina: 1,
		})

		await user.click(screen.getByRole('button', { name: 'Siguiente' }))
		await user.click(screen.getByRole('combobox', { name: 'Estado' }))
		await user.click(screen.getByRole('option', { name: 'Canceladas' }))

		expect(useReservationsMock).toHaveBeenLastCalledWith('', {
			q: 'Ana',
			estado: 'cancelada',
			pagina: 1,
		})
	})
})
