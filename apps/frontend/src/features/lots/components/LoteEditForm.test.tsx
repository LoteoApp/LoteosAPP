import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import LoteEditForm from './LoteEditForm'
import type { LoteoLote } from '../types'

function lote(overrides: Partial<LoteoLote> = {}): LoteoLote {
  return {
    id: 'lt-1',
    manzanaId: 'mz-1',
    numero: '7',
    precio: 150000,
    moneda: 'USD',
    superficie: 300,
    caracteristicas: 'Esquina',
    poligono: [],
    ...overrides,
  }
}

describe('LoteEditForm', () => {
  it('loads the lote and saves the typed values', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn().mockResolvedValue(true)

    render(
      <LoteEditForm lote={lote()} updateState={{ status: 'idle' }} onSave={onSave} />,
    )

    expect(screen.getByLabelText('Número')).toHaveValue('7')
    expect(screen.getByLabelText('Precio')).toHaveValue('150.000')
    expect(screen.getByLabelText('Superficie (m²)')).toHaveValue('300')
    expect(screen.getByLabelText('Características')).toHaveValue('Esquina')

    await user.clear(screen.getByLabelText('Número'))
    await user.type(screen.getByLabelText('Número'), '12')
    await user.clear(screen.getByLabelText('Precio'))
    await user.type(screen.getByLabelText('Precio'), '200000')
    expect(screen.getByLabelText('Precio')).toHaveValue('200.000')
    await user.click(screen.getByRole('button', { name: 'ARS' }))
    await user.clear(screen.getByLabelText('Superficie (m²)'))
    await user.type(screen.getByLabelText('Superficie (m²)'), '310.5')
    await user.clear(screen.getByLabelText('Características'))
    await user.type(screen.getByLabelText('Características'), 'Frente norte')
    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(onSave).toHaveBeenCalledWith({
      numero: '12',
      precio: 200000,
      moneda: 'ARS',
      superficie: 310.5,
      caracteristicas: 'Frente norte',
    })
    expect(await screen.findByRole('alert')).toHaveTextContent('Lote guardado')
  })

  it('shows a 409 on the numero field and keeps the typed value', async () => {
    render(
      <LoteEditForm
        lote={lote({ numero: '12' })}
        updateState={{
          status: 'error',
          message: 'Ya existe un lote con ese número en este loteo',
          field: 'numero',
        }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByLabelText('Número')).toHaveValue('12')
    expect(
      screen.getByText('Ya existe un lote con ese número en este loteo'),
    ).toBeInTheDocument()
    expect(screen.queryByRole('alert', { name: /ya existe/i })).not.toBeInTheDocument()
  })

  it('disables the submit button while saving', () => {
    render(
      <LoteEditForm
        lote={lote()}
        updateState={{ status: 'saving' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByRole('button', { name: 'Guardando…' })).toBeDisabled()
  })

  it('resets the draft when the lote changes', async () => {
    const user = userEvent.setup()
    const { rerender } = render(
      <LoteEditForm lote={lote()} updateState={{ status: 'idle' }} onSave={vi.fn()} />,
    )

    await user.clear(screen.getByLabelText('Número'))
    await user.type(screen.getByLabelText('Número'), '99')

    rerender(
      <LoteEditForm
        lote={lote({ id: 'lt-2', numero: '8' })}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByLabelText('Número')).toHaveValue('8')
  })

  it('shows client-side validation on submit without calling onSave', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn()

    render(
      <LoteEditForm lote={lote({ numero: '' })} updateState={{ status: 'idle' }} onSave={onSave} />,
    )

    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(onSave).not.toHaveBeenCalled()
    expect(screen.getByText(/número de lote es obligatorio/i)).toBeInTheDocument()
  })

  it('shows a banner when the error has no field', () => {
    render(
      <LoteEditForm
        lote={lote()}
        updateState={{ status: 'error', message: 'No tenés permiso para editar este loteo.' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByText('No tenés permiso para editar este loteo.')).toBeInTheDocument()
  })

  it('adds a saved currency that is not ARS or USD', () => {
    render(
      <LoteEditForm
        lote={lote({ moneda: 'EUR' })}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByRole('button', { name: 'EUR' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'ARS' })).toBeInTheDocument()
  })

  it('groups the precio with thousand dots as the amount is typed', async () => {
    const user = userEvent.setup()

    render(
      <LoteEditForm lote={lote({ precio: null })} updateState={{ status: 'idle' }} onSave={vi.fn()} />,
    )

    const precio = screen.getByLabelText('Precio')
    await user.type(precio, '1500')
    expect(precio).toHaveValue('1.500')
    await user.type(precio, '000,5')
    expect(precio).toHaveValue('1.500.000,5')
  })

  it('clears the currency when an existing price is removed', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn().mockResolvedValue(true)

    render(<LoteEditForm lote={lote()} updateState={{ status: 'idle' }} onSave={onSave} />)

    await user.clear(screen.getByLabelText('Precio'))
    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(onSave).toHaveBeenCalledWith({
      numero: '7',
      precio: null,
      moneda: '',
      superficie: 300,
      caracteristicas: 'Esquina',
    })
  })

  it('preloads superficie from the closed polygon when the saved value is null', async () => {
    const user = userEvent.setup()
    const square = [
      { x: 0, y: 0 },
      { x: 10, y: 0 },
      { x: 10, y: 10 },
      { x: 0, y: 10 },
    ]

    render(
      <LoteEditForm
        lote={lote({ superficie: null, poligono: square })}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByLabelText('Superficie (m²)')).toHaveValue('100')
    expect(screen.getByText('Calculada del plano. Podés modificarla.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Calcular del plano' })).not.toBeInTheDocument()

    await user.clear(screen.getByLabelText('Superficie (m²)'))
    await user.type(screen.getByLabelText('Superficie (m²)'), '95.5')
    expect(screen.getByLabelText('Superficie (m²)')).toHaveValue('95.5')
  })

  it('keeps a saved superficie instead of replacing it with the polygon area', () => {
    render(
      <LoteEditForm
        lote={lote({
          superficie: 300,
          poligono: [
            { x: 0, y: 0 },
            { x: 10, y: 0 },
            { x: 10, y: 10 },
            { x: 0, y: 10 },
          ],
        })}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByLabelText('Superficie (m²)')).toHaveValue('300')
    expect(screen.queryByText('Calculada del plano. Podés modificarla.')).not.toBeInTheDocument()
  })
})
