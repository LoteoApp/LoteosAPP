import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import LotsPage from './LotsPage'

function lwpolyline(layer: string, points: Array<[number, number]>): string {
  const lines = ['0', 'LWPOLYLINE', '8', layer, '90', String(points.length), '70', '1']
  for (const [x, y] of points) {
    lines.push('10', String(x), '20', String(y))
  }
  return lines.join('\n')
}

function dxfDocument(...entities: string[]): string {
  return ['0', 'SECTION', '2', 'ENTITIES', ...entities, '0', 'ENDSEC', '0', 'EOF', ''].join('\n')
}

const square: Array<[number, number]> = [
  [0, 0],
  [10, 0],
  [10, 10],
  [0, 10],
]

describe('LotsPage', () => {
  it('renders the loteo form with the plan pending, DXF optional', () => {
    render(<LotsPage />)

    expect(screen.getByRole('heading', { name: 'Nuevo loteo' })).toBeInTheDocument()
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
    expect(screen.getByLabelText('Nombre')).toBeInTheDocument()
    expect(screen.getByLabelText('Ubicación/Ciudad')).toBeInTheDocument()
    expect(screen.getByLabelText('Inmobiliarias')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Seleccionar todas' })).toBeInTheDocument()
    expect(screen.getByLabelText('Descripción')).toBeInTheDocument()
    expect(screen.getByLabelText('Archivo DXF')).toBeInTheDocument()
    expect(screen.getByText('Todavía no hay plano')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Guardar loteo' })).toBeDisabled()

    const nameInput = screen.getByLabelText('Nombre')
    const fileInput = screen.getByLabelText('Archivo DXF')
    expect(
      nameInput.compareDocumentPosition(fileInput) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })

  it('draws a DXF, marks the plan as loaded and hides a layer when its toggle is turned off', async () => {
    const user = userEvent.setup()
    render(<LotsPage />)

    const file = new File(
      [
        dxfDocument(
          lwpolyline('LOTEO', square),
          lwpolyline('MANZANA', square),
          lwpolyline('LOTES', square),
        ),
      ],
      'plano.dxf',
      { type: 'application/dxf' },
    )
    await user.upload(screen.getByLabelText('Archivo DXF'), file)

    expect(await screen.findByText('Plano cargado')).toBeInTheDocument()
    const svg = screen.getByRole('img', { name: 'Plano del loteo' })
    expect(svg.querySelector('[aria-label="Manzana"]')).not.toBeNull()
    expect(screen.getByText('Cargado desde plano.dxf.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Manzana' }))
    expect(svg.querySelector('[aria-label="Manzana"]')).toBeNull()
    expect(svg.querySelector('[aria-label="Loteo"]')).not.toBeNull()
  })

  it('lets the loteo be saved with data only, without a DXF', () => {
    render(<LotsPage />)

    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
    expect(screen.getByText('Opcional. Lo carga quien tenga el DXF.')).toBeInTheDocument()
    expect(screen.queryByRole('img', { name: 'Plano del loteo' })).not.toBeInTheDocument()
  })

  it('shows an error when the DXF has no usable polygons', async () => {
    const user = userEvent.setup()
    render(<LotsPage />)

    const file = new File(['0\nSECTION\n2\nENTITIES\n0\nENDSEC\n0\nEOF\n'], 'vacio.dxf', {
      type: 'application/dxf',
    })
    await user.upload(screen.getByLabelText('Archivo DXF'), file)

    expect(await screen.findByRole('alert')).toHaveTextContent('No se pudo leer el DXF')
    expect(screen.getByText('Todavía no hay plano')).toBeInTheDocument()
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
  })

  it('shows a warning when the DXF has overlapping lots', async () => {
    const user = userEvent.setup()
    render(<LotsPage />)

    const overlapping = dxfDocument(
      lwpolyline('LOTES', square),
      lwpolyline('LOTES', [
        [5, 5],
        [15, 5],
        [15, 15],
        [5, 15],
      ]),
    )
    const file = new File([overlapping], 'solape.dxf', { type: 'application/dxf' })
    await user.upload(screen.getByLabelText('Archivo DXF'), file)

    expect(await screen.findByRole('alert')).toHaveTextContent('Avisos de geometría')
    expect(screen.getByRole('img', { name: 'Plano del loteo' })).toBeInTheDocument()
  })

  it('discards the entered data and the loaded plan', async () => {
    const user = userEvent.setup()
    render(<LotsPage />)

    await user.type(screen.getByLabelText('Nombre'), 'Las Acacias')
    await user.click(screen.getByRole('button', { name: 'Seleccionar todas' }))
    const file = new File(
      [dxfDocument(lwpolyline('LOTES', square))],
      'plano.dxf',
      { type: 'application/dxf' },
    )
    await user.upload(screen.getByLabelText('Archivo DXF'), file)
    expect(await screen.findByText('Plano cargado')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Descartar' }))

    expect(screen.getByLabelText('Nombre')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Seleccionar todas' })).toBeInTheDocument()
    expect(screen.getByText('Plano pendiente')).toBeInTheDocument()
    expect(screen.getByText('Todavía no hay plano')).toBeInTheDocument()
  })
})
