import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { useLoteoFields } from './use-loteo-fields'

describe('useLoteoFields', () => {
  it('starts with empty values', () => {
    const { result } = renderHook(() => useLoteoFields())

    expect(result.current.values).toEqual({
      nombre: '',
      ubicacion: '',
      descripcion: '',
      inmobiliariaIds: [],
    })
  })

  it('updates values through onChange', () => {
    const { result } = renderHook(() => useLoteoFields())

    act(() => {
      result.current.onChange({
        nombre: 'Las Acacias',
        ubicacion: 'Cañuelas',
        descripcion: 'Segunda etapa',
        inmobiliariaIds: ['inm-altamira'],
      })
    })

    expect(result.current.values).toEqual({
      nombre: 'Las Acacias',
      ubicacion: 'Cañuelas',
      descripcion: 'Segunda etapa',
      inmobiliariaIds: ['inm-altamira'],
    })
  })

  it('resets to empty values', () => {
    const { result } = renderHook(() => useLoteoFields())

    act(() => {
      result.current.onChange({
        nombre: 'Las Acacias',
        ubicacion: 'Cañuelas',
        descripcion: 'Segunda etapa',
        inmobiliariaIds: ['inm-altamira'],
      })
    })
    act(() => {
      result.current.reset()
    })

    expect(result.current.values).toEqual({
      nombre: '',
      ubicacion: '',
      descripcion: '',
      inmobiliariaIds: [],
    })
  })
})
