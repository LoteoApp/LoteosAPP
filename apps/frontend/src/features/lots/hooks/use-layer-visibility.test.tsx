import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { useLayerVisibility } from './use-layer-visibility'
import { DXF_LAYERS } from '../types'

describe('useLayerVisibility', () => {
  it('starts with every layer visible', () => {
    const { result } = renderHook(() => useLayerVisibility())

    expect([...result.current.visibleLayers].sort()).toEqual([...DXF_LAYERS].sort())
  })

  it('replaces the visible set when toggled', () => {
    const { result } = renderHook(() => useLayerVisibility())

    act(() => {
      result.current.onVisibleLayersChange(new Set(['LOTEO']))
    })

    expect([...result.current.visibleLayers]).toEqual(['LOTEO'])
  })

  it('restores every layer on reset', () => {
    const { result } = renderHook(() => useLayerVisibility())

    act(() => {
      result.current.onVisibleLayersChange(new Set())
    })
    act(() => {
      result.current.reset()
    })

    expect([...result.current.visibleLayers].sort()).toEqual([...DXF_LAYERS].sort())
  })
})
