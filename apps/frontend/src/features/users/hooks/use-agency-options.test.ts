import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { listAgencies } from '../api/agencies'
import { useAgencyOptions } from './use-agency-options'

vi.mock('../api/agencies', () => ({
  listAgencies: vi.fn(),
}))

const listAgenciesMock = vi.mocked(listAgencies)

afterEach(() => {
  listAgenciesMock.mockReset()
})

describe('useAgencyOptions', () => {
  it('does not fetch while disabled', () => {
    const { result } = renderHook(() => useAgencyOptions('token', false))

    expect(listAgenciesMock).not.toHaveBeenCalled()
    expect(result.current.isLoading).toBe(false)
    expect(result.current.agencies).toEqual([])
  })

  it('loads the agencies once enabled', async () => {
    listAgenciesMock.mockResolvedValue([{ id: 'agency-1', razonSocial: 'Lotes del Sur' }])

    const { result, rerender } = renderHook(
      ({ enabled }) => useAgencyOptions('token', enabled),
      { initialProps: { enabled: false } },
    )
    expect(result.current.isLoading).toBe(false)

    rerender({ enabled: true })
    expect(result.current.isLoading).toBe(true)
    await waitFor(() => expect(result.current.agencies).toHaveLength(1))
    expect(result.current.isLoading).toBe(false)
    expect(listAgenciesMock).toHaveBeenCalledWith('token', expect.any(AbortSignal))
  })

  it('exposes the backend error message when the fetch fails', async () => {
    listAgenciesMock.mockRejectedValue(new Error('No se pudo cargar el catálogo'))

    const { result } = renderHook(() => useAgencyOptions('token', true))

    await waitFor(() => expect(result.current.error).toBe('No se pudo cargar el catálogo'))
    expect(result.current.agencies).toEqual([])
  })

  it('falls back to a generic message for a non-Error rejection', async () => {
    listAgenciesMock.mockRejectedValue('boom')

    const { result } = renderHook(() => useAgencyOptions('token', true))

    await waitFor(() => expect(result.current.error).toBe('Ocurrió un error inesperado.'))
  })

  it('re-fetches when the token changes', async () => {
    listAgenciesMock.mockResolvedValue([{ id: 'agency-1', razonSocial: 'Lotes del Sur' }])
    const { result, rerender } = renderHook(
      ({ token }) => useAgencyOptions(token, true),
      { initialProps: { token: 'token-1' } },
    )
    await waitFor(() => expect(result.current.agencies).toHaveLength(1))

    listAgenciesMock.mockResolvedValue([{ id: 'agency-2', razonSocial: 'Altamira' }])
    rerender({ token: 'token-2' })
    expect(result.current.isLoading).toBe(true)
    await waitFor(() => expect(result.current.agencies).toEqual([{ id: 'agency-2', razonSocial: 'Altamira' }]))
    expect(listAgenciesMock).toHaveBeenCalledTimes(2)
  })
})
