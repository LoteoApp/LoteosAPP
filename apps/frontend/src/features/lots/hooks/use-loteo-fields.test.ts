import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { useLoteoFields } from './use-loteo-fields'

describe('useLoteoFields', () => {
  it('starts with empty values', () => {
    const { result } = renderHook(() => useLoteoFields())

    expect(result.current.values).toEqual({
      name: '',
      location: '',
      description: '',
      agencyIds: [],
    })
  })

  it('updates values through onChange', () => {
    const { result } = renderHook(() => useLoteoFields())

    act(() => {
      result.current.onChange({
        name: 'Las Acacias',
        location: 'Cañuelas',
        description: 'Segunda etapa',
        agencyIds: ['inm-altamira'],
      })
    })

    expect(result.current.values).toEqual({
      name: 'Las Acacias',
      location: 'Cañuelas',
      description: 'Segunda etapa',
      agencyIds: ['inm-altamira'],
    })
  })

  it('resets to empty values', () => {
    const { result } = renderHook(() => useLoteoFields())

    act(() => {
      result.current.onChange({
        name: 'Las Acacias',
        location: 'Cañuelas',
        description: 'Segunda etapa',
        agencyIds: ['inm-altamira'],
      })
    })
    act(() => {
      result.current.reset()
    })

    expect(result.current.values).toEqual({
      name: '',
      location: '',
      description: '',
      agencyIds: [],
    })
  })
})
