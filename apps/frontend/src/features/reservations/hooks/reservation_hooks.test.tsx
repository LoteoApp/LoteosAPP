import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cancelReservation, createReservation, listEligibleSellers, listReservations } from '../api/reservations'
import { useEligibleSellers } from './use-eligible-sellers'
import { useReservationMutations } from './use-reservation-mutations'
import { useReservations } from './use-reservations'

vi.mock('../api/reservations', () => ({
	cancelReservation: vi.fn(),
	createReservation: vi.fn(),
	listEligibleSellers: vi.fn(),
	listReservations: vi.fn(),
}))

const listReservationsMock = vi.mocked(listReservations)
const listEligibleSellersMock = vi.mocked(listEligibleSellers)
const createReservationMock = vi.mocked(createReservation)
const cancelReservationMock = vi.mocked(cancelReservation)

const page = { reservas: [], pagina: 1, porPagina: 25, total: 0, paginas: 0 }
const created = {
	id: 'reservation-1', loteoId: 'loteo-1', loteoNombre: 'Las Acacias', loteId: 'lot-1', loteNumero: '7',
	cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
	vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrador' },
	usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
	estado: 'activa' as const, fechaVencimiento: '2026-09-21T12:00:00Z', fechaCreacion: '2026-09-06T12:00:00Z', fechaModificacion: '2026-09-06T12:00:00Z',
}

afterEach(() => {
	listReservationsMock.mockReset()
	listEligibleSellersMock.mockReset()
	createReservationMock.mockReset()
	cancelReservationMock.mockReset()
})

describe('reservation hooks', () => {
	it('loads reservations and refreshes the current query', async () => {
		listReservationsMock.mockResolvedValue(page)
		const { result } = renderHook(() => useReservations('token', { q: 'Ana' }))
		await waitFor(() => expect(result.current.isLoading).toBe(false))
		expect(result.current.page).toEqual(page)
		expect(listReservationsMock).toHaveBeenCalledWith('token', { q: 'Ana' }, expect.any(AbortSignal))

		act(() => result.current.refresh())
		await waitFor(() => expect(listReservationsMock).toHaveBeenCalledTimes(2))
	})

	it('keeps the previous page while a new query loads', async () => {
		const loadedPage = { ...page, reservas: [created], total: 1, paginas: 1 }
		let resolveNext: ((value: typeof page) => void) | undefined
		listReservationsMock
			.mockResolvedValueOnce(loadedPage)
			.mockImplementationOnce(() => new Promise((resolve) => { resolveNext = resolve }))
		const { result, rerender } = renderHook(({ search }) => useReservations('token', { q: search }), {
			initialProps: { search: 'old' },
		})
		await waitFor(() => expect(result.current.page).toEqual(loadedPage))

		rerender({ search: 'new' })
		await waitFor(() => expect(result.current.isLoading).toBe(true))
		expect(result.current.page).toEqual(loadedPage)

		resolveNext?.(page)
		await waitFor(() => expect(result.current.isLoading).toBe(false))
	})

	it('debounces search changes before requesting reservations', async () => {
		listReservationsMock.mockResolvedValue(page)
		const { rerender } = renderHook(({ search }) => useReservations('token', { q: search }), {
			initialProps: { search: '' },
		})
		await waitFor(() => expect(listReservationsMock).toHaveBeenCalledTimes(1))

		rerender({ search: 'A' })
		rerender({ search: 'An' })
		rerender({ search: 'Ana' })
		expect(listReservationsMock).toHaveBeenCalledTimes(1)
		await waitFor(() => expect(listReservationsMock).toHaveBeenCalledTimes(2))
		expect(listReservationsMock).toHaveBeenLastCalledWith('token', { q: 'Ana' }, expect.any(AbortSignal))
	})

	it('prepends a created reservation without reloading the list', async () => {
		listReservationsMock.mockResolvedValue(page)
		const { result } = renderHook(() => useReservations('token', {}))
		await waitFor(() => expect(result.current.isLoading).toBe(false))

		act(() => result.current.prepend(created))

		expect(result.current.page.reservas).toEqual([created])
		expect(result.current.page.total).toBe(1)
		expect(listReservationsMock).toHaveBeenCalledTimes(1)
	})

	it('does not prepend a reservation outside the active filters', async () => {
		for (const filters of [
			{ pagina: 2 },
			{ estado: 'vencida' as const },
			{ loteoId: 'other-loteo' },
			{ q: 'missing' },
		]) {
			listReservationsMock.mockResolvedValue(page)
			const { result, unmount } = renderHook(() => useReservations('token', filters))
			await waitFor(() => expect(result.current.isLoading).toBe(false))

			act(() => result.current.prepend(created))

			expect(result.current.page).toEqual(page)
			unmount()
			listReservationsMock.mockReset()
		}
	})

	it('prepends a reservation matching the search filter', async () => {
		listReservationsMock.mockResolvedValue(page)
		const { result } = renderHook(() => useReservations('token', { q: 'ANA' }))
		await waitFor(() => expect(result.current.isLoading).toBe(false))

		act(() => result.current.prepend(created))

		expect(result.current.page.reservas).toEqual([created])
	})

	it('resets a failed reservation load to an empty page', async () => {
		listReservationsMock.mockRejectedValue(new Error('No autorizado'))
		const { result } = renderHook(() => useReservations('token', {}))
		await waitFor(() => expect(result.current.error).toBe('No autorizado'))
		expect(result.current.page.reservas).toEqual([])
	})

	it('does not load reservations when the caller disables the query', async () => {
		const { result } = renderHook(() => useReservations('token', { loteoId: 'loteo-1' }, { enabled: false }))
		await waitFor(() => expect(result.current.isLoading).toBe(false))
		expect(listReservationsMock).not.toHaveBeenCalled()
		expect(result.current.page.reservas).toEqual([])
	})

	it('ignores a reservation response from a superseded request', async () => {
		let resolveFirst: ((value: typeof page) => void) | undefined
		listReservationsMock
			.mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve }))
			.mockResolvedValueOnce(page)
		const { rerender } = renderHook(({ search }) => useReservations('token', { q: search }), { initialProps: { search: 'old' } })
		rerender({ search: 'new' })
		resolveFirst?.(page)
		await waitFor(() => expect(listReservationsMock).toHaveBeenCalledTimes(2))
	})

	it('ignores an error from a superseded reservation request', async () => {
		let rejectFirst: ((reason?: unknown) => void) | undefined
		listReservationsMock
			.mockImplementationOnce(() => new Promise((_resolve, reject) => { rejectFirst = reject }))
			.mockResolvedValueOnce(page)
		const { rerender } = renderHook(({ search }) => useReservations('token', { q: search }), { initialProps: { search: 'old' } })
		rerender({ search: 'new' })
		rejectFirst?.(new Error('stale error'))
		await waitFor(() => expect(listReservationsMock).toHaveBeenCalledTimes(2))
	})

	it('loads sellers only for a selected loteo and reports errors', async () => {
		const { result, rerender } = renderHook(({ loteoId }) => useEligibleSellers('token', loteoId), { initialProps: { loteoId: '' } })
		expect(listEligibleSellersMock).not.toHaveBeenCalled()

		listEligibleSellersMock.mockResolvedValue([{ id: 'seller-1', nombre: 'Ana', apellido: 'Pérez', rol: 'administrador' }])
		rerender({ loteoId: 'loteo-1' })
		await waitFor(() => expect(result.current.sellers).toHaveLength(1))

		listEligibleSellersMock.mockRejectedValue(new Error('No se pudo cargar vendedores'))
		rerender({ loteoId: 'loteo-2' })
		await waitFor(() => expect(result.current.error).toBe('No se pudo cargar vendedores'))
	})

	it('ignores seller results from a superseded request', async () => {
		let resolveFirst: ((value: never[]) => void) | undefined
		listEligibleSellersMock
			.mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve }))
			.mockResolvedValueOnce([])
		const { rerender } = renderHook(({ loteoId }) => useEligibleSellers('token', loteoId), { initialProps: { loteoId: 'old' } })
		rerender({ loteoId: 'new' })
		resolveFirst?.([])
		await waitFor(() => expect(listEligibleSellersMock).toHaveBeenCalledTimes(2))
	})

	it('executes reservation mutations and exposes recoverable errors', async () => {
		createReservationMock.mockResolvedValue(created)
		cancelReservationMock.mockResolvedValue({ ...created, estado: 'cancelada' })
		const { result } = renderHook(() => useReservationMutations('token'))

		await act(async () => {
			await expect(result.current.create({ loteoId: 'loteo-1', loteId: 'lot-1', clienteId: 'client-1' }, 'key')).resolves.toEqual(created)
			await expect(result.current.cancel('reservation-1', 'Cliente desistió')).resolves.toMatchObject({ estado: 'cancelada' })
		})
		expect(createReservationMock).toHaveBeenCalledWith('token', { loteoId: 'loteo-1', loteId: 'lot-1', clienteId: 'client-1' }, 'key')

		createReservationMock.mockRejectedValue(new Error('Conflicto de disponibilidad'))
		await act(async () => {
			expect(await result.current.create({ loteoId: 'loteo-1', loteId: 'lot-1', clienteId: 'client-1' }, 'key-2')).toBeNull()
		})
		expect(result.current.error).toBe('Conflicto de disponibilidad')
		act(() => result.current.reset())
		expect(result.current.error).toBeNull()

		cancelReservationMock.mockRejectedValue(new Error('No se pudo cancelar'))
		await act(async () => {
			expect(await result.current.cancel('reservation-1', 'Motivo')).toBeNull()
		})
		expect(result.current.error).toBe('No se pudo cancelar')

		cancelReservationMock.mockRejectedValue('unknown failure')
		await act(async () => {
			expect(await result.current.cancel('reservation-1', 'Motivo')).toBeNull()
		})
		expect(result.current.error).toBe('Ocurrió un error inesperado.')
	})
})
