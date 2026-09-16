import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { usePlanSelection } from './use-plan-selection'
import type { DxfPolygon } from '../types'

const square = [
  { x: 0, y: 0 },
  { x: 10, y: 0 },
  { x: 10, y: 10 },
  { x: 0, y: 10 },
]

function polygon(overrides: Partial<DxfPolygon>): DxfPolygon {
  return {
    id: 'lote-1',
    layer: 'LOTES',
    handle: null,
    vertices: square,
    entity: { kind: 'lote', id: 'lt-1' },
    ...overrides,
  }
}

const polygons: DxfPolygon[] = [
  polygon({ id: 'loteo', layer: 'LOTEO', entity: { kind: 'loteo' } }),
  polygon({
    id: 'manzana-1',
    layer: 'MANZANA',
    entity: { kind: 'manzana', id: 'mz-1' },
  }),
  polygon({ id: 'lote-1', layer: 'LOTES', entity: { kind: 'lote', id: 'lt-1' } }),
]

describe('usePlanSelection', () => {
  it('selects a polygon and exposes its entity', () => {
    const { result } = renderHook(() => usePlanSelection(polygons))

    act(() => {
      result.current.select('lote-1')
    })

    expect(result.current.selectedPolygonId).toBe('lote-1')
    expect(result.current.selected).toEqual({ kind: 'lote', id: 'lt-1' })
  })

  it('selects from a table row even when that lote has no polygon', () => {
    const { result } = renderHook(() => usePlanSelection(polygons))

    act(() => {
      result.current.selectEntity({ kind: 'lote', id: 'ghost' })
    })

    expect(result.current.selectedPolygonId).toBeNull()
    expect(result.current.selected).toEqual({ kind: 'lote', id: 'ghost' })
  })

  it('selects the matching polygon from an entity', () => {
    const { result } = renderHook(() => usePlanSelection(polygons))

    act(() => {
      result.current.selectEntity({ kind: 'manzana', id: 'mz-1' })
    })

    expect(result.current.selectedPolygonId).toBe('manzana-1')
    expect(result.current.selected).toEqual({ kind: 'manzana', id: 'mz-1' })
  })

  it('clears the selection', () => {
    const { result } = renderHook(() => usePlanSelection(polygons))

    act(() => {
      result.current.select('lote-1')
      result.current.clear()
    })

    expect(result.current.selectedPolygonId).toBeNull()
    expect(result.current.selected).toBeNull()
  })

  it('clears when the selected polygon disappears from the plan', () => {
    const { result, rerender } = renderHook(
      ({ items }: { items: DxfPolygon[] }) => usePlanSelection(items),
      { initialProps: { items: polygons } },
    )

    act(() => {
      result.current.select('lote-1')
    })

    rerender({ items: polygons.filter((item) => item.id !== 'lote-1') })

    expect(result.current.selectedPolygonId).toBeNull()
    expect(result.current.selected).toBeNull()
  })

  it('selects null to drop the current polygon', () => {
    const { result } = renderHook(() => usePlanSelection(polygons))

    act(() => {
      result.current.select('lote-1')
      result.current.select(null)
    })

    expect(result.current.selected).toBeNull()
  })
})
