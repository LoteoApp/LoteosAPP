import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { Reservation } from '../types'
import CancelReservationDialog from './CancelReservationDialog'

const reservation: Reservation = {
  id: 'reservation-1',
  loteoId: 'loteo-1',
  loteoNombre: 'Las Acacias',
  loteId: 'lot-12345678',
  loteNumero: '7',
  cliente: { id: 'client-1', nombre: 'Ana', apellido: 'Pérez', dni: '30111222' },
  vendedor: { id: 'seller-1', nombre: 'Beto', apellido: 'Gómez', rol: 'administrador' },
  usuarioAlta: { id: 'actor-1', nombre: 'Carla', apellido: 'López', rol: 'administrativo' },
  estado: 'activa',
  fechaVencimiento: '2026-09-21T12:00:00Z',
  fechaCreacion: '2026-09-06T12:00:00Z',
  fechaModificacion: '2026-09-06T12:00:00Z',
}

describe('CancelReservationDialog', () => {
  it('identifies the reservation and closes after a successful cancellation', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn().mockResolvedValue(true)
    const onClose = vi.fn()

    render(<CancelReservationDialog reservation={reservation} isSubmitting={false} error={null} onSubmit={onSubmit} onClose={onClose} />)

    expect(screen.getByRole('heading', { name: 'Cancelar reserva' })).toBeInTheDocument()
    expect(screen.getByText('Las Acacias · Lote 7')).toBeInTheDocument()
    expect(screen.getByText('Cliente: Ana Pérez')).toBeInTheDocument()
    await user.type(screen.getByLabelText('Justificación de la cancelación'), ' Cliente desistió ')
    await user.click(screen.getByRole('button', { name: 'Cancelar reserva' }))

    expect(onSubmit).toHaveBeenCalledWith('Cliente desistió')
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('validates an empty reason and keeps the dialog open after an error', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn().mockResolvedValue(false)
    const onClose = vi.fn()

    render(<CancelReservationDialog reservation={{ ...reservation, loteNumero: '' }} isSubmitting={false} error="No se pudo cancelar" onSubmit={onSubmit} onClose={onClose} />)

    expect(screen.getByText('Las Acacias · Lote lot-1234')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Cancelar reserva' }))
    expect(screen.getByText('La justificación es obligatoria.')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()

    await user.type(screen.getByLabelText('Justificación de la cancelación'), 'Motivo')
    await user.click(screen.getByRole('button', { name: 'Cancelar reserva' }))
    expect(onSubmit).toHaveBeenCalledWith('Motivo')
    expect(onClose).not.toHaveBeenCalled()
    expect(screen.getByText('No se pudo cancelar')).toBeInTheDocument()
  })

  it('closes when the close button is pressed', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<CancelReservationDialog reservation={reservation} isSubmitting={false} error={null} onSubmit={vi.fn()} onClose={onClose} />)

    await user.click(screen.getByRole('button', { name: 'Cerrar cancelación' }))

    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
