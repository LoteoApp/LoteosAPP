import { describe, expect, it } from 'vitest'
import { planFromLoteoDetail } from './planFromLoteoDetail'
import type { LoteoDetail } from '../types'

const triangle = [
  { x: 0, y: 0 },
  { x: 1, y: 0 },
  { x: 1, y: 1 },
]

function detail(overrides: Partial<LoteoDetail> = {}): LoteoDetail {
  return {
    id: 'loteo-1',
    nombre: 'Las Acacias',
    ubicacion: '',
    descripcion: '',
    contorno: triangle,
    manzanas: [{ id: 'shared', numero: '1', poligono: triangle }],
    lotes: [{
      id: 'shared',
      manzanaId: 'shared',
      numero: '1',
      precio: null,
      moneda: '',
      superficie: null,
      caracteristicas: '',
      poligono: triangle,
    }],
    calles: [{ id: 'ca-1', nombre: 'Los Álamos', tipo: 'asfalto', poligono: triangle }],
    fechaCreacion: '2026-08-20T12:00:00Z',
    ...overrides,
  }
}

describe('planFromLoteoDetail', () => {
  it('maps the four layers into DxfPolygons', () => {
    const polygons = planFromLoteoDetail(detail())

    expect(polygons.map((polygon) => polygon.layer)).toEqual([
      'LOTEO',
      'MANZANA',
      'LOTES',
      'CALLE',
    ])
    expect(polygons.every((polygon) => polygon.vertices.length === 3)).toBe(true)
  })

  it('keeps ids unique even when a lote and a manzana share a database id', () => {
    const polygons = planFromLoteoDetail(detail())
    const ids = polygons.map((polygon) => polygon.id)

    expect(new Set(ids).size).toBe(ids.length)
    expect(ids).toContain('manzana-shared')
    expect(ids).toContain('lote-shared')
  })

  it('drops entities that have no ring yet, layer by layer', () => {
    const polygons = planFromLoteoDetail(
      detail({
        contorno: [],
        manzanas: [{ id: 'mz-1', numero: '1', poligono: [] }],
      }),
    )

    expect(polygons.map((polygon) => polygon.layer)).toEqual(['LOTES', 'CALLE'])

    const onlyContorno = planFromLoteoDetail(
      detail({
        lotes: [{
          id: 'lt-1',
          manzanaId: 'mz-1',
          numero: '1',
          precio: null,
          moneda: '',
          superficie: null,
          caracteristicas: '',
          poligono: [],
        }],
        calles: [{ id: 'ca-1', nombre: 'x', tipo: 'asfalto', poligono: [] }],
      }),
    )

    expect(onlyContorno.map((polygon) => polygon.layer)).toEqual(['LOTEO', 'MANZANA'])
  })

  it('returns an empty plan for a loteo without geometry', () => {
    const polygons = planFromLoteoDetail(
      detail({ contorno: [], manzanas: [], lotes: [], calles: [] }),
    )

    expect(polygons).toEqual([])
  })
})
