import { useCallback, useEffect, useState } from 'react'
import { ApiError } from '../../../shared/api/client'
import { getLoteo } from '../api/get-loteo'
import type { LoteoCalle, LoteoDetail, LoteoLote, LoteoManzana } from '../types'

const SESSION_EXPIRED_MESSAGE =
  'Tu sesión expiró. Volvé a iniciar sesión y probá de nuevo.'

function isNotFound(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    (error.status === 404 || error.code === 'loteo_not_found')
  )
}

// The api client's plain Error also carries a user-facing message, so don't
// route it through messageFromError (which hides non-ApiError messages).
function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : 'Ocurrió un error inesperado.'
}

export type UseLoteo =
  | { status: 'loading' }
  | { status: 'loaded'; loteo: LoteoDetail }
  | { status: 'not-found' }
  | { status: 'error'; message: string }

function pending(token: string): UseLoteo {
  return token === ''
    ? { status: 'error', message: SESSION_EXPIRED_MESSAGE }
    : { status: 'loading' }
}

export type UseLoteoResult = UseLoteo & {
  replaceLote: (lote: LoteoLote) => void
  replaceManzana: (manzana: LoteoManzana) => void
  replaceCalle: (calle: LoteoCalle) => void
}

export function useLoteo(loteoId: string, token: string): UseLoteoResult {
  const [state, setState] = useState<UseLoteo>(() => pending(token))

  // Reset to the pending state during render when the request key changes, so
  // the UI never shows the previous loteo while the next one loads.
  const requestKey = JSON.stringify([token, loteoId])
  const [loadedKey, setLoadedKey] = useState(requestKey)
  if (requestKey !== loadedKey) {
    setLoadedKey(requestKey)
    setState(pending(token))
  }

  useEffect(() => {
    if (token === '') {
      return
    }

    const controller = new AbortController()

    getLoteo(loteoId, token, { signal: controller.signal })
      .then((loteo) => {
        if (!controller.signal.aborted) {
          setState({ status: 'loaded', loteo })
        }
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) {
          return
        }
        if (isNotFound(error)) {
          setState({ status: 'not-found' })
          return
        }
        setState({ status: 'error', message: messageOf(error) })
      })

    return () => {
      controller.abort()
    }
  }, [loteoId, token])

  const replaceLote = useCallback((lote: LoteoLote) => {
    setState((current) => {
      if (current.status !== 'loaded') {
        return current
      }
      return {
        ...current,
        loteo: {
          ...current.loteo,
          lotes: current.loteo.lotes.map((item) =>
            item.id === lote.id
              ? { ...lote, poligono: lote.poligono.length > 0 ? lote.poligono : item.poligono }
              : item,
          ),
        },
      }
    })
  }, [])

  const replaceManzana = useCallback((manzana: LoteoManzana) => {
    setState((current) => {
      if (current.status !== 'loaded') {
        return current
      }
      return {
        ...current,
        loteo: {
          ...current.loteo,
          manzanas: current.loteo.manzanas.map((item) =>
            item.id === manzana.id
              ? {
                  ...manzana,
                  poligono: manzana.poligono.length > 0 ? manzana.poligono : item.poligono,
                }
              : item,
          ),
        },
      }
    })
  }, [])

  const replaceCalle = useCallback((calle: LoteoCalle) => {
    setState((current) => {
      if (current.status !== 'loaded') {
        return current
      }
      return {
        ...current,
        loteo: {
          ...current.loteo,
          calles: current.loteo.calles.map((item) =>
            item.id === calle.id
              ? {
                  ...calle,
                  poligono: calle.poligono.length > 0 ? calle.poligono : item.poligono,
                }
              : item,
          ),
        },
      }
    })
  }, [])

  return { ...state, replaceLote, replaceManzana, replaceCalle }
}
