import { apiUrl } from '../../../shared/config/env'
import type { Surveyor, SurveyorFormValues } from '../types'

const SURVEYORS_PATH = '/api/v1/agrimensores'
const GENERIC_ERROR = 'No se pudo completar la operación, intentá nuevamente.'

export type CreatedSurveyor = {
  surveyor: Surveyor
  temporaryPassword: string
}

async function readErrorMessage(response: Response): Promise<string> {
  try {
    const body: unknown = await response.json()
    if (body && typeof body === 'object' && 'message' in body) {
      const { message } = body as { message?: unknown }
      if (typeof message === 'string' && message !== '') {
        return message
      }
    }
  } catch {
    return GENERIC_ERROR
  }

  return GENERIC_ERROR
}

async function send(token: string, path: string, init: RequestInit = {}): Promise<Response> {
  const response = await fetch(`${apiUrl}${path}`, {
    ...init,
    headers: {
      ...(init.body === undefined ? {} : { 'Content-Type': 'application/json' }),
      Authorization: `Bearer ${token}`,
    },
  })

  if (!response.ok) {
    throw new Error(await readErrorMessage(response))
  }

  return response
}

export async function listSurveyors(token: string, includeInactive: boolean): Promise<Surveyor[]> {
  const query = includeInactive ? '?incluirBajas=true' : ''
  const response = await send(token, `${SURVEYORS_PATH}${query}`)
  const body: { agrimensores?: Surveyor[] } = await response.json()

  return body.agrimensores ?? []
}

export async function createSurveyor(
  token: string,
  values: SurveyorFormValues,
): Promise<CreatedSurveyor> {
  const response = await send(token, SURVEYORS_PATH, {
    method: 'POST',
    body: JSON.stringify(values),
  })
  const body: Surveyor & { temporaryPassword?: string } = await response.json()

  return { surveyor: body, temporaryPassword: body.temporaryPassword ?? '' }
}

export async function updateSurveyor(
  token: string,
  id: string,
  values: Pick<SurveyorFormValues, 'nombre' | 'apellido'>,
): Promise<Surveyor> {
  const response = await send(token, `${SURVEYORS_PATH}/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(values),
  })

  return response.json() as Promise<Surveyor>
}

export async function deactivateSurveyor(token: string, id: string): Promise<void> {
  await send(token, `${SURVEYORS_PATH}/${id}`, { method: 'DELETE' })
}
