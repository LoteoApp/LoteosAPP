import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useSaveLoteo } from './use-save-loteo'
import * as createLoteoModule from '../api/create-loteo'
import * as uploadModule from '../api/upload-loteo-dxf'
import { ApiError } from '../../../shared/api/client'
import type { LoteoFieldValues } from '../components/LoteoFields'
import type { DxfLayer, DxfPoint, DxfPolygon } from '../types'

const fields: LoteoFieldValues = {
  name: 'Las Acacias',
  location: 'Córdoba',
  description: '',
}

function rect(x: number, y: number, size = 10): DxfPoint[] {
  return [
    { x, y },
    { x: x + size, y },
    { x: x + size, y: y + size },
    { x, y: y + size },
  ]
}

function polygon(id: string, layer: DxfLayer, vertices: DxfPoint[]): DxfPolygon {
  return { id, layer, handle: null, vertices }
}

const planPolygons: DxfPolygon[] = [
  polygon('loteo-0', 'LOTEO', rect(0, 0, 100)),
  polygon('manzana-0', 'MANZANA', rect(0, 0, 20)),
  polygon('lote-0', 'LOTES', rect(4, 4)),
]

const dxfFile = new File(['0\nEOF\n'], 'plano.dxf', { type: 'application/dxf' })

function setup(token: string | null = 'tok') {
  return renderHook(({ accessToken }) => useSaveLoteo(accessToken), {
    initialProps: { accessToken: token },
  })
}

describe('useSaveLoteo', () => {
  it('creates a loteo without a DXF', async () => {
    const createLoteo = vi
      .spyOn(createLoteoModule, 'createLoteo')
      .mockResolvedValue({ id: 'loteo-1', nombre: 'Las Acacias' })
    const upload = vi.spyOn(uploadModule, 'uploadLoteoDxf').mockResolvedValue()

    const { result } = setup()
    await act(async () => {
      await result.current.save(fields, [], null)
    })

    expect(createLoteo).toHaveBeenCalledWith(
      { nombre: 'Las Acacias', ubicacion: 'Córdoba', descripcion: '', plano: null },
      'tok',
    )
    expect(upload).not.toHaveBeenCalled()
    expect(result.current).toMatchObject({ status: 'success', dxfWarning: null })
  })

  it('uploads the DXF after creating the loteo', async () => {
    vi.spyOn(createLoteoModule, 'createLoteo').mockResolvedValue({ id: 'loteo-1', nombre: 'x' })
    const upload = vi.spyOn(uploadModule, 'uploadLoteoDxf').mockResolvedValue()

    const { result } = setup()
    await act(async () => {
      await result.current.save(fields, planPolygons, dxfFile)
    })

    expect(upload).toHaveBeenCalledWith('loteo-1', dxfFile, 'tok')
    expect(result.current).toMatchObject({ status: 'success', dxfWarning: null })
  })

  it('reports a warning when the loteo is created but the DXF upload fails', async () => {
    vi.spyOn(createLoteoModule, 'createLoteo').mockResolvedValue({ id: 'loteo-1', nombre: 'x' })
    vi.spyOn(uploadModule, 'uploadLoteoDxf').mockRejectedValue(
      new ApiError('El almacenamiento no está disponible', 'storage_unavailable', 503),
    )

    const { result } = setup()
    await act(async () => {
      await result.current.save(fields, planPolygons, dxfFile)
    })

    expect(result.current.status).toBe('success')
    if (result.current.status === 'success') {
      expect(result.current.dxfWarning).toMatch(/no se pudo guardar el archivo DXF/i)
      expect(result.current.dxfWarning).toMatch(/almacenamiento/i)
    }
  })

  it('retries only the pending DXF upload without creating another loteo', async () => {
    const create = vi
      .spyOn(createLoteoModule, 'createLoteo')
      .mockResolvedValue({ id: 'loteo-1', nombre: 'x' })
    const upload = vi
      .spyOn(uploadModule, 'uploadLoteoDxf')
      .mockRejectedValueOnce(new ApiError('Storage unavailable', 'storage_unavailable', 503))
      .mockResolvedValueOnce()

    const { result } = setup()
    await act(async () => {
      await result.current.save(fields, planPolygons, dxfFile)
    })
    await act(async () => {
      await result.current.retryDxf()
    })

    expect(create).toHaveBeenCalledTimes(1)
    expect(upload).toHaveBeenCalledTimes(2)
    expect(upload).toHaveBeenLastCalledWith('loteo-1', dxfFile, 'tok')
    expect(result.current).toMatchObject({
      status: 'success',
      dxfWarning: null,
      isRetryingDxf: false,
    })
  })

  it('keeps the pending DXF available when the retry fails', async () => {
    vi.spyOn(createLoteoModule, 'createLoteo').mockResolvedValue({ id: 'loteo-1', nombre: 'x' })
    const upload = vi
      .spyOn(uploadModule, 'uploadLoteoDxf')
      .mockRejectedValueOnce(new ApiError('Storage unavailable', 'storage_unavailable', 503))
      .mockRejectedValueOnce(new ApiError('Still unavailable', 'storage_unavailable', 503))
      .mockResolvedValueOnce()

    const { result } = setup()
    await act(async () => {
      await result.current.save(fields, planPolygons, dxfFile)
    })
    await act(async () => {
      expect(await result.current.retryDxf()).toBe(false)
    })

    expect(result.current).toMatchObject({
      status: 'success',
      isRetryingDxf: false,
    })
    if (result.current.status === 'success') {
      expect(result.current.dxfWarning).toMatch(/still unavailable/i)
    }

    await act(async () => {
      expect(await result.current.retryDxf()).toBe(true)
    })
    expect(upload).toHaveBeenCalledTimes(3)
  })

  it('does not retry a pending DXF after the session expires', async () => {
    vi.spyOn(createLoteoModule, 'createLoteo').mockResolvedValue({ id: 'loteo-1', nombre: 'x' })
    const upload = vi
      .spyOn(uploadModule, 'uploadLoteoDxf')
      .mockRejectedValueOnce(new ApiError('Storage unavailable', 'storage_unavailable', 503))

    const { result, rerender } = setup()
    await act(async () => {
      await result.current.save(fields, planPolygons, dxfFile)
    })
    rerender({ accessToken: null })
    await act(async () => {
      expect(await result.current.retryDxf()).toBe(false)
    })

    expect(upload).toHaveBeenCalledTimes(1)
    expect(result.current).toMatchObject({ status: 'success', isRetryingDxf: false })
    if (result.current.status === 'success') {
      expect(result.current.dxfWarning).toMatch(/sesión expiró/i)
    }
  })

  it('does nothing when there is no pending DXF', async () => {
    const upload = vi.spyOn(uploadModule, 'uploadLoteoDxf')
    const { result } = setup()

    await act(async () => {
      expect(await result.current.retryDxf()).toBe(false)
    })

    expect(upload).not.toHaveBeenCalled()
    expect(result.current.status).toBe('idle')
  })

  it('surfaces the backend message when creating the loteo fails', async () => {
    vi.spyOn(createLoteoModule, 'createLoteo').mockRejectedValue(
      new ApiError('Falta el nombre', 'invalid_loteo_nombre', 400),
    )

    const { result } = setup()
    await act(async () => {
      await result.current.save(fields, [], null)
    })

    expect(result.current).toMatchObject({ status: 'error', message: 'Falta el nombre' })
  })

  it('does not call the API without a session token', async () => {
    const createLoteo = vi.spyOn(createLoteoModule, 'createLoteo')

    const { result } = setup(null)
    await act(async () => {
      await result.current.save(fields, [], null)
    })

    expect(createLoteo).not.toHaveBeenCalled()
    expect(result.current).toMatchObject({ status: 'error' })
    if (result.current.status === 'error') {
      expect(result.current.message).toMatch(/sesión expiró/i)
    }
  })

  it('requires a name before calling the API', async () => {
    const createLoteo = vi.spyOn(createLoteoModule, 'createLoteo')

    const { result } = setup()
    await act(async () => {
      await result.current.save({ ...fields, name: '   ' }, [], null)
    })

    expect(createLoteo).not.toHaveBeenCalled()
    if (result.current.status === 'error') {
      expect(result.current.message).toMatch(/obligatorio/i)
    }
  })

  it('blocks a plan the backend would reject before any request', async () => {
    const createLoteo = vi.spyOn(createLoteoModule, 'createLoteo')

    const { result } = setup()
    await act(async () => {
      await result.current.save(fields, [
        polygon('loteo-0', 'LOTEO', rect(0, 0, 100)),
        polygon('lote-0', 'LOTES', rect(4, 4)),
      ], null)
    })

    expect(createLoteo).not.toHaveBeenCalled()
    if (result.current.status === 'error') {
      expect(result.current.message).toMatch(/ninguna manzana/i)
    }
  })

  it('clears the state on reset', async () => {
    vi.spyOn(createLoteoModule, 'createLoteo').mockResolvedValue({ id: 'loteo-1', nombre: 'x' })

    const { result } = setup()
    await act(async () => {
      await result.current.save(fields, [], null)
    })
    act(() => {
      result.current.reset()
    })

    expect(result.current.status).toBe('idle')
  })
})
