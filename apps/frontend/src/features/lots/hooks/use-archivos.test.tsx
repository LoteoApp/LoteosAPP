import { act, renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useAttachments, type AttachmentTarget } from './use-archivos'
import * as archivosApi from '../api/archivos'
import type { Attachment } from '../types'

const loteoTarget: AttachmentTarget = { kind: 'loteo', loteoId: 'loteo-1' }
const loteTarget: AttachmentTarget = { kind: 'lote', loteoId: 'loteo-1', loteId: 'lote-1' }

const attachment: Attachment = {
  id: 'archivo-1',
  category: 'foto',
  nombreOriginal: 'foto.jpg',
  mimeType: 'image/jpeg',
  hashSha256: 'abc123',
  fechaCreacion: '2026-01-01T00:00:00Z',
}

describe('useAttachments', () => {
  it('loads the loteo-level attachments on mount', async () => {
    const list = vi.spyOn(archivosApi, 'listLoteoAttachments').mockResolvedValue([attachment])
    const { result } = renderHook(() => useAttachments(loteoTarget, 'tok'))

    expect(result.current.isLoading).toBe(true)
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(list).toHaveBeenCalledWith('loteo-1', 'tok', expect.any(AbortSignal))
    expect(result.current.attachments).toEqual([attachment])
    expect(result.current.error).toBeNull()
  })

  it('loads lote-scoped attachments through the lote endpoint', async () => {
    const list = vi.spyOn(archivosApi, 'listLoteAttachments').mockResolvedValue([attachment])
    const { result } = renderHook(() => useAttachments(loteTarget, 'tok'))

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(list).toHaveBeenCalledWith('loteo-1', 'lote-1', 'tok', expect.any(AbortSignal))
    expect(result.current.attachments).toEqual([attachment])
  })

  it('reports the load error', async () => {
    vi.spyOn(archivosApi, 'listLoteoAttachments').mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useAttachments(loteoTarget, 'tok'))

    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.error).toMatch(/inesperado/i)
    expect(result.current.attachments).toEqual([])
  })

  it('reports an expired session without calling the api when the token is null', () => {
    const list = vi.spyOn(archivosApi, 'listLoteoAttachments')
    const { result } = renderHook(() => useAttachments(loteoTarget, null))

    expect(list).not.toHaveBeenCalled()
    expect(result.current.isLoading).toBe(false)
    expect(result.current.error).toMatch(/sesión expiró/i)
  })

  it('refetches when the target changes from loteo to lote', async () => {
    const listLoteo = vi.spyOn(archivosApi, 'listLoteoAttachments').mockResolvedValue([attachment])
    const listLote = vi.spyOn(archivosApi, 'listLoteAttachments').mockResolvedValue([])
    const { result, rerender } = renderHook<ReturnType<typeof useAttachments>, { target: AttachmentTarget }>(
      ({ target }) => useAttachments(target, 'tok'),
      { initialProps: { target: loteoTarget } },
    )

    await waitFor(() => expect(result.current.attachments).toEqual([attachment]))

    rerender({ target: loteTarget })

    await waitFor(() => expect(listLote).toHaveBeenCalled())
    await waitFor(() => expect(result.current.attachments).toEqual([]))
    expect(listLoteo).toHaveBeenCalledTimes(1)
  })

  it('prepends a newly uploaded attachment', async () => {
    vi.spyOn(archivosApi, 'listLoteoAttachments').mockResolvedValue([])
    const upload = vi.spyOn(archivosApi, 'uploadLoteoAttachment').mockResolvedValue(attachment)
    const { result } = renderHook(() => useAttachments(loteoTarget, 'tok'))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    const file = new File(['x'], 'foto.jpg', { type: 'image/jpeg' })
    let ok = false
    await act(async () => {
      ok = await result.current.upload('foto', file)
    })

    expect(ok).toBe(true)
    expect(upload).toHaveBeenCalledWith('loteo-1', 'foto', file, 'tok')
    expect(result.current.attachments).toEqual([attachment])
  })

  it('uploads to the lote endpoint for a lote target', async () => {
    vi.spyOn(archivosApi, 'listLoteAttachments').mockResolvedValue([])
    const upload = vi.spyOn(archivosApi, 'uploadLoteAttachment').mockResolvedValue(attachment)
    const { result } = renderHook(() => useAttachments(loteTarget, 'tok'))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    const file = new File(['x'], 'plano.pdf', { type: 'application/pdf' })
    await act(async () => {
      await result.current.upload('plano', file)
    })

    expect(upload).toHaveBeenCalledWith('loteo-1', 'lote-1', 'plano', file, 'tok')
  })

  it('reports an upload failure without touching the current list', async () => {
    vi.spyOn(archivosApi, 'listLoteoAttachments').mockResolvedValue([attachment])
    vi.spyOn(archivosApi, 'uploadLoteoAttachment').mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useAttachments(loteoTarget, 'tok'))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    let ok = true
    await act(async () => {
      ok = await result.current.upload('foto', new File(['x'], 'foto.jpg'))
    })

    expect(ok).toBe(false)
    expect(result.current.error).toMatch(/inesperado/i)
    expect(result.current.attachments).toEqual([attachment])
  })

  it('does not upload when the token is null', async () => {
    const upload = vi.spyOn(archivosApi, 'uploadLoteoAttachment')
    const { result } = renderHook(() => useAttachments(loteoTarget, null))

    let ok = true
    await act(async () => {
      ok = await result.current.upload('foto', new File(['x'], 'foto.jpg'))
    })

    expect(ok).toBe(false)
    expect(upload).not.toHaveBeenCalled()
    expect(result.current.error).toMatch(/sesión expiró/i)
  })

  it('removes an attachment from the list on success', async () => {
    vi.spyOn(archivosApi, 'listLoteoAttachments').mockResolvedValue([attachment])
    const remove = vi.spyOn(archivosApi, 'deleteAttachment').mockResolvedValue(undefined)
    const { result } = renderHook(() => useAttachments(loteoTarget, 'tok'))
    await waitFor(() => expect(result.current.attachments).toEqual([attachment]))

    let ok = false
    await act(async () => {
      ok = await result.current.remove('archivo-1')
    })

    expect(ok).toBe(true)
    expect(remove).toHaveBeenCalledWith('loteo-1', 'archivo-1', 'tok')
    expect(result.current.attachments).toEqual([])
  })

  it('reports a delete failure and keeps the attachment in the list', async () => {
    vi.spyOn(archivosApi, 'listLoteoAttachments').mockResolvedValue([attachment])
    vi.spyOn(archivosApi, 'deleteAttachment').mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useAttachments(loteoTarget, 'tok'))
    await waitFor(() => expect(result.current.attachments).toEqual([attachment]))

    let ok = true
    await act(async () => {
      ok = await result.current.remove('archivo-1')
    })

    expect(ok).toBe(false)
    expect(result.current.attachments).toEqual([attachment])
    expect(result.current.error).toMatch(/inesperado/i)
  })
})
