import { act, renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useArchivos, type ArchivoTarget } from './use-archivos'
import * as archivosApi from '../api/archivos'
import type { Archivo } from '../types'

const loteoTarget: ArchivoTarget = { kind: 'loteo', loteoId: 'loteo-1' }
const loteTarget: ArchivoTarget = { kind: 'lote', loteoId: 'loteo-1', loteId: 'lote-1' }

const archivo: Archivo = {
  id: 'archivo-1',
  categoria: 'foto',
  nombreOriginal: 'foto.jpg',
  mimeType: 'image/jpeg',
  hashSha256: 'abc123',
  fechaCreacion: '2026-01-01T00:00:00Z',
}

describe('useArchivos', () => {
  it('loads the loteo-level archivos on mount', async () => {
    const list = vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([archivo])
    const { result } = renderHook(() => useArchivos(loteoTarget, 'tok'))

    expect(result.current.isLoading).toBe(true)
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(list).toHaveBeenCalledWith('loteo-1', 'tok', expect.any(AbortSignal))
    expect(result.current.archivos).toEqual([archivo])
    expect(result.current.error).toBeNull()
  })

  it('loads lote-scoped archivos through the lote endpoint', async () => {
    const list = vi.spyOn(archivosApi, 'listLoteArchivos').mockResolvedValue([archivo])
    const { result } = renderHook(() => useArchivos(loteTarget, 'tok'))

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(list).toHaveBeenCalledWith('loteo-1', 'lote-1', 'tok', expect.any(AbortSignal))
    expect(result.current.archivos).toEqual([archivo])
  })

  it('reports the load error', async () => {
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useArchivos(loteoTarget, 'tok'))

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.error).toMatch(/inesperado/i)
    expect(result.current.archivos).toEqual([])
  })

  it('reports an expired session without calling the api when the token is null', () => {
    const list = vi.spyOn(archivosApi, 'listLoteoArchivos')
    const { result } = renderHook(() => useArchivos(loteoTarget, null))

    expect(list).not.toHaveBeenCalled()
    expect(result.current.isLoading).toBe(false)
    expect(result.current.error).toMatch(/sesión expiró/i)
  })

  it('refetches when the target changes from loteo to lote', async () => {
    const listLoteo = vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([archivo])
    const listLote = vi.spyOn(archivosApi, 'listLoteArchivos').mockResolvedValue([])
    const { result, rerender } = renderHook<ReturnType<typeof useArchivos>, { target: ArchivoTarget }>(
      ({ target }) => useArchivos(target, 'tok'),
      { initialProps: { target: loteoTarget } },
    )

    await waitFor(() => expect(result.current.archivos).toEqual([archivo]))

    rerender({ target: loteTarget })

    await waitFor(() => expect(listLote).toHaveBeenCalled())
    await waitFor(() => expect(result.current.archivos).toEqual([]))
    expect(listLoteo).toHaveBeenCalledTimes(1)
  })

  it('prepends a newly uploaded archivo', async () => {
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([])
    const upload = vi.spyOn(archivosApi, 'uploadLoteoArchivo').mockResolvedValue(archivo)
    const { result } = renderHook(() => useArchivos(loteoTarget, 'tok'))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    const file = new File(['x'], 'foto.jpg', { type: 'image/jpeg' })
    let ok = false
    await act(async () => {
      ok = await result.current.upload('foto', file)
    })

    expect(ok).toBe(true)
    expect(upload).toHaveBeenCalledWith('loteo-1', 'foto', file, 'tok')
    expect(result.current.archivos).toEqual([archivo])
  })

  it('uploads to the lote endpoint for a lote target', async () => {
    vi.spyOn(archivosApi, 'listLoteArchivos').mockResolvedValue([])
    const upload = vi.spyOn(archivosApi, 'uploadLoteArchivo').mockResolvedValue(archivo)
    const { result } = renderHook(() => useArchivos(loteTarget, 'tok'))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    const file = new File(['x'], 'plano.pdf', { type: 'application/pdf' })
    await act(async () => {
      await result.current.upload('plano', file)
    })

    expect(upload).toHaveBeenCalledWith('loteo-1', 'lote-1', 'plano', file, 'tok')
  })

  it('reports an upload failure without touching the current list', async () => {
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([archivo])
    vi.spyOn(archivosApi, 'uploadLoteoArchivo').mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useArchivos(loteoTarget, 'tok'))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    let ok = true
    await act(async () => {
      ok = await result.current.upload('foto', new File(['x'], 'foto.jpg'))
    })

    expect(ok).toBe(false)
    expect(result.current.error).toMatch(/inesperado/i)
    expect(result.current.archivos).toEqual([archivo])
  })

  it('does not upload when the token is null', async () => {
    const upload = vi.spyOn(archivosApi, 'uploadLoteoArchivo')
    const { result } = renderHook(() => useArchivos(loteoTarget, null))

    let ok = true
    await act(async () => {
      ok = await result.current.upload('foto', new File(['x'], 'foto.jpg'))
    })

    expect(ok).toBe(false)
    expect(upload).not.toHaveBeenCalled()
    expect(result.current.error).toMatch(/sesión expiró/i)
  })

  it('removes an archivo from the list on success', async () => {
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([archivo])
    const remove = vi.spyOn(archivosApi, 'deleteArchivo').mockResolvedValue(undefined)
    const { result } = renderHook(() => useArchivos(loteoTarget, 'tok'))
    await waitFor(() => expect(result.current.archivos).toEqual([archivo]))

    let ok = false
    await act(async () => {
      ok = await result.current.remove('archivo-1')
    })

    expect(ok).toBe(true)
    expect(remove).toHaveBeenCalledWith('loteo-1', 'archivo-1', 'tok')
    expect(result.current.archivos).toEqual([])
  })

  it('reports a delete failure and keeps the archivo in the list', async () => {
    vi.spyOn(archivosApi, 'listLoteoArchivos').mockResolvedValue([archivo])
    vi.spyOn(archivosApi, 'deleteArchivo').mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useArchivos(loteoTarget, 'tok'))
    await waitFor(() => expect(result.current.archivos).toEqual([archivo]))

    let ok = true
    await act(async () => {
      ok = await result.current.remove('archivo-1')
    })

    expect(ok).toBe(false)
    expect(result.current.archivos).toEqual([archivo])
    expect(result.current.error).toMatch(/inesperado/i)
  })
})
