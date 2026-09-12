import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import CancelReservationForm from './CancelReservationForm'
import ReservationCreatePageSkeleton from './ReservationCreatePageSkeleton'
import ReservationDetails from './ReservationDetails'
import ReservationDetailsPageSkeleton from './ReservationDetailsPageSkeleton'
import ReservationFilters from './ReservationFilters'
import ReservationForm from './ReservationForm'
import ReservationStatusBadge from './ReservationStatusBadge'
import ReservationsList from './ReservationsList'
import ReservationsListSkeleton from './ReservationsListSkeleton'
import type { Reservation, ReservationClient, ReservationLot, ReservationLoteoOption, SellerOption } from '../types'

const lot: ReservationLot = {
	id: 'lot-1', numero: '7', estado: 'disponible', precio: 100000,
}

const loteo: ReservationLoteoOption = {
	id: 'loteo-1', nombre: 'Las Acacias',
}

const client: ReservationClient = {
	id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222',
}

const sellers: SellerOption[] = [
	{ id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrador' },
	{ id: 'seller-2', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
]

function reservation(overrides: Partial<Reservation> = {}): Reservation {
	return {
		id: 'reservation-1', loteoId: 'loteo-1', loteoNombre: 'Las Acacias', loteId: 'lot-1', loteNumero: '7',
		cliente: { id: client.id, nombre: client.nombre, apellido: client.apellido, dni: client.dni },
		vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrador' },
		usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
		estado: 'activa', puedeCancelar: true, fechaVencimiento: '2026-09-21T12:00:00Z', fechaCreacion: '2026-09-06T12:00:00Z',
		fechaModificacion: '2026-09-06T12:00:00Z', historial: [], ...overrides,
	}
}

function renderForm(overrides: Partial<React.ComponentProps<typeof ReservationForm>> = {}) {
	const onSubmit = vi.fn<(values: { loteoId: string; loteId: string; clienteId: string; vendedorId?: string }, key: string) => Promise<boolean>>().mockResolvedValue(true)
	const props: React.ComponentProps<typeof ReservationForm> = {
		loteos: [loteo], selectedLoteoId: loteo.id, selectedLoteId: lot.id, lots: [lot], clients: [client],
		sellers, isLoadingSellers: false, isSubmitting: false, error: null,
		onLoteoChange: vi.fn(), onLoteChange: vi.fn(), onSubmit, ...overrides,
	}
	render(<ReservationForm {...props} />)
	return { onSubmit }
}

describe('reservation components', () => {
	it('shows every reservation state with readable text', () => {
		render(<div>{(['activa', 'vencida', 'cancelada', 'convertida'] as const).map((state) => <ReservationStatusBadge key={state} state={state} />)}</div>)
		expect(screen.getByText('Activa')).toBeInTheDocument()
		expect(screen.getByText('Vencida')).toBeInTheDocument()
		expect(screen.getByText('Cancelada')).toBeInTheDocument()
		expect(screen.getByText('Convertida')).toBeInTheDocument()
	})

	it('filters reservations through accessible controls', async () => {
		const user = userEvent.setup()
		const onSearchChange = vi.fn()
		const onStateChange = vi.fn()
		render(<ReservationFilters search="Ana" state="" onSearchChange={onSearchChange} onStateChange={onStateChange} />)
		await user.type(screen.getByRole('searchbox', { name: 'Buscar' }), ' Pérez')
		expect(onSearchChange).toHaveBeenCalled()
		await user.click(screen.getByRole('combobox', { name: 'Estado' }))
		await user.click(screen.getByRole('option', { name: 'Canceladas' }))
		expect(onStateChange).toHaveBeenCalledWith('cancelada')
	})

	it('renders an empty list and only offers cancellation for active reservations', async () => {
		const onCancel = vi.fn()
		render(<MemoryRouter><ReservationsList reservations={[]} onCancel={onCancel} /></MemoryRouter>)
		expect(screen.getByText('No hay reservas que coincidan con los filtros.')).toBeInTheDocument()

		cleanup()
		render(<MemoryRouter><ReservationsList reservations={[reservation(), reservation({ id: 'reservation-2', loteNumero: '8', estado: 'vencida' })]} onCancel={onCancel} /></MemoryRouter>)
		expect(screen.getAllByRole('link', { name: 'Las Acacias' })[0]).toHaveAttribute('href', '/reservas/reservation-1')
		expect(screen.getByRole('link', { name: 'Ver detalle de la reserva del lote 7 en Las Acacias' })).toHaveAttribute('href', '/reservas/reservation-1')
		expect(screen.getByRole('link', { name: 'Ver detalle de la reserva del lote 8 en Las Acacias' })).toHaveAttribute('href', '/reservas/reservation-2')
		expect(screen.getByRole('button', { name: 'Cancelar' })).toBeInTheDocument()
		await userEvent.click(screen.getByRole('button', { name: 'Cancelar' }))
		expect(onCancel).toHaveBeenCalledWith(expect.objectContaining({ id: 'reservation-1' }))

		cleanup()
		render(<MemoryRouter><ReservationsList reservations={[reservation({ puedeCancelar: false })]} onCancel={onCancel} /></MemoryRouter>)
		expect(screen.queryByRole('button', { name: 'Cancelar' })).not.toBeInTheDocument()
		expect(screen.getByRole('link', { name: 'Ver detalle de la reserva del lote 7 en Las Acacias' })).toHaveAttribute('href', '/reservas/reservation-1')
		expect(screen.getByRole('link', { name: 'Ver loteo Las Acacias' })).toHaveAttribute('href', '/lotes/loteo-1')
	})

	it('renders reservation details and its audit history', () => {
		render(<ReservationDetails reservation={reservation({
		loteNumero: '', historial: [
			{ id: 'history-1', estado: 'activa', fecha: '2026-09-06T12:00:00Z', usuario: reservation().usuarioAlta },
			{ id: 'history-2', estado: 'cancelada', razon: 'Cliente desistió', fecha: '2026-09-07T12:00:00Z' },
		],
	})} />)
		expect(screen.getByText('Lote sin número')).toBeInTheDocument()
		expect(screen.getByText('Cliente desistió')).toBeInTheDocument()
		expect(screen.getByText('Por Carla López')).toBeInTheDocument()
		expect(screen.getByText('Historial')).toBeInTheDocument()
	})

	it('requires a cancellation reason and clears it after success', async () => {
		const user = userEvent.setup()
		const onSubmit = vi.fn<(reason: string) => Promise<boolean>>().mockResolvedValue(true)
		render(<CancelReservationForm isSubmitting={false} error={null} onSubmit={onSubmit} />)
		await user.click(screen.getByRole('button', { name: 'Cancelar reserva' }))
		expect(screen.getByText('La justificación es obligatoria.')).toBeInTheDocument()
		await user.type(screen.getByLabelText('Justificación de la cancelación'), '  Cliente desistió  ')
		await user.click(screen.getByRole('button', { name: 'Cancelar reserva' }))
		expect(onSubmit).toHaveBeenCalledWith('Cliente desistió')
		expect(screen.getByLabelText('Justificación de la cancelación')).toHaveValue('')
	})

	it('keeps the typed reason when cancellation fails and displays server errors', async () => {
		const user = userEvent.setup()
		const onSubmit = vi.fn<(reason: string) => Promise<boolean>>().mockResolvedValue(false)
		render(<CancelReservationForm isSubmitting={false} error="No se pudo cancelar" onSubmit={onSubmit} />)
		await user.type(screen.getByLabelText('Justificación de la cancelación'), 'Motivo')
		await user.click(screen.getByRole('button', { name: 'Cancelar reserva' }))
		expect(screen.getByText('No se pudo cancelar')).toBeInTheDocument()
		expect(screen.getByLabelText('Justificación de la cancelación')).toHaveValue('Motivo')
	})

	it('validates, submits and resets the reservation form', async () => {
		const user = userEvent.setup()
		const { onSubmit } = renderForm()
		await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))
		expect(screen.getByText('Seleccioná un cliente.')).not.toBeNull()

	await user.click(screen.getByRole('combobox', { name: 'Cliente' }))
	await user.click(await screen.findByRole('option', { name: /Pérez, Ana/ }))
	await user.click(screen.getByRole('combobox', { name: 'Vendedor' }))
	await user.click(await screen.findByRole('option', { name: /Gómez, Beto/ }))
		await user.click(screen.getByRole('button', { name: 'Confirmar reserva' }))
		expect(onSubmit).toHaveBeenCalledWith({ loteoId: 'loteo-1', loteId: 'lot-1', clienteId: 'client-1', vendedorId: 'seller-1' }, expect.any(String))
	})

	it('supports a fixed lot target, cancel action and disabled empty states', async () => {
		const onCancel = vi.fn()
		renderForm({ fixedTarget: true, onCancel, isSubmitting: true })
		expect(screen.getByText('Lote 7')).toBeInTheDocument()
		expect(screen.getByRole('button', { name: 'Cancelar' })).toBeInTheDocument()
		expect(screen.getByRole('button', { name: 'Guardando…' })).toBeDisabled()
		await userEvent.click(screen.getByRole('button', { name: 'Cancelar' }))
		expect(onCancel).toHaveBeenCalled()

		renderForm({ lots: [], clients: [], sellers: [], isLoadingSellers: true })
		expect(screen.getByText('No hay clientes activos')).toBeInTheDocument()
		expect(screen.getByText('Cargando vendedores…')).toBeInTheDocument()
	})

})

describe('ReservationCreatePageSkeleton', () => {
	it('announces a single loading region without exposing placeholder content', () => {
		render(<ReservationCreatePageSkeleton />)
		const placeholders = screen.getAllByRole('status')
		expect(placeholders).toHaveLength(1)
		expect(placeholders[0]).toHaveAccessibleName('Cargando los datos para crear la reserva…')
		expect(placeholders[0]).toHaveAttribute('aria-live', 'polite')
		expect(screen.queryByRole('button')).not.toBeInTheDocument()
		expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
	})
})

describe('ReservationDetailsPageSkeleton', () => {
	it('announces a single loading region without exposing placeholder content', () => {
		render(<ReservationDetailsPageSkeleton />)
		const placeholders = screen.getAllByRole('status')
		expect(placeholders).toHaveLength(1)
		expect(placeholders[0]).toHaveAccessibleName('Cargando el detalle de la reserva…')
		expect(placeholders[0]).toHaveAttribute('aria-live', 'polite')
		expect(screen.queryByRole('button')).not.toBeInTheDocument()
		expect(screen.queryByRole('list')).not.toBeInTheDocument()
	})
})

describe('ReservationsListSkeleton', () => {
	it('announces a single loading region with the requested amount of placeholder rows', () => {
		const { container } = render(<ReservationsListSkeleton rows={2} />)
		const placeholders = screen.getAllByRole('status')
		expect(placeholders).toHaveLength(1)
		expect(placeholders[0]).toHaveAccessibleName('Cargando reservas…')
		expect(placeholders[0]).toHaveAttribute('aria-live', 'polite')
		expect(container.querySelectorAll('li')).toHaveLength(2)
		expect(screen.queryByRole('button')).not.toBeInTheDocument()
		expect(screen.queryByRole('link')).not.toBeInTheDocument()
	})
})
