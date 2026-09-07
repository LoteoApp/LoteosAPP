import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import {
	cancelReservation,
	createReservation,
	getReservation,
	isReservationConflict,
	listEligibleSellers,
	listReservations,
} from './reservations'
import { RESERVATION_STATES } from '../types'

const apiFetchMock = vi.hoisted(() => vi.fn<(path: string, options?: Record<string, unknown>) => Promise<unknown>>())

vi.mock('../../../shared/api/client', async (importOriginal) => {
	const original = await importOriginal<typeof import('../../../shared/api/client')>()
	return { ...original, apiFetch: apiFetchMock }
})

function reservation(overrides: Record<string, unknown> = {}) {
	return {
		id: 'reservation-1',
		loteoId: 'loteo-1',
		loteoNombre: 'Las Acacias',
		loteId: 'lot-1',
		loteNumero: '7',
		cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
		vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrador' },
		usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
		estado: 'activa',
		fechaVencimiento: '2026-09-21T12:00:00Z',
		fechaCreacion: '2026-09-06T12:00:00Z',
		fechaModificacion: '2026-09-06T12:00:00Z',
		...overrides,
	}
}

afterEach(() => apiFetchMock.mockReset())

describe('reservation API', () => {
	it('exposes the complete reservation state vocabulary', () => {
		expect(RESERVATION_STATES).toEqual(['activa', 'vencida', 'cancelada', 'convertida'])
	})

	it('lists with encoded filters and validates the page', async () => {
		const page = { reservas: [reservation()], pagina: 2, porPagina: 10, total: 1, paginas: 1 }
		apiFetchMock.mockResolvedValue(page)

		await expect(listReservations('token', {
			estado: 'activa', loteoId: 'loteo 1', q: 'Ana Pérez', pagina: 2, porPagina: 10,
		})).resolves.toEqual(page)

		expect(apiFetchMock).toHaveBeenCalledWith(
			'/api/v1/reservas?estado=activa&loteoId=loteo+1&q=Ana+P%C3%A9rez&pagina=2&porPagina=10',
			{ token: 'token', signal: undefined },
		)
	})

	it('gets, creates and cancels a reservation with the expected requests', async () => {
		apiFetchMock
			.mockResolvedValueOnce(reservation())
			.mockResolvedValueOnce(reservation())
			.mockResolvedValueOnce(reservation({ estado: 'cancelada' }))

		await getReservation('token', 'reservation/1')
		await createReservation('token', { loteoId: 'loteo 1', loteId: 'lot/1', clienteId: 'client-1', vendedorId: 'seller-1' }, 'key-1')
		await cancelReservation('token', 'reservation/1', 'Cliente desistió')

		expect(apiFetchMock.mock.calls[0]).toEqual(['/api/v1/reservas/reservation%2F1', { token: 'token', signal: undefined }])
		expect(apiFetchMock.mock.calls[1]).toEqual([
			'/api/v1/loteos/loteo%201/lotes/lot%2F1/reservas',
			{
				method: 'POST', token: 'token',
				body: { clienteId: 'client-1', vendedorId: 'seller-1' },
				headers: { 'Idempotency-Key': 'key-1' },
			},
		])
		expect(apiFetchMock.mock.calls[2]).toEqual([
			'/api/v1/reservas/reservation%2F1/cancelar',
			{ method: 'POST', token: 'token', body: { razon: 'Cliente desistió' } },
		])
	})

	it('validates the seller catalog and rejects malformed responses', async () => {
		apiFetchMock.mockResolvedValueOnce({ vendedores: [{ id: 'seller-1', nombre: 'Ana', apellido: 'Pérez', rol: 'administrador' }] })
		await expect(listEligibleSellers('token', 'loteo/1')).resolves.toHaveLength(1)

		apiFetchMock.mockResolvedValueOnce({ vendedores: [{ id: 'seller-1' }] })
		await expect(listEligibleSellers('token', 'loteo/1')).rejects.toThrow('No se pudo completar la operación')
	})

	it('rejects malformed reservation payloads and identifies conflicts', async () => {
		apiFetchMock.mockResolvedValueOnce({ reservas: [] })
		await expect(listReservations('token')).rejects.toThrow('No se pudo completar la operación')

		apiFetchMock.mockResolvedValueOnce(reservation({ estado: 'desconocida' }))
		await expect(getReservation('token', 'reservation-1')).rejects.toThrow('No se pudo completar la operación')

		expect(isReservationConflict(new ApiError('occupied', 'reservation_already_active', 409))).toBe(true)
		expect(isReservationConflict(new ApiError('other', 'other_error', 400))).toBe(false)
		expect(isReservationConflict(new Error('other'))).toBe(false)
	})
})
