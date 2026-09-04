import { useEffect, useState } from 'react'
import { ApiError } from '../../../shared/api/client'
import { getLoteo } from '../api/get-loteo'
import type { LoteoDetail } from '../types'

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

export function useLoteo(loteoId: string, token: string): UseLoteo {
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

  return state
}
