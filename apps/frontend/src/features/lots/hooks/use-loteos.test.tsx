import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useLoteos } from './use-loteos'
import type { LoteoSummary } from '../types'

const listLoteosMock = vi.fn()

vi.mock('../api/list-loteos', () => ({
  listLoteos: (...args: unknown[]) => listLoteosMock(...args),
}))

function summary(overrides: Partial<LoteoSummary> = {}): LoteoSummary {
  return {
    id: 'loteo-1',
    nombre: 'Loteo Las Acacias',
    ubicacion: 'Río Ceballos, Córdoba',
    descripcion: '',
    cantidadManzanas: 12,
    cantidadLotes: 148,
    cantidadCalles: 8,
    tienePlano: true,
    ...overrides,
  }
}

afterEach(() => {
  listLoteosMock.mockReset()
})

describe('useLoteos', () => {
  it('loads the loteos on mount', async () => {
    listLoteosMock.mockResolvedValue([summary()])

    const { result } = renderHook(() => useLoteos('token-123', ''))

    expect(result.current.isLoading).toBe(true)

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.loteos).toHaveLength(1)
    expect(result.current.error).toBeNull()
    expect(listLoteosMock).toHaveBeenCalledWith(
      'token-123',
      expect.objectContaining({ q: undefined }),
    )
  })

  it('exposes the error message when the request fails', async () => {
    listLoteosMock.mockRejectedValue(new Error('No autorizado'))

    const { result } = renderHook(() => useLoteos('token-123', ''))

    await waitFor(() => expect(result.current.error).toBe('No autorizado'))
    expect(result.current.loteos).toEqual([])
  })

  it('drops the loaded result when a later request fails', async () => {
    listLoteosMock.mockResolvedValueOnce([summary()])

    const { result, rerender } = renderHook(({ search }) => useLoteos('token-123', search), {
      initialProps: { search: '' },
    })

    await waitFor(() => expect(result.current.loteos).toHaveLength(1))

    listLoteosMock.mockRejectedValueOnce(new Error('No autorizado'))
    rerender({ search: 'acacias' })

    await waitFor(() => expect(result.current.error).toBe('No autorizado'))
    expect(result.current.loteos).toEqual([])
  })

  it('falls back to a generic message for a non-Error rejection', async () => {
    listLoteosMock.mockRejectedValue('boom')

    const { result } = renderHook(() => useLoteos('token-123', ''))

    await waitFor(() => expect(result.current.error).toBe('Ocurrió un error inesperado.'))
  })

  it('ignores a rejection that resolves after the hook unmounts', async () => {
    let reject: (reason: unknown) => void = () => {}
    listLoteosMock.mockReturnValue(
      new Promise((_resolve, rejectFn) => {
        reject = rejectFn
      }),
    )

    const { result, unmount } = renderHook(() => useLoteos('token-123', ''))
    unmount()
    reject(new Error('too late'))
    await Promise.resolve()

    expect(result.current.error).toBeNull()
  })

  it('refetches with the debounced search text when it changes', async () => {
    listLoteosMock.mockResolvedValue([])

    const { rerender } = renderHook(({ search }) => useLoteos('token-123', search), {
      initialProps: { search: '' },
    })

    await waitFor(() => expect(listLoteosMock).toHaveBeenCalledTimes(1))

    rerender({ search: 'acacias' })

    await waitFor(() =>
      expect(listLoteosMock).toHaveBeenLastCalledWith(
        'token-123',
        expect.objectContaining({ q: 'acacias' }),
      ),
    )
  })

  it('shows the spinner again and clears the previous error on a new search', async () => {
    let resolveFirst: (value: LoteoSummary[]) => void = () => {}
    listLoteosMock.mockReturnValueOnce(
      new Promise<LoteoSummary[]>((resolve) => {
        resolveFirst = resolve
      }),
    )

    const { result, rerender } = renderHook(({ search }) => useLoteos('token-123', search), {
      initialProps: { search: '' },
    })

    resolveFirst([])
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    let resolveSecond: (value: LoteoSummary[]) => void = () => {}
    listLoteosMock.mockReturnValueOnce(
      new Promise<LoteoSummary[]>((resolve) => {
        resolveSecond = resolve
      }),
    )

    rerender({ search: 'acacias' })

    await waitFor(() => expect(result.current.isLoading).toBe(true))
    expect(result.current.error).toBeNull()

    resolveSecond([summary()])
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.loteos).toHaveLength(1)
  })
})
