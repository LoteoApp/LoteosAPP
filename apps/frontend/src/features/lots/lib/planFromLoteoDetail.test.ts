import { describe, expect, it } from 'vitest'
import { planFromLoteoDetail, planLabelsFromLoteoDetail } from './planFromLoteoDetail'
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
    manzanas: [{ id: 'shared', numero: '1', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: triangle }],
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
    expect(polygons.find((polygon) => polygon.layer === 'LOTES')?.caption).toBe('1')
    expect(polygons.find((polygon) => polygon.layer === 'MANZANA')?.caption).toBe('1')
    expect(polygons.find((polygon) => polygon.layer === 'CALLE')?.caption).toBe('Los Álamos')
  })

  it('omits captions when the number or name has not been loaded yet', () => {
    const polygons = planFromLoteoDetail(
      detail({
        manzanas: [{ ...detail().manzanas[0], numero: '' }],
        lotes: [{ ...detail().lotes[0], numero: '' }],
        calles: [{ ...detail().calles[0], nombre: '' }],
      }),
    )

    expect(polygons.find((polygon) => polygon.layer === 'MANZANA')?.caption).toBeUndefined()
    expect(polygons.find((polygon) => polygon.layer === 'LOTES')?.caption).toBeUndefined()
    expect(polygons.find((polygon) => polygon.layer === 'CALLE')?.caption).toBeUndefined()
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
        manzanas: [{ id: 'mz-1', numero: '1', tieneAgua: false, tieneCloaca: false, tieneLuz: false, tieneGas: false, calleIds: [], poligono: [] }],
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

  it('attaches a typed entity ref per layer', () => {
    const polygons = planFromLoteoDetail(detail())

    expect(polygons.map((polygon) => polygon.entity)).toEqual([
      { kind: 'loteo' },
      { kind: 'manzana', id: 'shared' },
      { kind: 'lote', id: 'shared' },
      { kind: 'calle', id: 'ca-1' },
    ])
  })
})

describe('planLabelsFromLoteoDetail', () => {
  it('labels each polygon by its domain name', () => {
    const labels = planLabelsFromLoteoDetail(detail())

    expect(labels.get('loteo')).toBe('Contorno del loteo')
    expect(labels.get('manzana-shared')).toBe('Manzana 1')
    expect(labels.get('lote-shared')).toBe('Lote 1')
    expect(labels.get('calle-ca-1')).toBe('Calle Los Álamos')
  })

  it('falls back when a lote or calle has no number or name yet', () => {
    const labels = planLabelsFromLoteoDetail(
      detail({
        lotes: [{ ...detail().lotes[0], numero: '' }],
        manzanas: [{ ...detail().manzanas[0], numero: '' }],
        calles: [{ ...detail().calles[0], nombre: '' }],
      }),
    )

    expect(labels.get('lote-shared')).toBe('Lote')
    expect(labels.get('manzana-shared')).toBe('Manzana')
    expect(labels.get('calle-ca-1')).toBe('Calle')
  })

  it('skips entities that have no ring', () => {
    const labels = planLabelsFromLoteoDetail(
      detail({
        contorno: [],
        manzanas: [{
          id: 'mz-1',
          numero: '1',
          tieneAgua: false,
          tieneCloaca: false,
          tieneLuz: false,
          tieneGas: false,
          calleIds: [],
          poligono: [],
        }],
      }),
    )

    expect(labels.has('loteo')).toBe(false)
    expect(labels.has('manzana-mz-1')).toBe(false)
    expect(labels.has('lote-shared')).toBe(true)
  })
})
