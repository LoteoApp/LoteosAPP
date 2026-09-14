import { renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import { useSaleSellers } from './use-sale-sellers'
import type { SellerOption } from '../types'

const seller: SellerOption = { id: 'us-1', nombre: 'Sofía', apellido: 'Luna', rol: 'administrativo' }

describe('useSaleSellers', () => {
  it('does not ask for sellers until there is a loteo', () => {
    const loadSellers = vi.fn()

    const { result } = renderHook(() => useSaleSellers('', loadSellers))

    expect(result.current).toEqual({ sellers: [], isLoading: false, error: null })
    expect(loadSellers).not.toHaveBeenCalled()
  })

  it('loads the sellers of the loteo and reloads when it changes', async () => {
    const loadSellers = vi.fn().mockResolvedValue([seller])

    const { result, rerender } = renderHook(({ loteoId }) => useSaleSellers(loteoId, loadSellers), {
      initialProps: { loteoId: 'loteo-1' },
    })

    expect(result.current.isLoading).toBe(true)
    await waitFor(() => expect(result.current.sellers).toEqual([seller]))
    expect(result.current.isLoading).toBe(false)

    rerender({ loteoId: 'loteo-2' })

    expect(result.current.sellers).toEqual([])
    expect(result.current.isLoading).toBe(true)
    await waitFor(() => expect(loadSellers).toHaveBeenLastCalledWith('loteo-2', expect.anything()))
    await waitFor(() => expect(result.current.isLoading).toBe(false))
  })

  it('exposes the backend message when the catalog fails', async () => {
    const loadSellers = vi.fn().mockRejectedValue(new ApiError('Sin permisos', 'forbidden', 403))

    const { result } = renderHook(() => useSaleSellers('loteo-1', loadSellers))

    await waitFor(() => expect(result.current.error).toBe('Sin permisos'))
    expect(result.current.isLoading).toBe(false)
    expect(result.current.sellers).toEqual([])
  })

  it('ignores a response that arrives after unmounting', async () => {
    let resolve: (sellers: SellerOption[]) => void = () => {}
    const loadSellers = vi.fn(
      () => new Promise<SellerOption[]>((res) => {
        resolve = res
      }),
    )

    const { result, unmount } = renderHook(() => useSaleSellers('loteo-1', loadSellers))
    const [, signal] = loadSellers.mock.calls[0] as unknown as [string, AbortSignal]

    unmount()
    expect(signal.aborted).toBe(true)
    resolve([seller])
    await Promise.resolve()

    expect(result.current.sellers).toEqual([])
  })
})
