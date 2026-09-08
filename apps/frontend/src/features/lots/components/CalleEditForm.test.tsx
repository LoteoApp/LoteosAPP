import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import CalleEditForm from './CalleEditForm'
import type { LoteoCalle } from '../types'

function calle(overrides: Partial<LoteoCalle> = {}): LoteoCalle {
  return {
    id: 'ca-1',
    nombre: 'Los Álamos',
    tipo: 'asfalto',
    poligono: [],
    ...overrides,
  }
}

describe('CalleEditForm', () => {
  it('loads and saves the calle values', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn().mockResolvedValue(true)

    render(
      <CalleEditForm
        calle={calle()}
        updateState={{ status: 'idle' }}
        onSave={onSave}
      />,
    )

    expect(screen.getByLabelText('Nombre')).toHaveValue('Los Álamos')
    expect(screen.getByRole('button', { name: 'Asfalto' })).toHaveAttribute('aria-pressed', 'true')

    await user.clear(screen.getByLabelText('Nombre'))
    await user.type(screen.getByLabelText('Nombre'), 'San Martín')
    await user.click(screen.getByRole('button', { name: 'Tierra' }))
    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(onSave).toHaveBeenCalledWith({ nombre: 'San Martín', tipo: 'tierra' })
    expect(await screen.findByRole('alert')).toHaveTextContent('Calle guardada')
  })

  it('rejects an empty nombre without calling onSave', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn()

    render(
      <CalleEditForm
        calle={calle({ nombre: '' })}
        updateState={{ status: 'idle' }}
        onSave={onSave}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Guardar' }))
    expect(onSave).not.toHaveBeenCalled()
    expect(screen.getByText(/nombre de la calle es obligatorio/i)).toBeInTheDocument()
  })

  it('clears the save notice when the server rejects the update', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn().mockResolvedValue(false)

    render(
      <CalleEditForm calle={calle()} updateState={{ status: 'idle' }} onSave={onSave} />,
    )

    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(onSave).toHaveBeenCalledOnce()
    expect(screen.queryByText('Calle guardada')).not.toBeInTheDocument()
  })

  it('shows server errors on the matching field and in a banner otherwise', () => {
    const { rerender } = render(
      <CalleEditForm
        calle={calle()}
        updateState={{ status: 'error', message: 'Nombre inválido', field: 'nombre' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByText('Nombre inválido')).toBeInTheDocument()
    expect(screen.getByLabelText('Nombre')).toHaveAttribute('aria-invalid', 'true')

    rerender(
      <CalleEditForm
        calle={calle()}
        updateState={{ status: 'error', message: 'No se pudo guardar' }}
        onSave={vi.fn()}
      />,
    )
    expect(screen.getByRole('alert')).toHaveTextContent('No se pudo guardar')
  })

  it('reloads values when a different calle is selected', async () => {
    const { rerender } = render(
      <CalleEditForm
        calle={calle()}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByLabelText('Nombre')).toHaveValue('Los Álamos')

    rerender(
      <CalleEditForm
        calle={calle({ id: 'ca-2', nombre: 'Belgrano', tipo: 'tierra' })}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByLabelText('Nombre')).toHaveValue('Belgrano')
    expect(screen.getByRole('button', { name: 'Tierra' })).toHaveAttribute('aria-pressed', 'true')
  })
})
