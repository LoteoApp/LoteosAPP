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

    const { polygons, issues } = parseDxf(dxf)

    expect(polygons).toHaveLength(4)
    expect(polygons.map((polygon) => polygon.layer).sort()).toEqual([
      'CALLE',
      'LOTEO',
      'LOTES',
      'MANZANA',
    ])
    expect(issues).toEqual([])
  })

  it('keeps the DXF handle when present and returns null otherwise', () => {
    const dxf = dxfDocument(
      lwpolyline({ layer: 'LOTEO', closed: true, points: square, handle: '2A' }),
      lwpolyline({ layer: 'MANZANA', closed: true, points: square }),
    )

    const { polygons } = parseDxf(dxf)

    expect(polygons.find((polygon) => polygon.layer === 'LOTEO')?.handle).toBe('2A')
    expect(polygons.find((polygon) => polygon.layer === 'MANZANA')?.handle).toBeNull()
  })

  it('returns the polygon vertices in order', () => {
    const dxf = dxfDocument(lwpolyline({ layer: 'LOTES', closed: true, points: square }))

    const {
      polygons: [polygon],
    } = parseDxf(dxf)

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

    const { polygons } = parseDxf(dxf)

    expect(polygons.map((polygon) => polygon.layer).sort()).toEqual(['CALLE', 'MANZANA'])
  })

  it('ignores layers outside LOTEO, MANZANA, LOTES and CALLE', () => {
    const dxf = dxfDocument(
      lwpolyline({ layer: 'MEJORA', closed: true, points: square }),
      lwpolyline({ layer: 'LOTES', closed: true, points: square }),
    )

    const { polygons, issues } = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
    expect(polygons[0].layer).toBe('LOTES')
    expect(issues).toEqual([])
  })

  it('reports open polylines on relevant layers as an issue instead of the polygon', () => {
    const dxf = dxfDocument(
      lwpolyline({ layer: 'LOTES', closed: false, points: square }),
      lwpolyline({ layer: 'CALLE', closed: true, points: square }),
    )

    const { polygons, issues } = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
    expect(polygons[0].layer).toBe('CALLE')
    expect(issues).toEqual([
      expect.objectContaining({ code: 'OPEN_GEOMETRY', layer: 'LOTES', polygonId: null }),
    ])
  })

  it('treats a ring redrawn back to its start point (no closed flag) as closed within survey tolerance', () => {
    // Real DXF files from surveyors close a ring by tracing back to the
    // start point instead of ticking the CAD tool's "closed" flag; the
    // gap between first and last vertex is float noise, not a real break.
    const dxf = dxfDocument(
      lwpolyline({
        layer: 'LOTES',
        closed: false,
        points: [
          [5454366.141711058, 6486276.574564104],
          [5454374.856759446, 6486325.809185373],
          [5454355.16291094, 6486329.295204728],
          [5454346.447862548, 6486280.060583459],
          [5454366.141711059, 6486276.574564104],
        ],
      }),
    )

    const { polygons, issues } = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
    expect(polygons[0].vertices).toHaveLength(4)
    expect(issues).toEqual([])
  })

  it('keeps flagging a gap larger than the closing tolerance as open', () => {
    const dxf = dxfDocument(
      lwpolyline({
        layer: 'LOTES',
        closed: false,
        points: [
          [0, 0],
          [10, 0],
          [10, 10],
          [0, 0.05],
        ],
      }),
      lwpolyline({ layer: 'CALLE', closed: true, points: square }),
    )

    const { polygons, issues } = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
    expect(issues).toEqual([
      expect.objectContaining({ code: 'OPEN_GEOMETRY', layer: 'LOTES', polygonId: null }),
    ])
  })

  it('does not report open geometry on layers outside the domain', () => {
    const dxf = dxfDocument(
      lwpolyline({ layer: 'MEJORA', closed: false, points: square }),
      lwpolyline({ layer: 'CALLE', closed: true, points: square }),
    )

    const { polygons, issues } = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
    expect(issues).toEqual([])
  })

  it('reports degenerate polygons with fewer than 3 vertices as an issue', () => {
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

    const { polygons, issues } = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
    expect(issues).toEqual([
      expect.objectContaining({ code: 'DEGENERATE_POLYGON', layer: 'LOTES', polygonId: null }),
    ])
  })

  it('reports self-intersecting polygons as an issue instead of the polygon', () => {
    const bowtie: Point[] = [
      [0, 0],
      [10, 10],
      [10, 0],
      [0, 10],
    ]
    const dxf = dxfDocument(
      lwpolyline({ layer: 'LOTES', closed: true, points: bowtie }),
      lwpolyline({ layer: 'CALLE', closed: true, points: square }),
    )

    const { polygons, issues } = parseDxf(dxf)

    expect(polygons).toHaveLength(1)
    expect(polygons[0].layer).toBe('CALLE')
    expect(issues).toEqual([
      expect.objectContaining({ code: 'SELF_INTERSECTING', layer: 'LOTES', polygonId: null }),
    ])
  })

  it('reports overlapping polygons within the same layer', () => {
    const overlapping: Point[] = [
      [5, 5],
      [15, 5],
      [15, 15],
      [5, 15],
    ]
    const dxf = dxfDocument(
      lwpolyline({ layer: 'LOTES', closed: true, points: square }),
      lwpolyline({ layer: 'LOTES', closed: true, points: overlapping }),
    )

    const { polygons, issues } = parseDxf(dxf)

    expect(polygons).toHaveLength(2)
    expect(issues).toEqual([expect.objectContaining({ code: 'OVERLAPPING', layer: 'LOTES' })])
  })

  it('caps how many overlap warnings it reports', () => {
    const entities = Array.from({ length: 25 }, (_, index) =>
      lwpolyline({
        layer: 'LOTES',
        closed: true,
        points: [
          [index * 0.1, 0],
          [index * 0.1 + 10, 0],
          [index * 0.1 + 10, 10],
          [index * 0.1, 10],
        ],
      }),
    )

    const { polygons, issues } = parseDxf(dxfDocument(...entities))

    expect(polygons).toHaveLength(25)
    expect(issues).toHaveLength(200)
  })

  it('does not report a lot nested inside a block as overlapping (different layers)', () => {
    const dxf = dxfDocument(
      lwpolyline({ layer: 'MANZANA', closed: true, points: square }),
      lwpolyline({
        layer: 'LOTES',
        closed: true,
        points: [
          [2, 2],
          [8, 2],
          [8, 8],
          [2, 8],
        ],
      }),
    )

    const { issues } = parseDxf(dxf)

    expect(issues).toEqual([])
  })

  it('does not report adjacent polygons that only share an edge as overlapping', () => {
    const neighbor: Point[] = [
      [10, 0],
      [20, 0],
      [20, 10],
      [10, 10],
    ]
    const dxf = dxfDocument(
      lwpolyline({ layer: 'LOTES', closed: true, points: square }),
      lwpolyline({ layer: 'LOTES', closed: true, points: neighbor }),
    )

    const { issues } = parseDxf(dxf)

    expect(issues).toEqual([])
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

    const {
      polygons: [polygon],
    } = parseDxf(dxf)

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

    const { polygons } = parseDxf(dxf)

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

    const {
      polygons: [polygon],
    } = parseDxf(dxf)

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

    const { polygons } = parseDxf(dxf)

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
