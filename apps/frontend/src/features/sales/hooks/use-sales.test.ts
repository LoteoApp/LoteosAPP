import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../../../shared/api/client'
import { listSales } from '../api/sales'
import { useSales } from './use-sales'
import type { SalePage } from '../types'

vi.mock('../api/sales', () => ({ listSales: vi.fn() }))

const listSalesMock = vi.mocked(listSales)
const page: SalePage = { ventas: [], pagina: 1, porPagina: 25, total: 0, paginas: 0 }

afterEach(() => {
  listSalesMock.mockReset()
  vi.useRealTimers()
})

describe('useSales', () => {
  it('loads the page for the filters and refreshes on demand', async () => {
    listSalesMock.mockResolvedValue({ ...page, total: 3 })

    const { result } = renderHook(() => useSales('token', { estado: 'activa', pagina: 2 }))

    expect(result.current.isLoading).toBe(true)
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.page.total).toBe(3)
    expect(listSalesMock).toHaveBeenCalledWith('token', { estado: 'activa', pagina: 2 }, expect.anything())

    act(() => result.current.refresh())
    await waitFor(() => expect(listSalesMock).toHaveBeenCalledTimes(2))
  })

  it('debounces the search before asking the API', async () => {
    vi.useFakeTimers()
    listSalesMock.mockResolvedValue(page)

    const { rerender } = renderHook(({ q }) => useSales('token', { q }), { initialProps: { q: '' } })
    await act(async () => {
      await vi.runOnlyPendingTimersAsync()
    })
    expect(listSalesMock).toHaveBeenCalledTimes(1)

    rerender({ q: 'An' })
    rerender({ q: 'Ana' })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })
    expect(listSalesMock).toHaveBeenCalledTimes(1)

    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })
    expect(listSalesMock).toHaveBeenCalledTimes(2)
    expect(listSalesMock).toHaveBeenLastCalledWith('token', { q: 'Ana' }, expect.anything())
  })

  it('exposes the load failure and an empty page', async () => {
    listSalesMock.mockRejectedValue(new ApiError('Servicio no disponible', 'unavailable', 503))

    const { result } = renderHook(() => useSales('token', {}))

    await waitFor(() => expect(result.current.error).toBe('Servicio no disponible'))
    expect(result.current.page).toEqual(page)
    expect(result.current.isLoading).toBe(false)
  })

  it('hides an unexpected failure behind the generic message', async () => {
    listSalesMock.mockRejectedValue(new TypeError('Failed to fetch'))

    const { result } = renderHook(() => useSales('token', {}))

    await waitFor(() => expect(result.current.error).toMatch(/error inesperado/i))
    expect(result.current.page).toEqual(page)
  })

  it('stays idle without a token', () => {
    const { result } = renderHook(() => useSales('', {}))

    expect(result.current).toMatchObject({ page, isLoading: false, error: null })
    expect(listSalesMock).not.toHaveBeenCalled()
  })
})
