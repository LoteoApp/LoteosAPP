import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Link, MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import ReservationDetailsPage from './ReservationDetailsPage'
import type { Reservation } from '../types'

const getReservationMock = vi.fn()
const cancelMock = vi.fn()

vi.mock('../api/reservations', () => ({
	getReservation: (...args: unknown[]) => getReservationMock(...args),
}))

vi.mock('../hooks/use-reservation-mutations', () => ({
	useReservationMutations: () => ({ cancel: cancelMock, isSubmitting: false, error: null, reset: vi.fn() }),
}))

function reservation(overrides: Partial<Reservation> = {}): Reservation {
	return {
		id: 'reservation-1', loteoId: 'loteo-1', loteoNombre: 'Las Acacias', loteId: 'lot-1', loteNumero: '7',
		cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
		vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrador' },
		usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
		estado: 'activa', fechaVencimiento: '2026-09-21T12:00:00Z', fechaCreacion: '2026-09-06T12:00:00Z', fechaModificacion: '2026-09-06T12:00:00Z', historial: [],
		...overrides,
	}
}

function renderPage(path = '/reservas/reservation-1') {
	render(
		<MemoryRouter initialEntries={[path]}>
			<Routes>
				<Route path="/reservas/:id" element={<ReservationDetailsPage accessToken="token" />} />
			</Routes>
		</MemoryRouter>,
	)
}

function Navigation() {
	return <Link to="/reservas/reservation-2">Ir a otra reserva</Link>
}

afterEach(() => {
	getReservationMock.mockReset()
	cancelMock.mockReset()
})

describe('ReservationDetailsPage', () => {
	it('does not request details before the session token is available', async () => {
		render(
			<MemoryRouter initialEntries={['/reservas/reservation-1']}>
				<Routes><Route path="/reservas/:id" element={<ReservationDetailsPage />} /></Routes>
			</MemoryRouter>,
		)
		await Promise.resolve()
		expect(getReservationMock).not.toHaveBeenCalled()
	})

	it('loads an active reservation and refreshes it after cancellation', async () => {
		const current = reservation()
		getReservationMock.mockResolvedValue(current)
		cancelMock.mockResolvedValue({ ...current, estado: 'cancelada' })
		const user = userEvent.setup()
		renderPage()

		expect(await screen.findByText('Las Acacias')).toBeInTheDocument()
		await user.click(screen.getByRole('button', { name: 'Cancelar' }))
		await user.type(screen.getByLabelText('Justificación de la cancelación'), 'Cliente desistió')
		await user.click(screen.getByRole('button', { name: 'Cancelar reserva' }))
		await waitFor(() => expect(cancelMock).toHaveBeenCalledWith('reservation-1', 'Cliente desistió'))
		expect(screen.getByText('Cancelada')).toBeInTheDocument()
		expect(screen.queryByRole('button', { name: 'Cancelar reserva' })).not.toBeInTheDocument()
	})

	it('shows the backend error', async () => {
		getReservationMock.mockRejectedValue(new Error('Reserva no encontrada'))
		renderPage('/reservas/missing')
		expect(await screen.findByText('No se pudo cargar la reserva')).toBeInTheDocument()
		expect(screen.getByText('Reserva no encontrada')).toBeInTheDocument()
	})

	it('uses a generic message for an unknown loading failure', async () => {
		getReservationMock.mockRejectedValue('unknown failure')
		renderPage()
		expect(await screen.findByText('No se pudo cargar la reserva.')).toBeInTheDocument()
	})

	it('offers a return link when the response is empty and hides cancellation for terminal states', async () => {
		getReservationMock.mockResolvedValueOnce(null as unknown as Reservation)
		renderPage()
		expect(await screen.findByRole('link', { name: 'Volver a reservas' })).toHaveAttribute('href', '/reservas')

		getReservationMock.mockResolvedValueOnce(reservation({ estado: 'vencida' }))
		renderPage()
		expect(await screen.findByText('Vencida')).toBeInTheDocument()
		expect(screen.queryByRole('button', { name: 'Cancelar reserva' })).not.toBeInTheDocument()
	})

	it('does not apply a cancellation response after navigating to another reservation', async () => {
		const current = reservation()
		const other = reservation({ id: 'reservation-2', loteoNombre: 'Los Aromos', loteId: 'lot-2' })
		getReservationMock.mockImplementation((_token: string, id: string) => Promise.resolve(id === other.id ? other : current))
		let resolveCancel: (value: Reservation) => void = () => undefined
		cancelMock.mockImplementation(() => new Promise<Reservation>((resolve) => { resolveCancel = resolve }))
		const user = userEvent.setup()
		render(
			<MemoryRouter initialEntries={['/reservas/reservation-1']}>
				<Navigation />
				<Routes><Route path="/reservas/:id" element={<ReservationDetailsPage accessToken="token" />} /></Routes>
			</MemoryRouter>,
		)
		await screen.findByText('Las Acacias')
		await user.click(screen.getByRole('button', { name: 'Cancelar' }))
		await user.type(screen.getByLabelText('Justificación de la cancelación'), 'Cliente desistió')
		await user.click(screen.getByRole('button', { name: 'Cancelar reserva' }))
		await user.click(screen.getByRole('button', { name: 'Cerrar cancelación' }))
		await user.click(screen.getByRole('link', { name: 'Ir a otra reserva' }))
		await screen.findByText('Los Aromos')
		resolveCancel({ ...current, estado: 'cancelada' })
		await waitFor(() => expect(screen.getByText('Los Aromos')).toBeInTheDocument())
		expect(screen.queryByText('Las Acacias')).not.toBeInTheDocument()
	})
})
