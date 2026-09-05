import { useCallback, useEffect, useState } from 'react'
import { messageFromError } from '../../../shared/api/client'
import {
  deleteArchivo,
  listLoteArchivos,
  listLoteoArchivos,
  uploadLoteArchivo,
  uploadLoteoArchivo,
} from '../api/archivos'
import type { Archivo, ArchivoCategoria } from '../types'

export type ArchivoTarget =
  | { kind: 'loteo'; loteoId: string }
  | { kind: 'lote'; loteoId: string; loteId: string }

const SESSION_EXPIRED_MESSAGE = 'Tu sesión expiró. Volvé a iniciar sesión y probá de nuevo.'

function list(target: ArchivoTarget, token: string, signal: AbortSignal): Promise<Archivo[]> {
  return target.kind === 'loteo'
    ? listLoteoArchivos(target.loteoId, token, signal)
    : listLoteArchivos(target.loteoId, target.loteId, token, signal)
}

function upload(
  target: ArchivoTarget,
  categoria: ArchivoCategoria,
  file: File,
  token: string,
): Promise<Archivo> {
  return target.kind === 'loteo'
    ? uploadLoteoArchivo(target.loteoId, categoria, file, token)
    : uploadLoteArchivo(target.loteoId, target.loteId, categoria, file, token)
}

type LoadState = { archivos: Archivo[]; isLoading: boolean; error: string | null }

function pending(token: string | null): LoadState {
  return token
    ? { archivos: [], isLoading: true, error: null }
    : { archivos: [], isLoading: false, error: SESSION_EXPIRED_MESSAGE }
}

export type UseArchivos = {
  archivos: Archivo[]
  isLoading: boolean
  isSubmitting: boolean
  error: string | null
  upload: (categoria: ArchivoCategoria, file: File) => Promise<boolean>
  remove: (archivoId: string) => Promise<boolean>
}

export function useArchivos(target: ArchivoTarget, token: string | null): UseArchivos {
  const loteoId = target.loteoId
  const loteId = target.kind === 'lote' ? target.loteId : null

  const [state, setState] = useState<LoadState>(() => pending(token))
  const [isSubmitting, setIsSubmitting] = useState(false)

  // Reset to the pending state during render when the request key changes,
  // so the UI never shows the previous target's archivos while the next
  // batch loads.
  const requestKey = JSON.stringify([token, loteoId, loteId])
  const [loadedKey, setLoadedKey] = useState(requestKey)
  if (requestKey !== loadedKey) {
    setLoadedKey(requestKey)
    setState(pending(token))
  }

  useEffect(() => {
    if (!token) {
      return
    }

    const controller = new AbortController()

    list(target, token, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) {
          setState({ archivos: loaded, isLoading: false, error: null })
        }
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setState({ archivos: [], isLoading: false, error: messageFromError(loadError) })
        }
      })

    return () => {
      controller.abort()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loteoId, loteId, token])

  const doUpload = useCallback(
    async (categoria: ArchivoCategoria, file: File): Promise<boolean> => {
      if (!token) {
        setState((current) => ({ ...current, error: SESSION_EXPIRED_MESSAGE }))
        return false
      }

      setIsSubmitting(true)
      try {
        const archivo = await upload(target, categoria, file, token)
        setState((current) => ({ archivos: [archivo, ...current.archivos], isLoading: false, error: null }))
        return true
      } catch (uploadError) {
        setState((current) => ({ ...current, error: messageFromError(uploadError) }))
        return false
      } finally {
        setIsSubmitting(false)
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [loteoId, loteId, token],
  )

  const remove = useCallback(
    async (archivoId: string): Promise<boolean> => {
      if (!token) {
        setState((current) => ({ ...current, error: SESSION_EXPIRED_MESSAGE }))
        return false
      }

      setIsSubmitting(true)
      try {
        await deleteArchivo(loteoId, archivoId, token)
        setState((current) => ({
          ...current,
          archivos: current.archivos.filter((archivo) => archivo.id !== archivoId),
          error: null,
        }))
        return true
      } catch (deleteError) {
        setState((current) => ({ ...current, error: messageFromError(deleteError) }))
        return false
      } finally {
        setIsSubmitting(false)
      }
    },
    [loteoId, token],
  )

  return { ...state, isSubmitting, upload: doUpload, remove }
}
