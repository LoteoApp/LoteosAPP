import { describe, expect, it } from 'vitest'
import { DxfParseError, parseDxf } from './parseDxf'

type Point = [number, number]

function lwpolyline(options: {
  layer: string
  closed: boolean
  points: Point[]
  handle?: string
}): string {
  const lines = ['0', 'LWPOLYLINE']
  if (options.handle) lines.push('5', options.handle)
  lines.push('8', options.layer, '90', String(options.points.length), '70', options.closed ? '1' : '0')
  for (const [x, y] of options.points) {
    lines.push('10', String(x), '20', String(y))
  }
  return lines.join('\n')
}

function polyline(options: { layer: string; closed: boolean; points: Point[] }): string {
  const lines = ['0', 'POLYLINE', '8', options.layer, '66', '1', '70', options.closed ? '1' : '0']
  for (const [x, y] of options.points) {
    lines.push('0', 'VERTEX', '8', options.layer, '10', String(x), '20', String(y))
  }
  lines.push('0', 'SEQEND')
  return lines.join('\n')
}

function dxfDocument(...entityBlocks: string[]): string {
  return ['0', 'SECTION', '2', 'ENTITIES', ...entityBlocks, '0', 'ENDSEC', '0', 'EOF', ''].join('\n')
}

function block(options: { name: string; entities: string[] }): string {
  return [
    '0',
    'BLOCK',
    '2',
    options.name,
    '8',
    '0',
    '10',
    '0',
    '20',
    '0',
    ...options.entities,
    '0',
    'ENDBLK',
  ].join('\n')
}

function insert(options: {
  block: string
  layer: string
  x: number
  y: number
  scaleX?: number
  scaleY?: number
  columnCount?: number
  rowCount?: number
}): string {
  const lines = [
    '0',
    'INSERT',
    '8',
    options.layer,
    '2',
    options.block,
    '10',
    String(options.x),
    '20',
    String(options.y),
  ]
  if (options.scaleX !== undefined) lines.push('41', String(options.scaleX))
  if (options.scaleY !== undefined) lines.push('42', String(options.scaleY))
  if (options.columnCount !== undefined) lines.push('70', String(options.columnCount))
  if (options.rowCount !== undefined) lines.push('71', String(options.rowCount))
  return lines.join('\n')
}

function dxfDocumentWithBlocks(blocks: string[], entityBlocks: string[]): string {
  return [
    '0',
    'SECTION',
    '2',
    'BLOCKS',
    ...blocks,
    '0',
    'ENDSEC',
    '0',
    'SECTION',
    '2',
    'ENTITIES',
    ...entityBlocks,
    '0',
    'ENDSEC',
    '0',
    'EOF',
    '',
  ].join('\n')
}

const square: Point[] = [
  [0, 0],
  [10, 0],
  [10, 10],
  [0, 10],
]

describe('parseDxf', () => {
  it('extracts closed polygons from all supported layers', () => {
    const dxf = dxfDocument(
      lwpolyline({ layer: 'LOTEO', closed: true, points: square, handle: '2A' }),
      lwpolyline({ layer: 'MANZANA', closed: true, points: square }),
      polyline({ layer: 'LOTES', closed: true, points: square }),
      lwpolyline({ layer: 'CALLE', closed: true, points: square }),
    )

    const polygons = parseDxf(dxf)

    expect(polygons).toHaveLength(4)
    expect(polygons.map((polygon) => polygon.layer).sort()).toEqual([
      'CALLE',
      'LOTEO',
      'LOTES',
      'MANZANA',
    ])
  })

  it('keeps the DXF handle when present and returns null otherwise', () => {
    const dxf = dxfDocument(
      lwpolyline({ layer: 'LOTEO', closed: true, points: square, handle: '2A' }),
      lwpolyline({ layer: 'MANZANA', closed: true, points: square }),
    )

    const polygons = parseDxf(dxf)

    expect(polygons.find((polygon) => polygon.layer === 'LOTEO')?.handle).toBe('2A')
    expect(polygons.find((polygon) => polygon.layer === 'MANZANA')?.handle).toBeNull()
  })

  it('returns the polygon vertices in order', () => {
    const dxf = dxfDocument(lwpolyline({ layer: 'LOTES', closed: true, points: square }))

    const [polygon] = parseDxf(dxf)

    expect(polygon.vertices).toEqual([
      { x: 0, y: 0 },
      { x: 10, y: 0 },
      { x: 10, y: 10 },
      { x: 0, y: 10 },
    ])
  })

  it('normalizes singular/plural layer name variants to the canonical layer', () => {
    const dxf = dxfDocument(
      lwpolyline({ layer: 'MANZANAS', closed: true, points: square }),
      lwpolyline({ layer: 'CALLES', closed: true, points: square }),
    )

    const polygons = parseDxf(dxf)

    expect(polygons.map((polygon) => polygon.layer).sort()).toEqual(['CALLE', 'MANZANA'])
  })

  it('ignores layers outside LOTEO, MANZANA, LOTES and CALLE', () => {
    const dxf = dxfDocument(
      lwpolyline({ layer: 'MEJORA', closed: true, points: square }),
      lwpolyline({ layer: 'LOTES', closed: true, points: square }),
    )

    const polygons = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
    expect(polygons[0].layer).toBe('LOTES')
  })

  it('ignores open polylines', () => {
    const dxf = dxfDocument(
      lwpolyline({ layer: 'LOTES', closed: false, points: square }),
      lwpolyline({ layer: 'CALLE', closed: true, points: square }),
    )

    const polygons = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
    expect(polygons[0].layer).toBe('CALLE')
  })

  it('ignores degenerate polygons with fewer than 3 vertices', () => {
    const dxf = dxfDocument(
      lwpolyline({
        layer: 'LOTES',
        closed: true,
        points: [
          [0, 0],
          [10, 0],
        ],
      }),
      lwpolyline({ layer: 'LOTES', closed: true, points: square }),
    )

    const polygons = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
  })

  it('throws DxfParseError when no valid polygons are found', () => {
    const dxf = dxfDocument(lwpolyline({ layer: 'MEJORA', closed: true, points: square }))

    expect(() => parseDxf(dxf)).toThrow(DxfParseError)
  })

  it('applies the block INSERT translation and scale to the polygon vertices', () => {
    const dxf = dxfDocumentWithBlocks(
      [block({ name: 'LOTE_BLOCK', entities: [lwpolyline({ layer: 'LOTES', closed: true, points: square })] })],
      [insert({ block: 'LOTE_BLOCK', layer: 'LOTES', x: 1000, y: 2000, scaleX: 2, scaleY: 2 })],
    )

    const [polygon] = parseDxf(dxf)

    expect(polygon.vertices).toEqual([
      { x: 1000, y: 2000 },
      { x: 1020, y: 2000 },
      { x: 1020, y: 2020 },
      { x: 1000, y: 2020 },
    ])
  })

  it('assigns a unique id to each polygon copied from the same block INSERT array', () => {
    const dxf = dxfDocumentWithBlocks(
      [block({ name: 'LOTE_BLOCK', entities: [lwpolyline({ layer: 'LOTES', closed: true, points: square, handle: 'AA' })] })],
      [insert({ block: 'LOTE_BLOCK', layer: 'LOTES', x: 0, y: 0, columnCount: 2, rowCount: 2 })],
    )

    const polygons = parseDxf(dxf)

    expect(polygons).toHaveLength(4)
    expect(polygons.every((polygon) => polygon.handle === 'AA')).toBe(true)
    expect(new Set(polygons.map((polygon) => polygon.id)).size).toBe(4)
  })

  it('rejects an INSERT array large enough to hang the parser', () => {
    const dxf = dxfDocumentWithBlocks(
      [block({ name: 'LOTE_BLOCK', entities: [lwpolyline({ layer: 'LOTES', closed: true, points: square })] })],
      [insert({ block: 'LOTE_BLOCK', layer: 'LOTES', x: 0, y: 0, columnCount: 100_000, rowCount: 100_000 })],
    )

    expect(() => parseDxf(dxf)).toThrow(DxfParseError)
  })

  it('tessellates a bulge into arc points instead of a straight edge', () => {
    const lines = [
      '0',
      'LWPOLYLINE',
      '8',
      'LOTES',
      '90',
      '3',
      '70',
      '1',
      '10',
      '0',
      '20',
      '0',
      '42',
      '1',
      '10',
      '10',
      '20',
      '0',
      '10',
      '10',
      '20',
      '10',
    ].join('\n')
    const dxf = dxfDocument(lines)

    const [polygon] = parseDxf(dxf)

    expect(polygon.vertices.length).toBeGreaterThan(3)
    expect(polygon.vertices.some((v) => v.y < -1)).toBe(true)
  })

  it('discards the whole polygon when it contains a non-finite vertex', () => {
    const lines = [
      '0',
      'LWPOLYLINE',
      '8',
      'LOTES',
      '90',
      '4',
      '70',
      '1',
      '10',
      '0',
      '20',
      '0',
      '10',
      'not-a-number',
      '20',
      '0',
      '10',
      '10',
      '20',
      '10',
      '10',
      '0',
      '20',
      '10',
    ].join('\n')
    const dxf = dxfDocument(lines, lwpolyline({ layer: 'LOTES', closed: true, points: square }))

    const polygons = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
  })

  it('rejects a file larger than the maximum allowed size', () => {
    const dxf = dxfDocument(lwpolyline({ layer: 'LOTES', closed: true, points: square }))
    const padded = dxf + '999'.repeat(7_000_000)

    expect(() => parseDxf(padded)).toThrow(DxfParseError)
  })

  it('wraps unexpected parser failures in a DxfParseError', () => {
    const dxf = dxfDocumentWithBlocks(
      [block({ name: 'SELF', entities: [insert({ block: 'SELF', layer: 'LOTES', x: 0, y: 0 })] })],
      [insert({ block: 'SELF', layer: 'LOTES', x: 0, y: 0 })],
    )

    expect(() => parseDxf(dxf)).toThrow(DxfParseError)
  })
})
