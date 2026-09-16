import { useCallback, useState } from 'react'
import { messageFromError } from '../../../shared/api/client'
import { createLoteo, type CreatedLoteo } from '../api/create-loteo'
import { uploadLoteoDxf } from '../api/upload-loteo-dxf'
import type { LoteoFieldValues } from '../components/LoteoFields'
import {
  BuildLoteoPayloadError,
  buildCreateLoteoPayload,
  type CreateLoteoPayload,
} from '../lib/buildCreateLoteoPayload'
import type { DxfPolygon } from '../types'

type SaveState =
  | { status: 'idle' }
  | { status: 'saving' }
  | { status: 'success'; dxfWarning: string | null; isRetryingDxf: boolean }
  | { status: 'error'; message: string }

type PendingDxf = {
  loteoId: string
  file: File
}

// The id of the created loteo plus any DXF-upload warning, so the caller can
// clear the form and decide whether to navigate: a null dxfWarning means the
// alta finished cleanly, a non-null one means the retry UI is still needed here.
export type SaveOutcome = {
  loteoId: string
  dxfWarning: string | null
}

export type UseSaveLoteoResult = SaveState & {
  // save resolves to a SaveOutcome when the loteo was created (even if the DXF
  // upload then failed), and to null when nothing was created.
  save: (
    fields: LoteoFieldValues,
    polygons: DxfPolygon[],
    dxfFile: File | null,
  ) => Promise<SaveOutcome | null>
  retryDxf: () => Promise<boolean>
  reset: () => void
}

const SESSION_EXPIRED_MESSAGE = 'Tu sesión expiró. Volvé a iniciar sesión y probá de nuevo.'
const NAME_REQUIRED_MESSAGE = 'El nombre del loteo es obligatorio.'
const PLAN_PREP_ERROR_MESSAGE = 'No se pudo preparar el plano para guardarlo.'

export function useSaveLoteo(accessToken: string | null): UseSaveLoteoResult {
  const [state, setState] = useState<SaveState>({ status: 'idle' })
  const [pendingDxf, setPendingDxf] = useState<PendingDxf | null>(null)

  const reset = useCallback(() => {
    setPendingDxf(null)
    setState({ status: 'idle' })
  }, [])

  const save = useCallback(
    async (fields: LoteoFieldValues, polygons: DxfPolygon[], dxfFile: File | null) => {
      if (!accessToken) {
        setState({ status: 'error', message: SESSION_EXPIRED_MESSAGE })
        return null
      }
      if (fields.name.trim() === '') {
        setState({ status: 'error', message: NAME_REQUIRED_MESSAGE })
        return null
      }

      let payload: CreateLoteoPayload
      try {
        payload = buildCreateLoteoPayload(fields, polygons)
      } catch (error) {
        setState({
          status: 'error',
          message: error instanceof BuildLoteoPayloadError ? error.message : PLAN_PREP_ERROR_MESSAGE,
        })
        return null
      }

      setState({ status: 'saving' })

      let created: CreatedLoteo
      try {
        created = await createLoteo(payload, accessToken)
      } catch (error) {
        setState({ status: 'error', message: messageFromError(error) })
        return null
      }

      let dxfWarning: string | null = null
      if (dxfFile) {
        try {
          await uploadLoteoDxf(created.id, dxfFile, accessToken)
        } catch (error) {
          dxfWarning = `El loteo se creó, pero no se pudo guardar el archivo DXF: ${messageFromError(error)}`
          setPendingDxf({ loteoId: created.id, file: dxfFile })
        }
      }

      setState({ status: 'success', dxfWarning, isRetryingDxf: false })
      return { loteoId: created.id, dxfWarning }
    },
    [accessToken],
  )

  const retryDxf = useCallback(async () => {
    if (!pendingDxf) return false
    if (!accessToken) {
      setState({ status: 'success', dxfWarning: SESSION_EXPIRED_MESSAGE, isRetryingDxf: false })
      return false
    }

    setState((current) =>
      current.status === 'success' ? { ...current, isRetryingDxf: true } : current,
    )
    try {
      await uploadLoteoDxf(pendingDxf.loteoId, pendingDxf.file, accessToken)
    } catch (error) {
      setState({
        status: 'success',
        dxfWarning: `El loteo se creó, pero no se pudo guardar el archivo DXF: ${messageFromError(error)}`,
        isRetryingDxf: false,
      })
      return false
    }

    setPendingDxf(null)
    setState({ status: 'success', dxfWarning: null, isRetryingDxf: false })
    return true
  }, [accessToken, pendingDxf])

  return { ...state, save, retryDxf, reset }
}
