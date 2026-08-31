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

export type UseSaveLoteoResult = SaveState & {
  // save resolves to true when the loteo was created (even if the DXF upload
  // then failed, which is surfaced as dxfWarning), so the caller can clear
  // the form. It resolves to false when nothing was created.
  save: (fields: LoteoFieldValues, polygons: DxfPolygon[], dxfFile: File | null) => Promise<boolean>
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
        return false
      }
      if (fields.name.trim() === '') {
        setState({ status: 'error', message: NAME_REQUIRED_MESSAGE })
        return false
      }

      let payload: CreateLoteoPayload
      try {
        payload = buildCreateLoteoPayload(fields, polygons)
      } catch (error) {
        setState({
          status: 'error',
          message: error instanceof BuildLoteoPayloadError ? error.message : PLAN_PREP_ERROR_MESSAGE,
        })
        return false
      }

      setState({ status: 'saving' })

      let created: CreatedLoteo
      try {
        created = await createLoteo(payload, accessToken)
      } catch (error) {
        setState({ status: 'error', message: messageFromError(error) })
        return false
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
      return true
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
