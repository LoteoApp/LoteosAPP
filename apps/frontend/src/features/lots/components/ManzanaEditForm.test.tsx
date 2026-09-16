import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import ManzanaEditForm from './ManzanaEditForm'
import type { LoteoCalle, LoteoManzana } from '../types'

function manzana(overrides: Partial<LoteoManzana> = {}): LoteoManzana {
  return {
    id: 'mz-1',
    numero: '1',
    tieneAgua: false,
    tieneCloaca: false,
    tieneLuz: false,
    tieneGas: false,
    calleIds: [],
    poligono: [],
    ...overrides,
  }
}

const calles: LoteoCalle[] = [
  { id: 'ca-1', nombre: 'Los Álamos', tipo: 'asfalto', poligono: [] },
  { id: 'ca-2', nombre: 'San Martín', tipo: 'tierra', poligono: [] },
]

describe('ManzanaEditForm', () => {
  it('loads and saves the manzana values', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn().mockResolvedValue(true)

    render(
      <ManzanaEditForm
        manzana={manzana({ tieneAgua: true, calleIds: ['ca-1'] })}
        calles={calles}
        loteCount={2}
        updateState={{ status: 'idle' }}
        onSave={onSave}
      />,
    )

    expect(screen.getByLabelText('Número')).toHaveValue('1')
    expect(screen.getByText('2 lotes en esta manzana.')).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'Los Álamos' })).toBeChecked()

    await user.clear(screen.getByLabelText('Número'))
    await user.type(screen.getByLabelText('Número'), 'B')
    await user.click(screen.getByRole('button', { name: 'Luz' }))
    await user.click(screen.getByRole('checkbox', { name: 'San Martín' }))
    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(onSave).toHaveBeenCalledWith({
      numero: 'B',
      tieneAgua: true,
      tieneCloaca: false,
      tieneLuz: true,
      tieneGas: false,
      calleIds: ['ca-1', 'ca-2'],
    })
    expect(await screen.findByRole('alert')).toHaveTextContent('Manzana guardada')
  })

  it('rejects an empty numero without calling onSave', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn()

    render(
      <ManzanaEditForm
        manzana={manzana({ numero: '' })}
        calles={calles}
        loteCount={0}
        updateState={{ status: 'idle' }}
        onSave={onSave}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Guardar' }))
    expect(onSave).not.toHaveBeenCalled()
    expect(screen.getByText(/número de manzana es obligatorio/i)).toBeInTheDocument()
  })

  it('clears the save notice when the server rejects the update', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn().mockResolvedValue(false)

    render(
      <ManzanaEditForm
        manzana={manzana()}
        calles={calles}
        loteCount={0}
        updateState={{ status: 'idle' }}
        onSave={onSave}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Guardar' }))

    expect(onSave).toHaveBeenCalledOnce()
    expect(screen.queryByText('Manzana guardada')).not.toBeInTheDocument()
  })

  it('disables a fifth street when four are already selected', () => {
    const manyCalles = [
      ...calles,
      { id: 'ca-3', nombre: 'Belgrano', tipo: 'tierra', poligono: [] },
      { id: 'ca-4', nombre: 'Sarmiento', tipo: 'tierra', poligono: [] },
      { id: 'ca-5', nombre: 'Mitre', tipo: 'tierra', poligono: [] },
    ]

    render(
      <ManzanaEditForm
        manzana={manzana({ calleIds: ['ca-1', 'ca-2', 'ca-3', 'ca-4'] })}
        calles={manyCalles}
        loteCount={0}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByRole('checkbox', { name: 'Mitre' })).toBeDisabled()
  })

  it('removes a selected street when its checkbox is cleared', async () => {
    const user = userEvent.setup()

    render(
      <ManzanaEditForm
        manzana={manzana({ calleIds: ['ca-1'] })}
        calles={calles}
        loteCount={0}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
      />,
    )

    const selected = screen.getByRole('checkbox', { name: /Los .*lamos/i })
    expect(selected).toBeChecked()
    await user.click(selected)
    expect(selected).not.toBeChecked()
  })

  it('shows server field errors, saving state and an empty street list', () => {
    const { rerender } = render(
      <ManzanaEditForm
        manzana={manzana()}
        calles={[]}
        loteCount={0}
        updateState={{ status: 'error', message: 'Calle inválida', field: 'calleIds' }}
        onSave={vi.fn()}
      />,
    )

    expect(screen.getByText(/todavía no tiene calles/i)).toBeInTheDocument()
    expect(screen.getByText('Calle inválida')).toBeInTheDocument()

    rerender(
      <ManzanaEditForm
        manzana={manzana({ id: 'mz-2', numero: '2' })}
        calles={[]}
        loteCount={0}
        updateState={{ status: 'error', message: 'Número inválido', field: 'numero' }}
        onSave={vi.fn()}
      />,
    )
    expect(screen.getByLabelText('Número')).toHaveAttribute('aria-invalid', 'true')

    rerender(
      <ManzanaEditForm
        manzana={manzana({ id: 'mz-2', numero: '2' })}
        calles={[]}
        loteCount={0}
        updateState={{ status: 'saving' }}
        onSave={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: /Guardando/ })).toBeDisabled()
  })
})
