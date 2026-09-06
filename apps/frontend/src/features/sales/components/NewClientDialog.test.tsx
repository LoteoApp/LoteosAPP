import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import NewClientDialog from './NewClientDialog'

describe('NewClientDialog', () => {
  it('stays closed until it is opened', () => {
    render(<NewClientDialog open={false} onSubmit={vi.fn()} onClose={vi.fn()} />)

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('submits the trimmed values', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    render(<NewClientDialog open onSubmit={onSubmit} onClose={vi.fn()} />)

    await user.type(screen.getByLabelText('Nombre'), '  Luis  ')
    await user.type(screen.getByLabelText('Apellido'), ' Gómez ')
    await user.type(screen.getByLabelText('DNI'), ' 28999111 ')
    await user.type(screen.getByLabelText('Celular'), ' 3510000000 ')
    await user.type(screen.getByLabelText('Correo electrónico'), ' luis@example.com ')
    await user.click(screen.getByRole('button', { name: 'Guardar cliente' }))

    expect(onSubmit).toHaveBeenCalledWith({
      nombre: 'Luis',
      apellido: 'Gómez',
      dni: '28999111',
      celular: '3510000000',
      email: 'luis@example.com',
    })
  })

  it('closes and clears what was typed when the user cancels', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()

    render(<NewClientDialog open onSubmit={vi.fn()} onClose={onClose} />)

    await user.type(screen.getByLabelText('Nombre'), 'Luis')
    await user.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(onClose).toHaveBeenCalled()
  })

  it('shows the error the server returned', () => {
    render(
      <NewClientDialog
        open
        error="El DNI ya está en uso"
        onSubmit={vi.fn()}
        onClose={vi.fn()}
      />,
    )

    expect(screen.getByRole('alert')).toHaveTextContent('El DNI ya está en uso')
  })

  it('blocks the form while it is saving', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    render(<NewClientDialog open isSubmitting onSubmit={onSubmit} onClose={vi.fn()} />)

    expect(screen.getByLabelText('Nombre')).toBeDisabled()
    const submit = screen.getByRole('button', { name: 'Guardando…' })
    expect(submit).toBeDisabled()

    await user.click(submit)
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('replaces the validation message once the fields are complete', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    render(<NewClientDialog open onSubmit={onSubmit} onClose={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: 'Guardar cliente' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Completá nombre, apellido y DNI.')

    await user.type(screen.getByLabelText('Nombre'), 'Luis')
    await user.type(screen.getByLabelText('Apellido'), 'Gómez')
    await user.type(screen.getByLabelText('DNI'), '28999111')
    await user.click(screen.getByRole('button', { name: 'Guardar cliente' }))

    await waitFor(() => {
      expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    })
    expect(onSubmit).toHaveBeenCalled()
  })
})
