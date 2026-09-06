import { act, renderHook, waitFor } from '@testing-library/react'
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

  it('replaces a lote in the loaded detail without refetching', async () => {
    getLoteoMock.mockResolvedValue(
      detail({
        lotes: [
          {
            id: 'lt-1',
            manzanaId: 'mz-1',
            numero: '7',
            precio: null,
            moneda: '',
            superficie: null,
            caracteristicas: '',
            poligono: [],
          },
        ],
      }),
    )

    const { result } = renderHook(() => useLoteo('loteo-1', 'token'))
    await waitFor(() => expect(result.current.status).toBe('loaded'))

    act(() => {
      result.current.replaceLote({
        id: 'lt-1',
        manzanaId: 'mz-1',
        numero: '12',
        precio: 100,
        moneda: 'ARS',
        superficie: 200,
        caracteristicas: 'Esquina',
        poligono: [],
      })
    })

    expect(getLoteoMock).toHaveBeenCalledTimes(1)
    if (result.current.status === 'loaded') {
      expect(result.current.loteo.lotes[0].numero).toBe('12')
    }
  })

  it('ignores replaceLote while the detail is not loaded', async () => {
    const { result } = renderHook(() => useLoteo('loteo-1', ''))

    act(() => {
      result.current.replaceLote({
        id: 'lt-1',
        manzanaId: 'mz-1',
        numero: '12',
        precio: null,
        moneda: '',
        superficie: null,
        caracteristicas: '',
        poligono: [],
      })
    })

    expect(result.current.status).toBe('error')
  })

  it('replaces a calle in the loaded detail without refetching', async () => {
    getLoteoMock.mockResolvedValue(
      detail({
        calles: [{ id: 'ca-1', nombre: '', tipo: '', poligono: [{ x: 0, y: 0 }, { x: 1, y: 0 }, { x: 1, y: 1 }] }],
      }),
    )

    const { result } = renderHook(() => useLoteo('loteo-1', 'token'))
    await waitFor(() => expect(result.current.status).toBe('loaded'))

    act(() => {
      result.current.replaceCalle({
        id: 'ca-1',
        nombre: 'Los Álamos',
        tipo: 'asfalto',
        poligono: [],
      })
    })

    expect(getLoteoMock).toHaveBeenCalledTimes(1)
    if (result.current.status === 'loaded') {
      expect(result.current.loteo.calles[0]).toMatchObject({
        nombre: 'Los Álamos',
        tipo: 'asfalto',
      })
      expect(result.current.loteo.calles[0].poligono).toHaveLength(3)
    }
  })

  it('replaces a manzana and preserves its geometry when needed', async () => {
    const originalPolygon = [{ x: 0, y: 0 }, { x: 1, y: 0 }, { x: 1, y: 1 }]
    getLoteoMock.mockResolvedValue(
      detail({
        manzanas: [{
          id: 'mz-1', numero: '1', tieneAgua: false, tieneCloaca: false,
          tieneLuz: false, tieneGas: false, calleIds: [], poligono: originalPolygon,
        }],
      }),
    )

    const { result } = renderHook(() => useLoteo('loteo-1', 'token'))
    await waitFor(() => expect(result.current.status).toBe('loaded'))

    act(() => {
      result.current.replaceManzana({
        id: 'mz-1', numero: '2', tieneAgua: true, tieneCloaca: false,
        tieneLuz: false, tieneGas: false, calleIds: [], poligono: [],
      })
    })

    if (result.current.status === 'loaded') {
      expect(result.current.loteo.manzanas[0]).toMatchObject({ numero: '2', tieneAgua: true })
      expect(result.current.loteo.manzanas[0].poligono).toEqual(originalPolygon)
    }
  })

  it('ignores replacements while the detail is not loaded', () => {
    const { result } = renderHook(() => useLoteo('loteo-1', ''))

    act(() => {
      result.current.replaceManzana({
        id: 'mz-1', numero: '2', tieneAgua: false, tieneCloaca: false,
        tieneLuz: false, tieneGas: false, calleIds: [], poligono: [],
      })
      result.current.replaceCalle({ id: 'ca-1', nombre: 'Nueva', tipo: '', poligono: [] })
    })

    expect(result.current.status).toBe('error')
  })

  it('ignores a rejected request after unmount', async () => {
    let reject: (reason?: unknown) => void = () => {}
    getLoteoMock.mockReturnValue(
      new Promise<LoteoDetail>((_, rejectPromise) => {
        reject = rejectPromise
      }),
    )

    const { result, unmount } = renderHook(() => useLoteo('loteo-1', 'token'))
    unmount()
    reject(new Error('cancelled'))
    await Promise.resolve()

    expect(result.current.status).toBe('loading')
  })
})
