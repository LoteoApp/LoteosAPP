import { apiFetch } from '../../../shared/api/client'

// uploadLoteoDxf sends the original DXF for an already-created loteo. The
// backend stores it under a versioned R2 key and records it in archivos.
export function uploadLoteoDxf(loteoId: string, file: File, token: string): Promise<void> {
  const form = new FormData()
  form.append('archivo', file, file.name)

  return apiFetch<void>(`/api/v1/loteos/${encodeURIComponent(loteoId)}/dxf`, {
    method: 'PUT',
    body: form,
    token,
  })
}
