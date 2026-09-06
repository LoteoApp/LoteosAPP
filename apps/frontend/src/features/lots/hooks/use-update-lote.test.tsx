import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useUpdateLote } from './use-update-lote'
import * as updateLoteModule from '../api/update-lote'
import { ApiError } from '../../../shared/api/client'
import type { LoteoLote } from '../types'
import type { UpdateLotePayload } from '../lib/loteFormValues'

const payload: UpdateLotePayload = {
  numero: '12',
  precio: 100,
  moneda: 'ARS',
  superficie: 200,
  caracteristicas: '',
}

const lote: LoteoLote = {
  id: 'lt-1',
  manzanaId: 'mz-1',
  numero: '12',
  precio: 100,
  moneda: 'ARS',
  superficie: 200,
  caracteristicas: '',
  poligono: [],
}

describe('useUpdateLote', () => {
  it('returns the updated lote on success', async () => {
    const update = vi.spyOn(updateLoteModule, 'updateLote').mockResolvedValue(lote)
    const { result } = renderHook(() => useUpdateLote('tok'))

    let saved: LoteoLote | null = null
    await act(async () => {
      saved = await result.current.update('loteo-1', 'lt-1', payload)
    })

    expect(update).toHaveBeenCalledWith('loteo-1', 'lt-1', payload, 'tok')
    expect(saved).toEqual(lote)
    expect(result.current.status).toBe('idle')
  })

  it('reports an expired session without calling the api when the token is null', async () => {
    const update = vi.spyOn(updateLoteModule, 'updateLote')
    const { result } = renderHook(() => useUpdateLote(null))

    let saved: LoteoLote | null = lote
    await act(async () => {
      saved = await result.current.update('loteo-1', 'lt-1', payload)
    })

    expect(update).not.toHaveBeenCalled()
    expect(saved).toBeNull()
    expect(result.current).toMatchObject({
      status: 'error',
      message: expect.stringMatching(/sesión expiró/i),
    })
  })

  it('maps known api codes onto form fields', async () => {
    vi.spyOn(updateLoteModule, 'updateLote').mockRejectedValue(
      new ApiError('Ya existe un lote con ese número en este loteo', 'lote_numero_in_use', 409),
    )
    const { result } = renderHook(() => useUpdateLote('tok'))

    await act(async () => {
      await result.current.update('loteo-1', 'lt-1', payload)
    })

    expect(result.current).toMatchObject({
      status: 'error',
      field: 'numero',
      message: expect.stringMatching(/ya existe/i),
    })
  })

  it('maps invalid_moneda onto the moneda field', async () => {
    vi.spyOn(updateLoteModule, 'updateLote').mockRejectedValue(
      new ApiError('La moneda debe ser un código de tres letras', 'invalid_moneda', 400),
    )
    const { result } = renderHook(() => useUpdateLote('tok'))

    await act(async () => {
      await result.current.update('loteo-1', 'lt-1', payload)
    })

    expect(result.current).toMatchObject({ status: 'error', field: 'moneda' })
  })

  it('maps a network error onto a banner without a field', async () => {
    vi.spyOn(updateLoteModule, 'updateLote').mockRejectedValue(
      new ApiError('No se pudo conectar con el servidor.', 'network_error', 0),
    )
    const { result } = renderHook(() => useUpdateLote('tok'))

    await act(async () => {
      await result.current.update('loteo-1', 'lt-1', payload)
    })

    expect(result.current).toMatchObject({
      status: 'error',
      message: expect.stringMatching(/conectar/i),
    })
    if (result.current.status === 'error') {
      expect(result.current.field).toBeUndefined()
    }
  })

  it('maps the remaining known codes onto their fields', async () => {
    const cases = [
      ['invalid_lote_numero', 'numero'],
      ['invalid_precio', 'precio'],
      ['invalid_superficie', 'superficie'],
      ['lote_caracteristicas_too_long', 'caracteristicas'],
      ['forbidden', undefined],
    ] as const

    for (const [code, field] of cases) {
      vi.spyOn(updateLoteModule, 'updateLote').mockRejectedValueOnce(
        new ApiError('x', code, 400),
      )
      const { result } = renderHook(() => useUpdateLote('tok'))
      await act(async () => {
        await result.current.update('loteo-1', 'lt-1', payload)
      })
      if (result.current.status === 'error') {
        expect(result.current.field).toBe(field)
      }
    }
  })

  it('clears the error on reset', async () => {
    vi.spyOn(updateLoteModule, 'updateLote').mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useUpdateLote('tok'))

    await act(async () => {
      await result.current.update('loteo-1', 'lt-1', payload)
    })
    act(() => {
      result.current.reset()
    })

    expect(result.current.status).toBe('idle')
  })

  it('ignores a response from an older request', async () => {
    let resolveFirst!: (value: LoteoLote) => void
    const first = new Promise<LoteoLote>((resolve) => {
      resolveFirst = resolve
    })
    const update = vi
      .spyOn(updateLoteModule, 'updateLote')
      .mockReturnValueOnce(first)
      .mockResolvedValueOnce(lote)
    const { result } = renderHook(() => useUpdateLote('tok'))

    let firstSave: Promise<LoteoLote | null> = Promise.resolve(null)
    await act(async () => {
      firstSave = result.current.update('loteo-1', 'lt-1', payload)
    })
    await act(async () => {
      await result.current.update('loteo-1', 'lt-1', payload)
    })

    resolveFirst(lote)
    await act(async () => {
      await firstSave
    })

    expect(update).toHaveBeenCalledTimes(2)
    expect(result.current.status).toBe('idle')
  })

  it('ignores a response after the access token changes', async () => {
    let resolveRequest!: (value: LoteoLote) => void
    const request = new Promise<LoteoLote>((resolve) => {
      resolveRequest = resolve
    })
    vi.spyOn(updateLoteModule, 'updateLote').mockReturnValue(request)
    const { result, rerender } = renderHook(
      ({ token }: { token: string | null }) => useUpdateLote(token),
      { initialProps: { token: 'tok' as string | null } },
    )

    let saved: LoteoLote | null = lote
    let savedPromise!: Promise<LoteoLote | null>
    act(() => {
      savedPromise = result.current.update('loteo-1', 'lt-1', payload)
    })
    rerender({ token: null })
    resolveRequest(lote)

    await act(async () => {
      saved = await savedPromise
    })

    expect(saved).toBeNull()
    expect(result.current.status).toBe('idle')
  })

  it('ignores a response after unmount', async () => {
    let resolveRequest!: (value: LoteoLote) => void
    const request = new Promise<LoteoLote>((resolve) => {
      resolveRequest = resolve
    })
    vi.spyOn(updateLoteModule, 'updateLote').mockReturnValue(request)
    const { result, unmount } = renderHook(() => useUpdateLote('tok'))

    let saved: LoteoLote | null = lote
    let savedPromise!: Promise<LoteoLote | null>
    act(() => {
      savedPromise = result.current.update('loteo-1', 'lt-1', payload)
    })
    unmount()
    resolveRequest(lote)

    await act(async () => {
      saved = await savedPromise
    })

    expect(saved).toBeNull()
  })
})
