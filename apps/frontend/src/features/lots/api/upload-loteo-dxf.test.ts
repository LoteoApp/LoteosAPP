import { describe, expect, it, vi } from 'vitest'
import { uploadLoteoDxf } from './upload-loteo-dxf'
import * as client from '../../../shared/api/client'

describe('uploadLoteoDxf', () => {
  it('PUTs the file as multipart form data to the loteo dxf endpoint', async () => {
    const apiFetch = vi.spyOn(client, 'apiFetch').mockResolvedValue(undefined)
    const file = new File(['0\nEOF\n'], 'plano.dxf', { type: 'application/dxf' })

    await uploadLoteoDxf('loteo 1/x', file, 'tok')

    expect(apiFetch).toHaveBeenCalledTimes(1)
    const [path, options] = apiFetch.mock.calls[0]
    expect(path).toBe('/api/v1/loteos/loteo%201%2Fx/dxf')
    expect(options?.method).toBe('PUT')
    expect(options?.token).toBe('tok')
    expect(options?.body).toBeInstanceOf(FormData)
    const sent = (options?.body as FormData).get('archivo')
    expect(sent).toBeInstanceOf(File)
    expect((sent as File).name).toBe('plano.dxf')
  })
})
