import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { useDxfPlan } from './use-dxf-plan'
import { DXF_LAYERS, type DxfParseResult } from '../types'

const result: DxfParseResult = {
  polygons: [
    {
      id: 'p1',
      layer: 'LOTES',
      handle: '1A',
      vertices: [
        { x: 0, y: 0 },
        { x: 10, y: 0 },
        { x: 10, y: 10 },
      ],
    },
  ],
  issues: [
    {
      code: 'OVERLAPPING',
      layer: 'LOTES',
      message: 'Lotes superpuestos',
      handle: '1A',
      polygonId: 'p1',
      relatedPolygonId: 'p2',
    },
  ],
}

describe('useDxfPlan', () => {
  it('starts empty with every layer visible', () => {
    const { result: hookResult } = renderHook(() => useDxfPlan())

    expect(hookResult.current.hasPlan).toBe(false)
    expect(hookResult.current.fileName).toBeNull()
    expect(hookResult.current.error).toBeNull()
    expect(hookResult.current.polygons).toEqual([])
    expect(hookResult.current.issues).toEqual([])
    expect([...hookResult.current.visibleLayers].sort()).toEqual([...DXF_LAYERS].sort())
  })

  it('exposes the parsed plan after onParsed', () => {
    const { result: hookResult } = renderHook(() => useDxfPlan())

    act(() => {
      hookResult.current.onParsed(result, 'las-acacias.dxf')
    })

    expect(hookResult.current.hasPlan).toBe(true)
    expect(hookResult.current.fileName).toBe('las-acacias.dxf')
    expect(hookResult.current.polygons).toEqual(result.polygons)
    expect(hookResult.current.issues).toEqual(result.issues)
    expect(hookResult.current.error).toBeNull()
  })

  it('exposes the message and clears the plan after onError', () => {
    const { result: hookResult } = renderHook(() => useDxfPlan())

    act(() => {
      hookResult.current.onParsed(result, 'las-acacias.dxf')
    })
    act(() => {
      hookResult.current.onError('No se pudo interpretar el archivo DXF.')
    })

    expect(hookResult.current.error).toBe('No se pudo interpretar el archivo DXF.')
    expect(hookResult.current.hasPlan).toBe(false)
    expect(hookResult.current.fileName).toBeNull()
  })

  it('clears the plan on onCleared', () => {
    const { result: hookResult } = renderHook(() => useDxfPlan())

    act(() => {
      hookResult.current.onParsed(result, 'las-acacias.dxf')
    })
    act(() => {
      hookResult.current.onCleared()
    })

    expect(hookResult.current.hasPlan).toBe(false)
    expect(hookResult.current.fileName).toBeNull()
  })

  it('resets the plan and the visible layers', () => {
    const { result: hookResult } = renderHook(() => useDxfPlan())

    act(() => {
      hookResult.current.onParsed(result, 'las-acacias.dxf')
    })
    act(() => {
      hookResult.current.onVisibleLayersChange(new Set(['LOTES']))
    })
    act(() => {
      hookResult.current.reset()
    })

    expect(hookResult.current.hasPlan).toBe(false)
    expect([...hookResult.current.visibleLayers].sort()).toEqual([...DXF_LAYERS].sort())
  })
})
