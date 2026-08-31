import { describe, expect, it } from 'vitest'
import { BuildLoteoPayloadError, buildCreateLoteoPayload } from './buildCreateLoteoPayload'
import type { LoteoFieldValues } from '../components/LoteoFields'
import type { DxfLayer, DxfPoint, DxfPolygon } from '../types'

function rect(x: number, y: number, size = 10): DxfPoint[] {
  return [
    { x, y },
    { x: x + size, y },
    { x: x + size, y: y + size },
    { x, y: y + size },
  ]
}

function polygon(id: string, layer: DxfLayer, vertices: DxfPoint[], handle: string | null = null): DxfPolygon {
  return { id, layer, handle, vertices }
}

const fields: LoteoFieldValues = {
  name: '  Las Acacias  ',
  location: '  Córdoba ',
  description: ' Al norte ',
}

describe('buildCreateLoteoPayload', () => {
  it('sends no plan and trims the text fields when there are no polygons', () => {
    const payload = buildCreateLoteoPayload(fields, [])

    expect(payload).toEqual({
      nombre: 'Las Acacias',
      ubicacion: 'Córdoba',
      descripcion: 'Al norte',
      plano: null,
    })
  })

  it('maps every layer and points each lote at the manzana that contains it', () => {
    const polygons: DxfPolygon[] = [
      polygon('loteo-0', 'LOTEO', rect(0, 0, 200), '1A'),
      polygon('manzana-0', 'MANZANA', rect(0, 0, 20), '2A'),
      polygon('manzana-1', 'MANZANA', rect(100, 0, 20)),
      polygon('lote-0', 'LOTES', rect(4, 4)),
      polygon('lote-1', 'LOTES', rect(104, 4)),
      polygon('calle-0', 'CALLE', rect(0, 40)),
    ]

    const { plano } = buildCreateLoteoPayload(fields, polygons)

    expect(plano).not.toBeNull()
    expect(plano?.loteo).toEqual({ handle: '1A', vertices: rect(0, 0, 200) })
    expect(plano?.manzanas.map((manzana) => manzana.ref)).toEqual(['manzana-0', 'manzana-1'])
    expect(plano?.manzanas[1].handle).toBe('')
    expect(plano?.lotes[0].manzanaRef).toBe('manzana-0')
    expect(plano?.lotes[1].manzanaRef).toBe('manzana-1')
    expect(plano?.calles).toEqual([{ handle: '', vertices: rect(0, 40) }])
  })

  it('falls back to the nearest manzana when a lote sits outside all of them', () => {
    const polygons: DxfPolygon[] = [
      polygon('loteo-0', 'LOTEO', rect(0, 0, 200)),
      polygon('manzana-near', 'MANZANA', rect(0, 0, 20)),
      polygon('manzana-far', 'MANZANA', rect(500, 0, 20)),
      polygon('lote-0', 'LOTES', rect(-40, -40)),
    ]

    const { plano } = buildCreateLoteoPayload(fields, polygons)

    expect(plano?.lotes[0].manzanaRef).toBe('manzana-near')
  })

  it('rejects a plan without a LOTEO polygon', () => {
    const polygons: DxfPolygon[] = [polygon('manzana-0', 'MANZANA', rect(0, 0, 20))]

    expect(() => buildCreateLoteoPayload(fields, polygons)).toThrow(BuildLoteoPayloadError)
    expect(() => buildCreateLoteoPayload(fields, polygons)).toThrow(/capa LOTEO/)
  })

  it('rejects a plan with more than one LOTEO polygon', () => {
    const polygons: DxfPolygon[] = [
      polygon('loteo-0', 'LOTEO', rect(0, 0, 200)),
      polygon('loteo-1', 'LOTEO', rect(300, 0, 200)),
    ]

    expect(() => buildCreateLoteoPayload(fields, polygons)).toThrow(/único polígono/)
  })

  it('rejects a plan with lotes but no manzana', () => {
    const polygons: DxfPolygon[] = [
      polygon('loteo-0', 'LOTEO', rect(0, 0, 200)),
      polygon('lote-0', 'LOTES', rect(4, 4)),
    ]

    expect(() => buildCreateLoteoPayload(fields, polygons)).toThrow(/ninguna manzana/)
  })
})
