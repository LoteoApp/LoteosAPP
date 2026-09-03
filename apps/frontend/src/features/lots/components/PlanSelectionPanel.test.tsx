import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import PlanSelectionPanel from './PlanSelectionPanel'
import type { LoteoDetail } from '../types'

const triangle = [
  { x: 0, y: 0 },
  { x: 1, y: 0 },
  { x: 1, y: 1 },
]

function loteo(overrides: Partial<LoteoDetail> = {}): LoteoDetail {
  return {
    id: 'loteo-1',
    nombre: 'Las Acacias',
    ubicacion: '',
    descripcion: '',
    contorno: triangle,
    manzanas: [{
      id: 'mz-1',
      numero: '1',
      tieneAgua: false,
      tieneCloaca: false,
      tieneLuz: false,
      tieneGas: false,
      calleIds: [],
      poligono: triangle,
    }],
    lotes: [
      {
        id: 'lt-1',
        manzanaId: 'mz-1',
        numero: '7',
        precio: null,
        moneda: '',
        superficie: null,
        caracteristicas: '',
        poligono: triangle,
      },
    ],
    calles: [{ id: 'ca-1', nombre: 'Los Álamos', tipo: 'asfalto', poligono: triangle }],
    fechaCreacion: '2026-08-20T12:00:00Z',
    ...overrides,
  }
}

const labels = new Map([
  ['lote-lt-1', 'Lote 7'],
  ['manzana-mz-1', 'Manzana 1'],
  ['calle-ca-1', 'Calle Los Álamos'],
])

describe('PlanSelectionPanel', () => {
  it('prompts the user when nothing is selected', () => {
    render(
      <PlanSelectionPanel
        selected={null}
        loteo={loteo()}
        polygonLabels={labels}
        selectedPolygonId={null}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Tocá una manzana, un lote o una calle.')).toBeInTheDocument()
  })

  it('shows the lote form for a selected lote', () => {
    render(
      <PlanSelectionPanel
        selected={{ kind: 'lote', id: 'lt-1' }}
        loteo={loteo()}
        polygonLabels={labels}
        selectedPolygonId="lote-lt-1"
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Lote 7')).toBeInTheDocument()
    expect(screen.getByLabelText('Número')).toHaveValue('7')
  })

  it('shows the manzana form for a selected manzana', () => {
    render(
      <PlanSelectionPanel
        selected={{ kind: 'manzana', id: 'mz-1' }}
        loteo={loteo()}
        polygonLabels={labels}
        selectedPolygonId="manzana-mz-1"
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Manzana 1')).toBeInTheDocument()
    expect(screen.getByLabelText('Número')).toHaveValue('1')
    expect(screen.getByText('1 lote en esta manzana.')).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'Los Álamos' })).toBeInTheDocument()
  })

  it('shows the calle form for a selected calle', () => {
    render(
      <PlanSelectionPanel
        selected={{ kind: 'calle', id: 'ca-1' }}
        loteo={loteo()}
        polygonLabels={labels}
        selectedPolygonId="calle-ca-1"
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Calle Los Álamos')).toBeInTheDocument()
    expect(screen.getByLabelText('Nombre')).toHaveValue('Los Álamos')
    expect(screen.getByRole('button', { name: 'Asfalto' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('shows selected data without edit controls when the user cannot edit', () => {
    render(
      <PlanSelectionPanel
        canEdit={false}
        selected={{ kind: 'lote', id: 'lt-1' }}
        loteo={loteo()}
        polygonLabels={labels}
        selectedPolygonId="lote-lt-1"
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Número')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Guardar' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Precio')).not.toBeInTheDocument()
  })

  it('renders read-only manzana and calle details for non-editors', () => {
    const { rerender } = render(
      <PlanSelectionPanel
        canEdit={false}
        selected={{ kind: 'manzana', id: 'mz-1' }}
        loteo={loteo({ manzanas: [{ ...loteo().manzanas[0], tieneAgua: true, calleIds: ['ca-1'] }] })}
        polygonLabels={labels}
        selectedPolygonId="manzana-mz-1"
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Agua')).toBeInTheDocument()
    expect(screen.getByText(/Los .*lamos/i)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Guardar' })).not.toBeInTheDocument()

    rerender(
      <PlanSelectionPanel
        canEdit={false}
        selected={{ kind: 'calle', id: 'ca-1' }}
        loteo={loteo()}
        polygonLabels={labels}
        selectedPolygonId="calle-ca-1"
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('asfalto')).toBeInTheDocument()
    expect(screen.queryByLabelText('Nombre')).not.toBeInTheDocument()
  })

  it('forwards a save from the lote form', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn().mockResolvedValue(true)

    render(
      <PlanSelectionPanel
        selected={{ kind: 'lote', id: 'lt-1' }}
        loteo={loteo()}
        polygonLabels={labels}
        selectedPolygonId="lote-lt-1"
        updateState={{ status: 'idle' }}
        onSave={onSave}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Guardar' }))
    expect(onSave).toHaveBeenCalledWith(
      'lt-1',
      expect.objectContaining({ numero: '7' }),
    )
  })

  it('forwards a save from the calle form', async () => {
    const user = userEvent.setup()
    const onSaveCalle = vi.fn().mockResolvedValue(true)

    render(
      <PlanSelectionPanel
        selected={{ kind: 'calle', id: 'ca-1' }}
        loteo={loteo()}
        polygonLabels={labels}
        selectedPolygonId="calle-ca-1"
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={onSaveCalle}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Guardar' }))
    expect(onSaveCalle).toHaveBeenCalledWith(
      'ca-1',
      expect.objectContaining({ nombre: 'Los Álamos', tipo: 'asfalto' }),
    )
  })

  it('treats the loteo contour as an empty selection', () => {
    render(
      <PlanSelectionPanel
        selected={{ kind: 'loteo' }}
        loteo={loteo()}
        polygonLabels={labels}
        selectedPolygonId="loteo"
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Tocá una manzana, un lote o una calle.')).toBeInTheDocument()
  })

  it('falls back when the selected entity is no longer in the loteo', () => {
    render(
      <PlanSelectionPanel
        selected={{ kind: 'lote', id: 'missing' }}
        loteo={loteo()}
        polygonLabels={labels}
        selectedPolygonId={null}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Tocá una manzana, un lote o una calle.')).toBeInTheDocument()
  })

  it('titles a manzana and a calle from the loteo when the plan has no label', () => {
    const unnamed = loteo({
      manzanas: [{
        id: 'mz-1',
        numero: '',
        tieneAgua: false,
        tieneCloaca: false,
        tieneLuz: false,
        tieneGas: false,
        calleIds: [],
        poligono: triangle,
      }],
      lotes: [
        { ...loteo().lotes[0], numero: '', manzanaId: 'mz-1' },
        { ...loteo().lotes[0], id: 'lt-2', numero: '8', manzanaId: 'mz-1' },
      ],
      calles: [{ id: 'ca-1', nombre: '', tipo: '', poligono: triangle }],
    })

    const { rerender } = render(
      <PlanSelectionPanel
        selected={{ kind: 'manzana', id: 'mz-1' }}
        loteo={unnamed}
        polygonLabels={new Map()}
        selectedPolygonId="manzana-mz-1"
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Manzana')).toBeInTheDocument()
    expect(screen.getByText('2 lotes en esta manzana.')).toBeInTheDocument()
    expect(screen.getByLabelText('Número')).toHaveValue('')

    rerender(
      <PlanSelectionPanel
        selected={{ kind: 'calle', id: 'ca-1' }}
        loteo={unnamed}
        polygonLabels={new Map()}
        selectedPolygonId="calle-ca-1"
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Calle')).toBeInTheDocument()
    expect(screen.getByLabelText('Nombre')).toHaveValue('')
  })

  it('titles a lote without a number when the plan has no label', () => {
    render(
      <PlanSelectionPanel
        selected={{ kind: 'lote', id: 'lt-1' }}
        loteo={loteo({ lotes: [{ ...loteo().lotes[0], numero: '' }] })}
        polygonLabels={new Map()}
        selectedPolygonId={null}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Lote')).toBeInTheDocument()
  })

  it('still identifies a missing manzana or calle without crashing', () => {
    const { rerender } = render(
      <PlanSelectionPanel
        selected={{ kind: 'manzana', id: 'ghost' }}
        loteo={loteo()}
        polygonLabels={new Map()}
        selectedPolygonId={null}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Tocá una manzana, un lote o una calle.')).toBeInTheDocument()

    rerender(
      <PlanSelectionPanel
        selected={{ kind: 'calle', id: 'ghost' }}
        loteo={loteo()}
        polygonLabels={new Map()}
        selectedPolygonId={null}
        updateState={{ status: 'idle' }}
        onSave={vi.fn()}
        manzanaUpdateState={{ status: 'idle' }}
        onSaveManzana={vi.fn()}
        calleUpdateState={{ status: 'idle' }}
        onSaveCalle={vi.fn()}
      />,
    )

    expect(screen.getByText('Tocá una manzana, un lote o una calle.')).toBeInTheDocument()
  })
})
