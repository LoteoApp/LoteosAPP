import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import CreateClientDialog from './CreateClientDialog'
import type { Cliente } from '../types'

const createClientMock = vi.fn()

vi.mock('../api/clients', () => ({ createClient: (...args: unknown[]) => createClientMock(...args) }))

const existingClient: Cliente = {
	id: 'client-existing', nombre: 'Ana', apellido: 'Pérez', dni: '30111222', celular: '', email: '',
}

function DialogHarness({ clients = [existingClient], onCreated = vi.fn() }: {
  clients?: Cliente[]
  onCreated?: (client: Cliente) => void
}) {
  const [open, setOpen] = useState(false)
  return (
    <CreateClientDialog
      accessToken="test-token"
      clients={clients}
      open={open}
      onOpenChange={setOpen}
      onCreated={onCreated}
      trigger={<button type="button">Nuevo cliente</button>}
    />
  )
}

afterEach(() => vi.clearAllMocks())

describe('CreateClientDialog', () => {
  it('validates required fields and duplicate DNI before creating a client', async () => {
    const user = userEvent.setup()
    const onCreated = vi.fn()
    createClientMock.mockResolvedValue({ ...existingClient, id: 'client-new', dni: '30111223' })
    render(<DialogHarness onCreated={onCreated} />)

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Completá nombre, apellido y DNI.')

    await user.type(screen.getByLabelText('Nombre'), 'Ana')
    await user.type(screen.getByLabelText('Apellido'), 'Pérez')
    await user.type(screen.getByLabelText('DNI'), existingClient.dni)
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Ya existe un cliente con ese DNI.')

    await user.clear(screen.getByLabelText('DNI'))
    await user.type(screen.getByLabelText('DNI'), '30111223')
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))

    expect(await screen.findByRole('button', { name: 'Nuevo cliente' })).toBeInTheDocument()
    expect(createClientMock).toHaveBeenCalledWith('test-token', {
      nombre: 'Ana', apellido: 'Pérez', dni: '30111223', celular: '', email: '',
    })
    expect(onCreated).toHaveBeenCalledWith(expect.objectContaining({ id: 'client-new' }))
    expect(screen.queryByRole('heading', { name: 'Nuevo cliente' })).not.toBeInTheDocument()
  })

  it('shows create errors, clears them on reopen, and closes from the form', async () => {
    const user = userEvent.setup()
    createClientMock.mockRejectedValueOnce('offline').mockRejectedValueOnce(new Error('El DNI ya está en uso'))
    render(<DialogHarness clients={[]} />)

    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    await user.type(screen.getByLabelText('Nombre'), 'Juan')
    await user.type(screen.getByLabelText('Apellido'), 'García')
    await user.type(screen.getByLabelText('DNI'), '20111222')
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))
    expect(await screen.findByText('No se pudo crear el cliente.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Cerrar alta de cliente' }))
    await user.click(screen.getByRole('button', { name: 'Nuevo cliente' }))
    expect(screen.queryByText('No se pudo crear el cliente.')).not.toBeInTheDocument()

    await user.type(screen.getByLabelText('Nombre'), 'Juan')
    await user.type(screen.getByLabelText('Apellido'), 'García')
    await user.type(screen.getByLabelText('DNI'), '20111222')
    await user.click(screen.getByRole('button', { name: 'Crear cliente' }))
    expect(await screen.findByText('El DNI ya está en uso')).toBeInTheDocument()

	await user.click(screen.getByRole('button', { name: 'Cancelar' }))
    expect(screen.queryByRole('heading', { name: 'Nuevo cliente' })).not.toBeInTheDocument()
  })
})
