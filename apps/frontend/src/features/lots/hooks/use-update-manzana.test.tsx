import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useUpdateManzana } from './use-update-manzana'
import * as updateManzanaModule from '../api/update-manzana'
import { ApiError } from '../../../shared/api/client'

const payload = {
  numero: 'A',
  tieneAgua: true,
  tieneCloaca: false,
  tieneLuz: false,
  tieneGas: false,
  calleIds: ['ca-1'],
}

describe('useUpdateManzana', () => {
  it('returns the updated manzana on success', async () => {
    vi.spyOn(updateManzanaModule, 'updateManzana').mockResolvedValue({
      id: 'mz-1',
      numero: 'A',
      tieneAgua: true,
      tieneCloaca: false,
      tieneLuz: false,
      tieneGas: false,
      calleIds: ['ca-1'],
      poligono: [],
    })
    const { result } = renderHook(() => useUpdateManzana('tok'))

    await act(async () => {
      await result.current.update('loteo-1', 'mz-1', payload)
    })

    expect(result.current.status).toBe('idle')
  })

  it('maps a duplicate number onto the numero field', async () => {
    vi.spyOn(updateManzanaModule, 'updateManzana').mockRejectedValue(
      new ApiError('Ya existe una manzana con ese número en este loteo', 'manzana_numero_in_use', 409),
    )
    const { result } = renderHook(() => useUpdateManzana('tok'))

    await act(async () => {
      await result.current.update('loteo-1', 'mz-1', payload)
    })

    expect(result.current).toMatchObject({ status: 'error', field: 'numero' })
  })

  it('reports an expired session without calling the api', async () => {
    const update = vi.spyOn(updateManzanaModule, 'updateManzana')
    const { result } = renderHook(() => useUpdateManzana(null))

    await act(async () => {
      await result.current.update('loteo-1', 'mz-1', payload)
    })

    expect(update).not.toHaveBeenCalled()
    expect(result.current.status).toBe('error')
  })
})
