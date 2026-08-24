import { describe, expect, it } from 'vitest'
import { DxfParseError } from './parseDxf'
import { readDxfFile } from './readDxfFile'

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

describe('readDxfFile', () => {
  it('parses a .dxf file into polygons', async () => {
    const content = dxfDocument(lwpolyline('LOTES', square))
    const file = new File([content], 'plano.dxf', { type: 'application/dxf' })

    const result = await readDxfFile(file)

    expect(result.polygons).toHaveLength(1)
    expect(result.polygons[0].layer).toBe('LOTES')
    expect(result.issues).toEqual([])
  })

  it('rejects a file that is not a DXF', async () => {
    const file = new File(['not a dxf'], 'plano.txt', { type: 'text/plain' })

    await expect(readDxfFile(file)).rejects.toEqual(
      new DxfParseError('El archivo debe tener extensión .dxf.'),
    )
  })

  it('rejects a file that exceeds the size limit', async () => {
    const file = new File(['x'], 'huge.dxf')
    Object.defineProperty(file, 'size', { value: 20_000_001 })

    await expect(readDxfFile(file)).rejects.toEqual(
      new DxfParseError('El archivo DXF supera el tamaño máximo permitido.'),
    )
  })
})
