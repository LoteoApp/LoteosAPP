import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useLoteo } from './use-loteo'
import { ApiError } from '../../../shared/api/client'
import type { LoteoDetail } from '../types'

const getLoteoMock = vi.fn<
  (loteoId: string, token: string, options?: { signal?: AbortSignal }) => Promise<LoteoDetail>
>()

vi.mock('../api/get-loteo', () => ({
  getLoteo: (loteoId: string, token: string, options?: { signal?: AbortSignal }) =>
    getLoteoMock(loteoId, token, options),
}))

function detail(overrides: Partial<LoteoDetail> = {}): LoteoDetail {
  return {
    id: 'loteo-1',
    nombre: 'Las Acacias',
    ubicacion: 'Río Ceballos',
    descripcion: '',
    contorno: [],
    manzanas: [],
    lotes: [],
    calles: [],
    fechaCreacion: '2026-08-20T12:00:00Z',
    ...overrides,
  }
}

afterEach(() => {
  getLoteoMock.mockReset()
})

describe('useLoteo', () => {
  it('moves from loading to loaded', async () => {
    getLoteoMock.mockResolvedValue(detail({ nombre: 'Altos del Sur' }))

    const { result } = renderHook(() => useLoteo('loteo-1', 'token'))

    expect(result.current.status).toBe('loading')
    await waitFor(() => expect(result.current.status).toBe('loaded'))
    if (result.current.status === 'loaded') {
      expect(result.current.loteo.nombre).toBe('Altos del Sur')
    }
  })

  it('maps a 404 ApiError to not-found', async () => {
    getLoteoMock.mockRejectedValue(
      new ApiError('El loteo solicitado no existe', 'loteo_not_found', 404),
    )

    const { result } = renderHook(() => useLoteo('missing', 'token'))

    await waitFor(() => expect(result.current.status).toBe('not-found'))
  })

  it('maps any other failure to error with its message', async () => {
    getLoteoMock.mockRejectedValue(new Error('No se pudo cargar el loteo, intentá nuevamente.'))

    const { result } = renderHook(() => useLoteo('loteo-1', 'token'))

    await waitFor(() => expect(result.current.status).toBe('error'))
    if (result.current.status === 'error') {
      expect(result.current.message).toMatch(/no se pudo cargar el loteo/i)
    }
  })

  it('reports an expired session without calling the api when the token is empty', async () => {
    const { result } = renderHook(() => useLoteo('loteo-1', ''))

    await waitFor(() => expect(result.current.status).toBe('error'))
    if (result.current.status === 'error') {
      expect(result.current.message).toMatch(/sesión expiró/i)
    }
    expect(getLoteoMock).not.toHaveBeenCalled()
  })

  it('refetches when the loteo id changes', async () => {
    getLoteoMock.mockImplementation(async (loteoId) => detail({ id: loteoId, nombre: loteoId }))

    const { result, rerender } = renderHook(({ id }) => useLoteo(id, 'token'), {
      initialProps: { id: 'loteo-1' },
    })

    await waitFor(() => expect(result.current.status).toBe('loaded'))

    rerender({ id: 'loteo-2' })

    await waitFor(() => {
      expect(result.current.status).toBe('loaded')
      if (result.current.status === 'loaded') {
        expect(result.current.loteo.id).toBe('loteo-2')
      }
    })
    expect(getLoteoMock).toHaveBeenCalledTimes(2)
  })

  it('ignores a resolution that lands after unmount', async () => {
    let resolve: (value: LoteoDetail) => void = () => {}
    getLoteoMock.mockReturnValue(
      new Promise<LoteoDetail>((r) => {
        resolve = r
      }),
    )

    const { result, unmount } = renderHook(() => useLoteo('loteo-1', 'token'))
    unmount()
    resolve(detail())

    expect(result.current.status).toBe('loading')
  })
})
