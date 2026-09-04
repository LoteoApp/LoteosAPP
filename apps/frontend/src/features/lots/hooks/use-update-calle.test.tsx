import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useUpdateCalle } from './use-update-calle'
import * as updateCalleModule from '../api/update-calle'
import { ApiError } from '../../../shared/api/client'

const payload = {
  nombre: 'Los Álamos',
  tipo: 'asfalto',
}

describe('useUpdateCalle', () => {
  it('returns the updated calle on success', async () => {
    vi.spyOn(updateCalleModule, 'updateCalle').mockResolvedValue({
      id: 'ca-1',
      nombre: 'Los Álamos',
      tipo: 'asfalto',
      poligono: [],
    })
    const { result } = renderHook(() => useUpdateCalle('tok'))

    await act(async () => {
      await result.current.update('loteo-1', 'ca-1', payload)
    })

    expect(result.current.status).toBe('idle')
  })

  it('maps an invalid type onto the tipo field', async () => {
    vi.spyOn(updateCalleModule, 'updateCalle').mockRejectedValue(
      new ApiError('El tipo de calle debe ser asfalto, tierra, brosa o granito', 'invalid_calle_tipo', 400),
    )
    const { result } = renderHook(() => useUpdateCalle('tok'))

    await act(async () => {
      await result.current.update('loteo-1', 'ca-1', payload)
    })

    expect(result.current).toMatchObject({ status: 'error', field: 'tipo' })
  })

  it('reports an expired session without calling the api', async () => {
    const update = vi.spyOn(updateCalleModule, 'updateCalle')
    const { result } = renderHook(() => useUpdateCalle(null))

    await act(async () => {
      await result.current.update('loteo-1', 'ca-1', payload)
    })

    expect(update).not.toHaveBeenCalled()
    expect(result.current.status).toBe('error')
  })
})
