import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import LoteoPlanCard from './LoteoPlanCard'
import { DXF_LAYERS, type DxfLayer, type DxfParseResult, type DxfPolygon } from '../types'

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

const polygon: DxfPolygon = {
  id: 'p1',
  layer: 'LOTES',
  handle: '1A',
  vertices: [
    { x: 0, y: 0 },
    { x: 10, y: 0 },
    { x: 10, y: 10 },
  ],
}

describe('LoteoPlanCard', () => {
  it('shows the empty state when there is no plan', () => {
    render(
      <LoteoPlanCard
        hasPlan={false}
        fileName={null}
        error={null}
        issues={[]}
        polygons={[]}
        visibleLayers={new Set(DXF_LAYERS)}
        onVisibleLayersChange={vi.fn()}
        onParsed={vi.fn()}
        onError={vi.fn()}
        onCleared={vi.fn()}
      />,
    )

    expect(screen.getByText('Todavía no hay plano')).toBeInTheDocument()
    expect(screen.queryByRole('img', { name: 'Plano del loteo' })).not.toBeInTheDocument()
  })

  it('shows the viewer and the layer toggles when there is a plan', () => {
    render(
      <LoteoPlanCard
        hasPlan
        fileName="plano.dxf"
        error={null}
        issues={[]}
        polygons={[polygon]}
        visibleLayers={new Set(DXF_LAYERS)}
        onVisibleLayersChange={vi.fn()}
        onParsed={vi.fn()}
        onError={vi.fn()}
        onCleared={vi.fn()}
      />,
    )

    expect(screen.getByText('Cargado desde plano.dxf.')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Plano del loteo' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lotes' })).toBeInTheDocument()
    expect(screen.queryByText('Todavía no hay plano')).not.toBeInTheDocument()
  })

  it('shows the parse error', () => {
    render(
      <LoteoPlanCard
        hasPlan={false}
        fileName={null}
        error="No se pudo interpretar el archivo DXF."
        issues={[]}
        polygons={[]}
        visibleLayers={new Set(DXF_LAYERS)}
        onVisibleLayersChange={vi.fn()}
        onParsed={vi.fn()}
        onError={vi.fn()}
        onCleared={vi.fn()}
      />,
    )

    expect(screen.getByRole('alert')).toHaveTextContent('No se pudo leer el DXF')
  })

  function Harness() {
    const [hasPlan, setHasPlan] = useState(false)
    const [fileName, setFileName] = useState<string | null>(null)
    const [polygons, setPolygons] = useState<DxfPolygon[]>([])
    const [visibleLayers, setVisibleLayers] = useState<ReadonlySet<DxfLayer>>(
      () => new Set(DXF_LAYERS),
    )

    function handleParsed(result: DxfParseResult, file: File) {
      setHasPlan(result.polygons.length > 0)
      setFileName(file.name)
      setPolygons(result.polygons)
    }

    return (
      <LoteoPlanCard
        hasPlan={hasPlan}
        fileName={fileName}
        error={null}
        issues={[]}
        polygons={polygons}
        visibleLayers={visibleLayers}
        onVisibleLayersChange={setVisibleLayers}
        onParsed={handleParsed}
        onError={vi.fn()}
        onCleared={vi.fn()}
      />
    )
  }

  it('switches to the loaded plan after uploading a DXF', async () => {
    const user = userEvent.setup()
    render(<Harness />)

    const file = new File(
      [dxfDocument(lwpolyline('LOTES', square))],
      'las-acacias.dxf',
      { type: 'application/dxf' },
    )
    await user.upload(screen.getByLabelText('Archivo DXF'), file)

    expect(await screen.findByRole('img', { name: 'Plano del loteo' })).toBeInTheDocument()
    expect(screen.getByText('Cargado desde las-acacias.dxf.')).toBeInTheDocument()
  })
})
